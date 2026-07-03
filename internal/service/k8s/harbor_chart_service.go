package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type harborChartListItem struct {
	Name          string `json:"name"`
	LatestVersion string `json:"latest_version"`
	TotalVersions int    `json:"total_versions"`
	Deprecated    bool   `json:"deprecated"`
}

type harborChartVersionItem struct {
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

func (s *K8sHelmService) harborHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
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

func (s *K8sHelmService) harborRequest(ctx context.Context, cfg config.HarborConfig, method, path string) ([]byte, int, error) {
	url := fmt.Sprintf("https://%s%s", harborBaseURL(cfg.URL), path)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if auth := s.harborAuthHeader(cfg); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.harborHTTPClient().Do(req)
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
		return nil, bizerrors.Internalf(ctx, "helm.harbor", "list_charts", err, "拉取 Harbor Chart 列表失败")
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return nil, constants.ErrBadRequestWithMsg("Harbor 鉴权失败，请配置 cicd_harbor_username / cicd_harbor_password")
	}
	if status >= 400 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("Harbor 返回 HTTP %d: %s", status, truncateBody(body, 200)))
	}
	var raw map[string]harborChartListItem
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, bizerrors.Internalf(ctx, "helm.harbor", "parse_charts", err, "解析 Harbor Chart 列表失败")
	}
	kw := strings.ToLower(strings.TrimSpace(keyword))
	out := make([]HarborChartSummary, 0, len(raw))
	for name, item := range raw {
		if kw != "" && !strings.Contains(strings.ToLower(name), kw) {
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
		return nil, bizerrors.Internalf(ctx, "helm.harbor", "list_versions", err, "拉取 Chart 版本失败")
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
