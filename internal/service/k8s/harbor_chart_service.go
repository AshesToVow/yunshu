package k8s

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

type HarborChartSummary struct {
	Name          string `json:"name"`
	LatestVersion string `json:"latest_version"`
	TotalVersions int    `json:"total_versions"`
	Deprecated    bool   `json:"deprecated"`
}

type HarborChartVersion struct {
	Version    string `json:"version"`
	AppVersion string `json:"app_version,omitempty"`
	Created    string `json:"created,omitempty"`
	Deprecated bool   `json:"deprecated"`
}

type HarborConfigInfo struct {
	URL          string `json:"url"`
	Project      string `json:"project"`
	OCIPrefix    string `json:"oci_prefix"`
	ChartRepoURL string `json:"chart_repo_url"`
	AuthConfigured bool `json:"auth_configured"`
}

type harborChartSummaryItem struct {
	Name          string `json:"name"`
	LatestVersion string `json:"latest_version"`
	TotalVersions int    `json:"total_versions"`
	Deprecated    bool   `json:"deprecated"`
	Created       string `json:"created"`
}

type harborChartVersionItem struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	AppVersion string `json:"app_version"`
	Created    string `json:"created"`
	Deprecated bool   `json:"deprecated"`
}

func (s *K8sHelmService) resolveHarbor(ctx context.Context) (config.HarborConfig, error) {
	cfg := s.cicdBase.Harbor
	if s.db != nil {
		cfg = dictconfig.ResolveCicdConfig(ctx, s.db, s.cicdBase, dictconfig.DefaultCicdDictTypes()).Harbor
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return cfg, constants.ErrBadRequestWithMsg("Harbor 地址未配置，请在数据字典设置 cicd_harbor_url")
	}
	if strings.TrimSpace(cfg.ProjectGroup) == "" {
		cfg.ProjectGroup = "registry"
	}
	return cfg, nil
}

func harborBaseURL(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	return strings.TrimRight(u, "/")
}

func (s *K8sHelmService) harborHTTPClient(serverName string) *http.Client {
	tlsCfg := &tls.Config{InsecureSkipVerify: true} // #nosec G402 — 内网 Harbor 自签证书
	if serverName = strings.TrimSpace(serverName); serverName != "" {
		tlsCfg.ServerName = serverName
	}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
}

func (s *K8sHelmService) harborAuthHeader(cfg config.HarborConfig) string {
	user := strings.TrimSpace(cfg.Username)
	pass := strings.TrimSpace(cfg.Password)
	if user == "" || pass == "" {
		return ""
	}
	token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	return "Basic " + token
}

func harborConnectHost(cfg config.HarborConfig) string {
	if ip := strings.TrimSpace(cfg.HostIP); ip != "" {
		return ip
	}
	return harborBaseURL(cfg.URL)
}

func (s *K8sHelmService) harborRequest(ctx context.Context, cfg config.HarborConfig, method, path string) ([]byte, int, error) {
	host := harborBaseURL(cfg.URL)
	url := fmt.Sprintf("https://%s%s", harborConnectHost(cfg), path)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Host = host
	if auth := s.harborAuthHeader(cfg); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.harborHTTPClient(host).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// GetHarborInfo 返回 Harbor 连接信息（不含密码）。
func (s *K8sHelmService) GetHarborInfo(ctx context.Context) (*HarborConfigInfo, error) {
	cfg, err := s.resolveHarbor(ctx)
	if err != nil {
		return nil, err
	}
	host := harborBaseURL(cfg.URL)
	project := strings.TrimSpace(cfg.ProjectGroup)
	return &HarborConfigInfo{
		URL:            host,
		Project:        project,
		OCIPrefix:      fmt.Sprintf("oci://%s/%s", host, project),
		ChartRepoURL:   fmt.Sprintf("https://%s/chartrepo/%s", host, project),
		AuthConfigured: strings.TrimSpace(cfg.Username) != "" && strings.TrimSpace(cfg.Password) != "",
	}, nil
}

// ListHarborCharts 列出 Harbor 项目中的 Helm Chart（Chart Museum API）。
func (s *K8sHelmService) ListHarborCharts(ctx context.Context, keyword string) ([]HarborChartSummary, error) {
	cfg, err := s.resolveHarbor(ctx)
	if err != nil {
		return nil, err
	}
	project := strings.TrimSpace(cfg.ProjectGroup)
	path := fmt.Sprintf("/api/chartrepo/%s/charts", project)
	body, status, err := s.harborRequest(ctx, cfg, http.MethodGet, path)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法连接 Harbor（%s）: %v", harborConnectHost(cfg), err))
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return nil, constants.ErrBadRequestWithMsg("Harbor 鉴权失败，请配置 cicd_harbor_username / cicd_harbor_password")
	}
	if status >= 400 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Harbor 返回 HTTP %d: %s", status, truncateBody(body, 200)))
	}
	out, err := parseHarborChartListBody(body)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm.harbor", "parse_charts", err, "解析 Harbor Chart 列表失败: %s", truncateBody(body, 200))
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw != "" {
		filtered := out[:0]
		for _, item := range out {
			if strings.Contains(strings.ToLower(item.Name), kw) {
				filtered = append(filtered, item)
			}
		}
		out = filtered
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseHarborChartListBody(body []byte) ([]HarborChartSummary, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed == "null" {
		return []HarborChartSummary{}, nil
	}
	// Harbor 2.x Chart Museum：顶层 JSON 数组
	if strings.HasPrefix(trimmed, "[") {
		var items []harborChartSummaryItem
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, err
		}
		out := make([]HarborChartSummary, 0, len(items))
		for _, item := range items {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				continue
			}
			out = append(out, HarborChartSummary{
				Name:          name,
				LatestVersion: item.LatestVersion,
				TotalVersions: item.TotalVersions,
				Deprecated:    item.Deprecated,
			})
		}
		return out, nil
	}
	// 经典 ChartMuseum：map[chartName][]versions
	var raw map[string][]harborChartVersionItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]HarborChartSummary, 0, len(raw))
	for name, versions := range raw {
		out = append(out, summarizeHarborChart(name, versions))
	}
	return out, nil
}

func summarizeHarborChart(name string, versions []harborChartVersionItem) HarborChartSummary {
	if len(versions) == 0 {
		return HarborChartSummary{Name: name}
	}
	latest := versions[0]
	latestAt := parseHarborChartTime(latest.Created)
	for _, v := range versions[1:] {
		if t := parseHarborChartTime(v.Created); t.After(latestAt) {
			latest = v
			latestAt = t
		}
	}
	return HarborChartSummary{
		Name:          name,
		LatestVersion: latest.Version,
		TotalVersions: len(versions),
		Deprecated:    latest.Deprecated,
	}
}

func parseHarborChartTime(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t
	}
	return time.Time{}
}

// ListHarborChartVersions 列出指定 Chart 的版本。
func (s *K8sHelmService) ListHarborChartVersions(ctx context.Context, chartName string) ([]HarborChartVersion, error) {
	chartName = strings.TrimSpace(chartName)
	if chartName == "" {
		return nil, constants.ErrBadRequestWithMsg("chart 名称不能为空")
	}
	cfg, err := s.resolveHarbor(ctx)
	if err != nil {
		return nil, err
	}
	project := strings.TrimSpace(cfg.ProjectGroup)
	path := fmt.Sprintf("/api/chartrepo/%s/charts/%s", project, chartName)
	body, status, err := s.harborRequest(ctx, cfg, http.MethodGet, path)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法连接 Harbor（%s）: %v", harborConnectHost(cfg), err))
	}
	if status == http.StatusNotFound {
		return []HarborChartVersion{}, nil
	}
	if status >= 400 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Harbor 返回 HTTP %d", status))
	}
	var raw []harborChartVersionItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, bizerrors.Internalf(ctx, "helm.harbor", "parse_versions", err, "解析 Chart 版本失败")
	}
	out := make([]HarborChartVersion, 0, len(raw))
	for _, v := range raw {
		out = append(out, HarborChartVersion{
			Version:    v.Version,
			AppVersion: v.AppVersion,
			Created:    v.Created,
			Deprecated: v.Deprecated,
		})
	}
	return out, nil
}

func truncateBody(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// harborOCIChartRef 生成 OCI Chart 引用，供 helm install/upgrade 使用。
func harborOCIChartRef(cfg config.HarborConfig, chartName string) string {
	return fmt.Sprintf("oci://%s/%s/%s", harborBaseURL(cfg.URL), strings.TrimSpace(cfg.ProjectGroup), strings.TrimSpace(chartName))
}

// ensureHarborRegistryLogin 为 OCI pull 配置 registry 凭据（Helm registry client）。
func (s *K8sHelmService) harborRegistryLogin(cfg config.HarborConfig) (username, password string, insecure bool) {
	return strings.TrimSpace(cfg.Username), strings.TrimSpace(cfg.Password), true
}
