package consulclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

// Client 轻量 Consul HTTP API（catalog），无第三方依赖。
type Client struct {
	Address    string
	Token      string
	Datacenter string
	HTTPClient *http.Client
}

type CatalogService struct {
	ID                       string
	Node                     string
	Address                  string
	ServiceAddress           string
	ServicePort              int
	ServiceName              string
	ServiceID                string
	ServiceTags              []string
	ServiceMeta              map[string]string
	ServiceTaggedAddresses   map[string]any
	Checks                   []CatalogCheck
}

type CatalogCheck struct {
	Status string
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) base() string {
	return strings.TrimRight(strings.TrimSpace(c.Address), "/")
}

func (c *Client) doJSON(ctx context.Context, path string, out any) error {
	base := c.base()
	if base == "" {
		return fmt.Errorf("consul address is empty")
	}
	u, err := neturl.Parse(base + path)
	if err != nil {
		return err
	}
	q := u.Query()
	if dc := strings.TrimSpace(c.Datacenter); dc != "" {
		q.Set("dc", dc)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if t := strings.TrimSpace(c.Token); t != "" {
		req.Header.Set("X-Consul-Token", t)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consul status %d: %s", resp.StatusCode, truncate(string(body), 256))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// ListServiceNames GET /v1/catalog/services
func (c *Client) ListServiceNames(ctx context.Context) (map[string][]string, error) {
	var m map[string][]string
	if err := c.doJSON(ctx, "/v1/catalog/services", &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ListServiceInstances GET /v1/health/service/:name?pass=false 含检查状态
func (c *Client) ListServiceInstances(ctx context.Context, service string) ([]CatalogService, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, fmt.Errorf("service name empty")
	}
	path := "/v1/health/service/" + neturl.PathEscape(service)
	var raw []struct {
		Node struct {
			Node    string `json:"Node"`
			Address string `json:"Address"`
		} `json:"Node"`
		Service struct {
			ID      string            `json:"ID"`
			Service string            `json:"Service"`
			Tags    []string          `json:"Tags"`
			Address string            `json:"Address"`
			Port    int               `json:"Port"`
			Meta    map[string]string `json:"Meta"`
		} `json:"Service"`
		Checks []struct {
			Status string `json:"Status"`
		} `json:"Checks"`
	}
	if err := c.doJSON(ctx, path, &raw); err != nil {
		return nil, err
	}
	out := make([]CatalogService, 0, len(raw))
	for _, row := range raw {
		checks := make([]CatalogCheck, 0, len(row.Checks))
		for _, ch := range row.Checks {
			checks = append(checks, CatalogCheck{Status: ch.Status})
		}
		addr := strings.TrimSpace(row.Service.Address)
		if addr == "" {
			addr = strings.TrimSpace(row.Node.Address)
		}
		out = append(out, CatalogService{
			Node:           row.Node.Node,
			Address:        addr,
			ServiceAddress: row.Service.Address,
			ServicePort:    row.Service.Port,
			ServiceName:    row.Service.Service,
			ServiceID:      row.Service.ID,
			ServiceTags:    row.Service.Tags,
			ServiceMeta:    row.Service.Meta,
			Checks:         checks,
		})
	}
	return out, nil
}

// Ping GET /v1/status/leader
func (c *Client) Ping(ctx context.Context) error {
	var leader string
	if err := c.doJSON(ctx, "/v1/status/leader", &leader); err != nil {
		return err
	}
	if strings.TrimSpace(leader) == "" {
		return fmt.Errorf("consul leader empty")
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// AggregateHealth 取最差检查状态。
func AggregateHealth(checks []CatalogCheck) string {
	worst := "passing"
	for _, c := range checks {
		st := strings.ToLower(strings.TrimSpace(c.Status))
		switch st {
		case "critical":
			return "critical"
		case "warning":
			worst = "warning"
		case "passing", "":
		default:
			if worst == "passing" {
				worst = st
			}
		}
	}
	return worst
}

// HasTag 判断 tags 是否包含指定 tag（忽略大小写）。
func HasTag(tags []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	if want == "" {
		return true
	}
	for _, t := range tags {
		if strings.ToLower(strings.TrimSpace(t)) == want {
			return true
		}
	}
	return false
}
