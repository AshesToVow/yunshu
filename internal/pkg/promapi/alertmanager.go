package promapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
)

// DeriveAlertmanagerURL 从 Prometheus base_url 推导 Alertmanager 地址（常见 9090→9093）。
// 若无法推导则返回空字符串。
func DeriveAlertmanagerURL(promBaseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(promBaseURL), "/")
	if base == "" {
		return ""
	}
	u, err := neturl.Parse(base)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	port := u.Port()
	switch port {
	case "9090":
		u.Host = net.JoinHostPort(host, "9093")
	case "9093":
		// 已是 Alertmanager 端口
	default:
		if port == "" {
			u.Host = net.JoinHostPort(host, "9093")
		} else {
			// 非标准端口时不猜测，由数据源显式配置 alertmanager_url
			return ""
		}
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/")
}

// AlertmanagerSilences GET /api/v2/silences
func (c *Client) AlertmanagerSilences(ctx context.Context) (json.RawMessage, int, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("alertmanager base_url is empty")
	}
	u := base + "/api/v2/silences"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	c.authHeader(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("alertmanager status %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}
