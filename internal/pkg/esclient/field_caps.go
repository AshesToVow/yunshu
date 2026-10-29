package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// FieldCaps 查询字段在索引中的真实类型（跨索引合并），聚合前用它区分
// keyword / text / 数值，避免对 text 字段做 terms 导致整个请求 400。
func (c *Client) FieldCaps(ctx context.Context, indexPattern string, fields []string) (map[string][]string, error) {
	if c == nil {
		return nil, fmt.Errorf("elasticsearch client nil")
	}
	idx := strings.TrimSpace(indexPattern)
	if idx == "" {
		idx = c.cfg.IndexPattern
	}
	names := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		names = append(names, f)
	}
	if len(names) == 0 {
		return map[string][]string{}, nil
	}
	// 字段名来自固定候选列表（字母/数字/点/下划线），无需再转义
	path := fmt.Sprintf("/%s/_field_caps?fields=%s&ignore_unavailable=true&allow_no_indices=true",
		idx, strings.Join(names, ","))
	raw, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("elasticsearch field_caps failed: status=%d body=%s", status, truncate(string(raw), 256))
	}
	var resp struct {
		Fields map[string]map[string]struct {
			Type string `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	out := make(map[string][]string, len(resp.Fields))
	for field, types := range resp.Fields {
		list := make([]string, 0, len(types))
		for name, meta := range types {
			if meta.Type != "" {
				name = meta.Type
			}
			list = append(list, name)
		}
		out[field] = list
	}
	return out, nil
}
