package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// GetIndexSettings 获取索引 settings（含 analysis/分词器）。
func (c *Client) GetIndexSettings(ctx context.Context, index string) (map[string]any, error) {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return nil, fmt.Errorf("index required")
	}
	raw, status, err := c.doRequest(ctx, http.MethodGet, "/"+index+"/_settings?flat_settings=false&include_defaults=false", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("get settings failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetIndexMapping 获取索引 mapping。
func (c *Client) GetIndexMapping(ctx context.Context, index string) (map[string]any, error) {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return nil, fmt.Errorf("index required")
	}
	raw, status, err := c.doRequest(ctx, http.MethodGet, "/"+index+"/_mapping", nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("get mapping failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ScrollHit 单条文档。
type ScrollHit struct {
	ID     string
	Source map[string]any
}

// ScrollAll 滚动导出文档（最多 maxDocs；maxDocs<=0 表示默认 50000）。
func (c *Client) ScrollAll(ctx context.Context, index string, maxDocs int) ([]ScrollHit, error) {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return nil, fmt.Errorf("index required")
	}
	if maxDocs <= 0 {
		maxDocs = 50000
	}
	if maxDocs > 200000 {
		maxDocs = 200000
	}
	body := map[string]any{
		"size":    500,
		"query":   map[string]any{"match_all": map[string]any{}},
		"sort":    []string{"_doc"},
		"_source": true,
	}
	payload, _ := json.Marshal(body)
	path := fmt.Sprintf("/%s/_search?scroll=2m", index)
	raw, status, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("scroll search failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	var first scrollResp
	if err := json.Unmarshal(raw, &first); err != nil {
		return nil, err
	}
	out := make([]ScrollHit, 0, len(first.Hits.Hits))
	appendHits := func(hits []scrollHitItem) {
		for _, h := range hits {
			out = append(out, ScrollHit{ID: h.ID, Source: h.Source})
		}
	}
	appendHits(first.Hits.Hits)
	scrollID := first.ScrollID
	defer func() {
		if scrollID == "" {
			return
		}
		clearBody, _ := json.Marshal(map[string]any{"scroll_id": []string{scrollID}})
		_, _, _ = c.doRequest(ctx, http.MethodDelete, "/_search/scroll", clearBody)
	}()

	for len(first.Hits.Hits) > 0 && len(out) < maxDocs {
		nextBody, _ := json.Marshal(map[string]any{"scroll": "2m", "scroll_id": scrollID})
		raw, status, err = c.doRequest(ctx, http.MethodPost, "/_search/scroll", nextBody)
		if err != nil {
			return out, err
		}
		if status >= 300 {
			return out, fmt.Errorf("scroll continue failed: status=%d", status)
		}
		var next scrollResp
		if err := json.Unmarshal(raw, &next); err != nil {
			return out, err
		}
		if next.ScrollID != "" {
			scrollID = next.ScrollID
		}
		if len(next.Hits.Hits) == 0 {
			break
		}
		remain := maxDocs - len(out)
		if len(next.Hits.Hits) > remain {
			next.Hits.Hits = next.Hits.Hits[:remain]
		}
		appendHits(next.Hits.Hits)
		first = next
	}
	return out, nil
}

type scrollResp struct {
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Hits []scrollHitItem `json:"hits"`
	} `json:"hits"`
}

type scrollHitItem struct {
	ID     string         `json:"_id"`
	Source map[string]any `json:"_source"`
}
