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

type Client struct {
	cfg    config.ElasticsearchConfig
	http   *http.Client
	baseURL string
}

func New(cfg config.ElasticsearchConfig) (*Client, error) {
	n := cfg.Normalized()
	if !n.Enabled {
		return nil, fmt.Errorf("elasticsearch disabled")
	}
	if len(n.Addresses) == 0 {
		return nil, fmt.Errorf("elasticsearch addresses required")
	}
	var lastErr error
	for _, addr := range n.Addresses {
		base := strings.TrimRight(strings.TrimSpace(addr), "/")
		if base == "" {
			continue
		}
		c := &Client{
			cfg:     n,
			baseURL: base,
			http: &http.Client{
				Timeout: time.Duration(n.TimeoutSeconds) * time.Second,
			},
		}
		if err := c.Ping(context.Background()); err != nil {
			lastErr = err
			continue
		}
		return c, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("elasticsearch unreachable: %w", lastErr)
	}
	return nil, fmt.Errorf("elasticsearch addresses required")
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("elasticsearch client nil")
	}
	_, status, err := c.doRequest(ctx, http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	if status >= 500 {
		return fmt.Errorf("elasticsearch ping status=%d", status)
	}
	return nil
}

func (c *Client) Search(ctx context.Context, indexPattern string, body map[string]any) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("elasticsearch client nil")
	}
	idx := strings.TrimSpace(indexPattern)
	if idx == "" {
		idx = c.cfg.IndexPattern
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/%s/_search", idx)
	raw, status, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("elasticsearch search failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
