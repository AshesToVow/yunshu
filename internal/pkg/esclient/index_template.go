package esclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// PutLegacyIndexTemplate 写入 ES 7 兼容的 _template（index_patterns + mappings）。
func (c *Client) PutLegacyIndexTemplate(ctx context.Context, name string, body map[string]any) error {
	if c == nil {
		return fmt.Errorf("elasticsearch client nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("template name required")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/_template/%s", name)
	raw, status, err := c.doRequest(ctx, http.MethodPut, path, payload)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("put index template failed: status=%d body=%s", status, truncate(string(raw), 512))
	}
	return nil
}
