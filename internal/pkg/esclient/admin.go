package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunshu/internal/config"
)

func (c *Client) IndexDoc(ctx context.Context, index, id string, doc map[string]any) error {
	if c == nil {
		return fmt.Errorf("elasticsearch client nil")
	}
	index = strings.Trim(strings.TrimSpace(index), "/")
	id = strings.TrimSpace(id)
	if index == "" || id == "" {
		return fmt.Errorf("index and id required")
	}
	payload, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/%s/_doc/%s", index, id)
	raw, status, err := c.doRequest(ctx, http.MethodPut, path, payload)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("index doc failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	return nil
}

func (c *Client) ClusterHealth(ctx context.Context) (map[string]any, error) {
	raw, status, err := c.doRequest(ctx, http.MethodGet, "/_cluster/health", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("cluster health failed: status=%d", status)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CatNodes(ctx context.Context) ([]map[string]any, error) {
	raw, status, err := c.doRequest(ctx, http.MethodGet, "/_cat/nodes?format=json&h=name,ip,heap.percent,ram.percent,cpu,load_1m,node.role,master", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("cat nodes failed: status=%d", status)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CatShards(ctx context.Context, indexPattern string, limit int) ([]map[string]any, error) {
	path := "/_cat/shards?format=json"
	if p := strings.TrimSpace(indexPattern); p != "" && p != "*" {
		path = fmt.Sprintf("/_cat/shards/%s?format=json", p)
	}
	raw, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("cat shards failed: status=%d", status)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (c *Client) OpenIndex(ctx context.Context, index string) error {
	return c.indexAction(ctx, index, "_open")
}

func (c *Client) CloseIndex(ctx context.Context, index string) error {
	return c.indexAction(ctx, index, "_close")
}

func (c *Client) indexAction(ctx context.Context, index, action string) error {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return fmt.Errorf("index required")
	}
	path := fmt.Sprintf("/%s/%s", index, action)
	raw, status, err := c.doRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("%s failed: status=%d body=%s", action, status, truncate(string(raw), 300))
	}
	return nil
}

// ProxyResult 受限 REST 代理结果。
type ProxyResult struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// ProxyREST 仅允许安全探查类路径。
func (c *Client) ProxyREST(ctx context.Context, method, path string, body []byte) (*ProxyResult, error) {
	if c == nil {
		return nil, fmt.Errorf("elasticsearch client nil")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	path = "/" + strings.TrimPrefix(strings.TrimSpace(path), "/")
	if !proxyPathAllowed(method, path) {
		return nil, fmt.Errorf("路径或方法不允许: %s %s", method, path)
	}
	raw, status, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	return &ProxyResult{Status: status, Body: json.RawMessage(raw)}, nil
}

func proxyPathAllowed(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodHead:
	default:
		return false
	}
	p := strings.ToLower(path)
	for _, pref := range []string{"/_cluster", "/_cat", "/_nodes", "/_stats", "/_aliases"} {
		if strings.HasPrefix(p, pref) {
			return true
		}
	}
	if strings.Contains(p, "/_search") || strings.HasSuffix(p, "/_mapping") || strings.HasSuffix(p, "/_settings") {
		if method == http.MethodGet || method == http.MethodPost {
			return !strings.Contains(p, "_scripts") && !strings.Contains(p, "painless")
		}
	}
	if p == "/" {
		return method == http.MethodGet || method == http.MethodHead
	}
	return false
}

// NewUnmanaged 创建客户端（跳过 Ping），用于已校验的连接配置。
func NewUnmanaged(cfg config.ElasticsearchConfig) (*Client, error) {
	n := cfg.Normalized()
	base := ""
	for _, addr := range n.Addresses {
		addr = strings.TrimRight(strings.TrimSpace(addr), "/")
		if addr != "" {
			base = addr
			break
		}
	}
	if base == "" {
		return nil, fmt.Errorf("elasticsearch addresses required")
	}
	timeout := n.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}
	return &Client{
		cfg:     n,
		baseURL: base,
		http:    &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}, nil
}
