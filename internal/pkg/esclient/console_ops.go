package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SearchRaw 对指定索引执行 _search，返回原始 JSON。
func (c *Client) SearchRaw(ctx context.Context, index string, body []byte) ([]byte, error) {
	return c.jsonOK(ctx, http.MethodPost, "/"+index+"/_search", body)
}

// GetDoc 读取单条文档 _source。
func (c *Client) GetDoc(ctx context.Context, index, id string) (json.RawMessage, bool, error) {
	raw, status, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/%s/_doc/%s", index, url.PathEscape(id)), nil)
	if err != nil {
		return nil, false, err
	}
	if status == http.StatusNotFound {
		return nil, false, nil
	}
	if status >= 300 {
		return nil, false, fmt.Errorf("get doc failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	var wrap struct {
		Found  bool            `json:"found"`
		Source json.RawMessage `json:"_source"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, false, err
	}
	if !wrap.Found {
		return nil, false, nil
	}
	if len(wrap.Source) == 0 {
		wrap.Source = json.RawMessage("null")
	}
	return wrap.Source, true, nil
}

// PutDoc 写入或覆盖文档（refresh=wait_for）。
func (c *Client) PutDoc(ctx context.Context, index, id string, body []byte) error {
	_, err := c.jsonOK(ctx, http.MethodPut, fmt.Sprintf("/%s/_doc/%s?refresh=wait_for", index, url.PathEscape(id)), body)
	return err
}

// DeleteDoc 删除文档。不存在视为成功。
func (c *Client) DeleteDoc(ctx context.Context, index, id string) error {
	raw, status, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/%s/_doc/%s?refresh=wait_for", index, url.PathEscape(id)), nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status >= 300 {
		return fmt.Errorf("delete doc failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	return nil
}

func (c *Client) jsonOK(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("elasticsearch client nil")
	}
	raw, status, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("elasticsearch %s %s failed: status=%d body=%s", method, path, status, truncate(string(raw), 512))
	}
	return raw, nil
}

func (c *Client) ListLegacyTemplates(ctx context.Context) (map[string]json.RawMessage, error) {
	raw, err := c.jsonOK(ctx, http.MethodGet, "/_template", nil)
	if err != nil {
		return nil, err
	}
	out := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetLegacyTemplate(ctx context.Context, name string) (json.RawMessage, error) {
	raw, err := c.jsonOK(ctx, http.MethodGet, "/_template/"+name, nil)
	if err != nil {
		return nil, err
	}
	var wrap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	body, ok := wrap[name]
	if !ok {
		for _, v := range wrap {
			return v, nil
		}
		return nil, fmt.Errorf("template %s not found", name)
	}
	return body, nil
}

func (c *Client) PutLegacyTemplate(ctx context.Context, name string, body []byte) error {
	_, err := c.jsonOK(ctx, http.MethodPut, "/_template/"+name, body)
	return err
}

func (c *Client) DeleteLegacyTemplate(ctx context.Context, name string) error {
	raw, status, err := c.doRequest(ctx, http.MethodDelete, "/_template/"+name, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status >= 300 {
		return fmt.Errorf("delete template failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	return nil
}

func (c *Client) ListComposableTemplates(ctx context.Context) ([]ComposableTemplate, error) {
	raw, status, err := c.doRequest(ctx, http.MethodGet, "/_index_template", nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status >= 300 {
		msg := strings.ToLower(string(raw))
		if status == http.StatusBadRequest && strings.Contains(msg, "index_template") {
			return nil, nil
		}
		return nil, fmt.Errorf("list index templates failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	var wrap struct {
		IndexTemplates []struct {
			Name          string          `json:"name"`
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	out := make([]ComposableTemplate, 0, len(wrap.IndexTemplates))
	for _, it := range wrap.IndexTemplates {
		out = append(out, ComposableTemplate{Name: it.Name, Body: it.IndexTemplate})
	}
	return out, nil
}

// ComposableTemplate ES 7.8+ _index_template。
type ComposableTemplate struct {
	Name string
	Body json.RawMessage
}

func (c *Client) GetComposableTemplate(ctx context.Context, name string) (json.RawMessage, error) {
	raw, err := c.jsonOK(ctx, http.MethodGet, "/_index_template/"+name, nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		IndexTemplates []struct {
			IndexTemplate json.RawMessage `json:"index_template"`
		} `json:"index_templates"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.IndexTemplates) == 0 {
		return nil, fmt.Errorf("index template %s not found", name)
	}
	return wrap.IndexTemplates[0].IndexTemplate, nil
}

func (c *Client) PutComposableTemplate(ctx context.Context, name string, body []byte) error {
	_, err := c.jsonOK(ctx, http.MethodPut, "/_index_template/"+name, body)
	return err
}

func (c *Client) DeleteComposableTemplate(ctx context.Context, name string) error {
	raw, status, err := c.doRequest(ctx, http.MethodDelete, "/_index_template/"+name, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status >= 300 {
		return fmt.Errorf("delete index template failed: status=%d body=%s", status, truncate(string(raw), 300))
	}
	return nil
}

// StartReindex 异步 reindex，返回 ES task id。
func (c *Client) StartReindex(ctx context.Context, body []byte) (string, error) {
	raw, err := c.jsonOK(ctx, http.MethodPost, "/_reindex?wait_for_completion=false", body)
	if err != nil {
		return "", err
	}
	var wrap struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return "", err
	}
	if strings.TrimSpace(wrap.Task) == "" {
		return "", fmt.Errorf("reindex did not return task id")
	}
	return wrap.Task, nil
}

// GetTask 查询 _tasks/{id}。
func (c *Client) GetTask(ctx context.Context, taskID string) ([]byte, error) {
	return c.jsonOK(ctx, http.MethodGet, "/_tasks/"+taskID, nil)
}

// CancelTask 取消任务。
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	_, err := c.jsonOK(ctx, http.MethodPost, "/_tasks/"+taskID+"/_cancel", nil)
	return err
}
