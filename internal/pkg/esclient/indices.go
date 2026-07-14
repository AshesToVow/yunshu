package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type IndexInfo struct {
	Name       string
	StoreBytes int64
	DocsCount  int64
}

var indexDateSuffix = regexp.MustCompile(`(\d{4})[.\-](\d{2})[.\-](\d{2})$`)

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	url := c.baseURL + path
	var reader io.Reader
	if len(body) > 0 {
		reader = strings.NewReader(string(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	return raw, resp.StatusCode, err
}

func (c *Client) CatIndices(ctx context.Context, indexPattern string) ([]IndexInfo, error) {
	pattern := strings.TrimSpace(indexPattern)
	if pattern == "" {
		pattern = c.cfg.IndexPattern
	}
	path := fmt.Sprintf("/_cat/indices/%s?format=json&bytes=b", pattern)
	raw, status, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("cat indices failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]IndexInfo, 0, len(rows))
	for _, row := range rows {
		name, _ := row["index"].(string)
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, IndexInfo{
			Name:       name,
			StoreBytes: parseInt64Any(row["store.size"]),
			DocsCount:  parseInt64Any(row["docs.count"]),
		})
	}
	return out, nil
}

func (c *Client) DeleteIndex(ctx context.Context, index string) error {
	path := "/" + strings.TrimPrefix(strings.TrimSpace(index), "/")
	raw, status, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status >= 300 && status != 404 {
		return fmt.Errorf("delete index failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	return nil
}

func (c *Client) DeleteByQuery(ctx context.Context, indexPattern string, body map[string]any) (int64, error) {
	pattern := strings.TrimSpace(indexPattern)
	if pattern == "" {
		pattern = c.cfg.IndexPattern
	}
	path := fmt.Sprintf("/%s/_delete_by_query?conflicts=proceed", pattern)
	payload, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	raw, status, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return 0, err
	}
	if status >= 300 {
		return 0, fmt.Errorf("delete_by_query failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, err
	}
	return parseInt64Any(out["deleted"]), nil
}

// ParseIndexDate 从索引名尾部解析日期（支持 yunshu-logs-2026.07.13）。
func ParseIndexDate(indexName string) (time.Time, bool) {
	m := indexDateSuffix.FindStringSubmatch(strings.TrimSpace(indexName))
	if len(m) != 4 {
		return time.Time{}, false
	}
	y, _ := strconv.Atoi(m[1])
	mo, _ := strconv.Atoi(m[2])
	d, _ := strconv.Atoi(m[3])
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if t.Year() != y || int(t.Month()) != mo || t.Day() != d {
		return time.Time{}, false
	}
	return t, true
}

func parseInt64Any(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}
