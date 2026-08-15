package esclient

import (
	"bytes"
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

// ProxyREST 代理 ES REST（管理员控制台）。允许 GET/POST/PUT/DELETE/HEAD；
// 禁止脚本执行、节点关机等高危路径。
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
	return &ProxyResult{Status: status, Body: encodeProxyBody(raw)}, nil
}

// encodeProxyBody 保证响应体可被 JSON 序列化。
// _cat?v 等接口返回纯文本，直接塞进 json.RawMessage 会导致外层 Success 编码失败 (500)。
func encodeProxyBody(raw []byte) json.RawMessage {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return json.RawMessage("null")
	}
	if json.Valid(raw) {
		return json.RawMessage(raw)
	}
	quoted, err := json.Marshal(string(raw))
	if err != nil {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(quoted)
}

func proxyPathAllowed(method, path string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodHead:
	default:
		return false
	}
	full := strings.ToLower(path)
	if proxyPathBlocked(full) {
		return false
	}
	p := full
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	if p == "/" {
		return method == http.MethodGet || method == http.MethodHead
	}
	for _, pref := range []string{
		"/_cluster", "/_cat", "/_nodes", "/_stats", "/_aliases", "/_alias",
		"/_tasks", "/_ingest", "/_template", "/_index_template", "/_component_template",
		"/_ilm", "/_data_stream", "/_resolve",
	} {
		if strings.HasPrefix(p, pref) {
			return true
		}
	}
	for _, token := range []string{
		"/_search", "/_msearch", "/_count", "/_explain", "/_validate",
		"/_mapping", "/_settings", "/_aliases", "/_alias",
		"/_doc", "/_create", "/_update", "/_source", "/_bulk", "/_mget",
		"/_delete_by_query", "/_update_by_query", "/_reindex",
		"/_refresh", "/_flush", "/_forcemerge", "/_open", "/_close",
		"/_shrink", "/_split", "/_clone", "/_rollover",
	} {
		if strings.Contains(p, token) {
			return true
		}
	}
	// 索引资源：/my-index、/a,b 等（首段不以 _ 开头）
	return isIndexResourcePath(p)
}

func proxyPathBlocked(p string) bool {
	if strings.Contains(p, "_scripts") || strings.Contains(p, "painless") {
		return true
	}
	if strings.Contains(p, "shutdown") {
		return true
	}
	return false
}

func isIndexResourcePath(p string) bool {
	p = strings.Trim(p, "/")
	if p == "" {
		return false
	}
	first := p
	if i := strings.IndexByte(p, '/'); i >= 0 {
		first = p[:i]
	}
	if first == "" || strings.HasPrefix(first, "_") {
		return false
	}
	return true
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
