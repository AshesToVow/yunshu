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
	Name       string `json:"name"`
	StoreBytes int64  `json:"store_bytes"`
	DocsCount  int64  `json:"docs_count"`
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
	path := "/_cat/indices?format=json&bytes=b"
	if pattern != "" && pattern != "*" {
		path = fmt.Sprintf("/_cat/indices/%s?format=json&bytes=b", pattern)
	}
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

// BulkResult 描述一次 _bulk 请求的结果。区分「传输级失败」（返回 error，调用方应重试）
// 与「单条文档失败」（Failed>0，好文档已成功写入，坏文档重试也无用，调用方应记录并跳过）。
type BulkResult struct {
	Failed     int    // 被 ES 拒绝的文档数（映射冲突等）
	FirstError string // 首条失败原因样本，便于定位
}

func (c *Client) Bulk(ctx context.Context, ndjson []byte) (*BulkResult, error) {
	if c == nil {
		return nil, fmt.Errorf("elasticsearch client nil")
	}
	if len(ndjson) == 0 {
		return &BulkResult{}, nil
	}
	raw, status, err := c.doRequest(ctx, http.MethodPost, "/_bulk", ndjson)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("elasticsearch bulk failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	var out struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int             `json:"status"`
			Error  json.RawMessage `json:"error"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		// 响应无法解析：请求已成功（status<300），视为已写入，避免误重试导致重复。
		return &BulkResult{}, nil
	}
	res := &BulkResult{}
	if !out.Errors {
		return res, nil
	}
	for _, item := range out.Items {
		for _, action := range item {
			if action.Status >= 300 || len(action.Error) > 0 {
				res.Failed++
				if res.FirstError == "" && len(action.Error) > 0 {
					res.FirstError = truncate(string(action.Error), 300)
				}
			}
		}
	}
	return res, nil
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

// IndexExists 判断索引是否存在。
func (c *Client) IndexExists(ctx context.Context, index string) (bool, error) {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return false, fmt.Errorf("index required")
	}
	_, status, err := c.doRequest(ctx, http.MethodHead, "/"+index, nil)
	if err != nil {
		return false, err
	}
	if status == http.StatusOK {
		return true, nil
	}
	if status == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("head index failed: status=%d", status)
}

// CreateIndex 创建索引（body 含 settings/mappings）。
func (c *Client) CreateIndex(ctx context.Context, index string, body map[string]any) error {
	index = strings.Trim(strings.TrimSpace(index), "/")
	if index == "" {
		return fmt.Errorf("index required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	raw, status, err := c.doRequest(ctx, http.MethodPut, "/"+index, payload)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("create index failed: status=%d body=%s", status, truncate(string(raw), 400))
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

// ParseIndexDate 从索引名尾部解析日期（支持 yunshu-agent-7-2026.07.13 / yunshu-logs-2026.07.13）。
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
