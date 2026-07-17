package k8s

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
)

type K8sHelmService struct {
	runtime  *K8sRuntimeService
	db       *gorm.DB
	cicdBase config.CicdConfig
}

func NewK8sHelmService(runtime *K8sRuntimeService, db *gorm.DB, cicdBase config.CicdConfig) *K8sHelmService {
	return &K8sHelmService{runtime: runtime, db: db, cicdBase: cicdBase}
}

type HelmReleaseItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version,omitempty"`
	Version    int    `json:"version"`
	Status     string `json:"status"`
	Updated    string `json:"updated,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type HelmReleaseHistoryItem struct {
	Revision   int    `json:"revision"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version,omitempty"`
	Updated    string `json:"updated,omitempty"`
	Description string `json:"description,omitempty"`
}

type HelmReleaseValuesResponse struct {
	Values map[string]interface{} `json:"values"`
}

type HelmInstallRequest struct {
	ClusterID       uint                   `json:"cluster_id" binding:"required"`
	Namespace       string                 `json:"namespace" binding:"required"`
	ReleaseName     string                 `json:"release_name" binding:"required"`
	ChartName       string                 `json:"chart_name" binding:"required"`
	ChartVersion    string                 `json:"chart_version"`
	Values          map[string]interface{} `json:"values"`
	CreateNamespace bool                   `json:"create_namespace"`
}

type HelmUpgradeRequest struct {
	ClusterID    uint                   `json:"cluster_id" binding:"required"`
	Namespace    string                 `json:"namespace" binding:"required"`
	ReleaseName  string                 `json:"release_name" binding:"required"`
	ChartName    string                 `json:"chart_name"`
	ChartVersion string                 `json:"chart_version"`
	Values       map[string]interface{} `json:"values"`
}

type HelmRollbackRequest struct {
	ClusterID   uint   `json:"cluster_id" binding:"required"`
	Namespace   string `json:"namespace" binding:"required"`
	ReleaseName string `json:"release_name" binding:"required"`
	Revision    int    `json:"revision" binding:"required"`
}

type HelmReleaseListQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Keyword   string `form:"keyword"`
}

type HelmReleaseNameQuery struct {
	ClusterID   uint   `form:"cluster_id" binding:"required"`
	Namespace   string `form:"namespace" binding:"required"`
	ReleaseName string `form:"release_name" binding:"required"`
}

type HelmChartVersionsQuery struct {
	ChartName string `form:"chart_name" binding:"required"`
}

type HelmChartsListQuery struct {
	Keyword string `form:"keyword"`
}

func (s *K8sHelmService) newActionConfig(ctx context.Context, clusterID uint, namespace string) (*action.Configuration, *cli.EnvSettings, error) {
	_, restCfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return nil, nil, err
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = "default"
	}
	settings := cli.New()
	settings.SetNamespace(ns)
	actionConfig := new(action.Configuration)
	getter := newHelmRESTClientGetter(restCfg)
	if err := actionConfig.Init(getter, ns, "secrets", func(format string, v ...interface{}) {}); err != nil {
		return nil, nil, bizerrors.Internalf(ctx, "helm", "init", err, constants.ErrFmt559cb56d5b9d)
	}
	return actionConfig, settings, nil
}

func (s *K8sHelmService) registryClient(cfg config.HarborConfig) (*registry.Client, error) {
	user, pass, _ := s.harborRegistryLogin(cfg)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 — 内网 Harbor 自签证书
	}
	httpClient := &http.Client{Transport: tr, Timeout: 5 * time.Minute}
	opts := []registry.ClientOption{
		registry.ClientOptEnableCache(true),
		registry.ClientOptHTTPClient(httpClient),
	}
	if user != "" && pass != "" {
		opts = append(opts, registry.ClientOptBasicAuth(user, pass))
	}
	return registry.NewClient(opts...)
}

func (s *K8sHelmService) locateOCIChart(
	actionConfig *action.Configuration,
	settings *cli.EnvSettings,
	cfg config.HarborConfig,
	chartName, chartVersion string,
) (string, *registry.Client, error) {
	ref := harborOCIChartRef(cfg, chartName)
	regClient, err := s.registryClient(cfg)
	if err != nil {
		return "", nil, err
	}
	inst := action.NewInstall(actionConfig)
	inst.SetRegistryClient(regClient)
	inst.Version = strings.TrimSpace(chartVersion)
	chartPath, err := inst.ChartPathOptions.LocateChart(ref, settings)
	if err != nil {
		return "", nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("定位 Chart 失败: %v", err))
	}
	return chartPath, regClient, nil
}

func releaseToItem(rel *release.Release) HelmReleaseItem {
	if rel == nil {
		return HelmReleaseItem{}
	}
	chartName := ""
	appVer := ""
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		chartName = rel.Chart.Metadata.Name
		if rel.Chart.Metadata.Version != "" {
			chartName = chartName + "-" + rel.Chart.Metadata.Version
		}
		appVer = rel.Chart.Metadata.AppVersion
	}
	updated := ""
	if !rel.Info.LastDeployed.IsZero() {
		updated = rel.Info.LastDeployed.Format(time.RFC3339)
	}
	return HelmReleaseItem{
		Name:       rel.Name,
		Namespace:  rel.Namespace,
		Chart:      chartName,
		AppVersion: appVer,
		Version:    rel.Version,
		Status:     string(rel.Info.Status),
		Updated:    updated,
		Notes:      rel.Info.Notes,
	}
}

// ListCharts 列出 Harbor Chart（HTTP query 入口）。
func (s *K8sHelmService) ListCharts(ctx context.Context, q HelmChartsListQuery) ([]HarborChartSummary, error) {
	return s.ListHarborCharts(ctx, q.Keyword)
}

// ChartVersions 列出 Chart 版本（HTTP query 入口）。
func (s *K8sHelmService) ChartVersions(ctx context.Context, q HelmChartVersionsQuery) ([]HarborChartVersion, error) {
	return s.ListHarborChartVersions(ctx, q.ChartName)
}
func (s *K8sHelmService) ListReleases(ctx context.Context, q HelmReleaseListQuery) ([]HelmReleaseItem, error) {
	ns := strings.TrimSpace(q.Namespace)
	actionConfig, _, err := s.newActionConfig(ctx, q.ClusterID, ns)
	if err != nil {
		return nil, err
	}
	list := action.NewList(actionConfig)
	if ns == "" {
		list.AllNamespaces = true
	} else {
		list.SetStateMask()
	}
	releases, err := list.Run()
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "list", err, "列出 Helm Release 失败")
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]HelmReleaseItem, 0, len(releases))
	for _, rel := range releases {
		item := releaseToItem(rel)
		if kw != "" {
			hay := strings.ToLower(item.Name + " " + item.Chart + " " + item.Namespace)
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// GetRelease 获取单个 Release 详情。
func (s *K8sHelmService) GetRelease(ctx context.Context, q HelmReleaseNameQuery) (*HelmReleaseItem, error) {
	actionConfig, _, err := s.newActionConfig(ctx, q.ClusterID, q.Namespace)
	if err != nil {
		return nil, err
	}
	get := action.NewGet(actionConfig)
	rel, err := get.Run(strings.TrimSpace(q.ReleaseName))
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Internalf(ctx, "helm", "get", err, "获取 Release 失败")
	}
	item := releaseToItem(rel)
	return &item, nil
}

// GetReleaseHistory Release 历史版本。
func (s *K8sHelmService) GetReleaseHistory(ctx context.Context, q HelmReleaseNameQuery) ([]HelmReleaseHistoryItem, error) {
	actionConfig, _, err := s.newActionConfig(ctx, q.ClusterID, q.Namespace)
	if err != nil {
		return nil, err
	}
	hist := action.NewHistory(actionConfig)
	hist.Max = 256
	releases, err := hist.Run(strings.TrimSpace(q.ReleaseName))
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "history", err, "获取 Release 历史失败")
	}
	out := make([]HelmReleaseHistoryItem, 0, len(releases))
	for _, rel := range releases {
		chartLabel := ""
		appVer := ""
		if rel.Chart != nil && rel.Chart.Metadata != nil {
			chartLabel = rel.Chart.Metadata.Name + "-" + rel.Chart.Metadata.Version
			appVer = rel.Chart.Metadata.AppVersion
		}
		updated := ""
		if !rel.Info.LastDeployed.IsZero() {
			updated = rel.Info.LastDeployed.Format(time.RFC3339)
		}
		out = append(out, HelmReleaseHistoryItem{
			Revision:    rel.Version,
			Status:      string(rel.Info.Status),
			Chart:       chartLabel,
			AppVersion:  appVer,
			Updated:     updated,
			Description: rel.Info.Description,
		})
	}
	return out, nil
}

// GetReleaseValues 当前 Release 的 values。
func (s *K8sHelmService) GetReleaseValues(ctx context.Context, q HelmReleaseNameQuery) (*HelmReleaseValuesResponse, error) {
	actionConfig, _, err := s.newActionConfig(ctx, q.ClusterID, q.Namespace)
	if err != nil {
		return nil, err
	}
	get := action.NewGetValues(actionConfig)
	vals, err := get.Run(strings.TrimSpace(q.ReleaseName))
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "values", err, "获取 Release Values 失败")
	}
	return &HelmReleaseValuesResponse{Values: vals}, nil
}

// Install 从 Harbor OCI 安装 Chart。
func (s *K8sHelmService) Install(ctx context.Context, req HelmInstallRequest) (*HelmReleaseItem, error) {
	harborCfg, err := s.resolveHarbor(ctx)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	actionConfig, settings, err := s.newActionConfig(ctx, req.ClusterID, ns)
	if err != nil {
		return nil, err
	}
	chartPath, regClient, err := s.locateOCIChart(actionConfig, settings, harborCfg, req.ChartName, req.ChartVersion)
	if err != nil {
		return nil, err
	}
	install := action.NewInstall(actionConfig)
	install.ReleaseName = strings.TrimSpace(req.ReleaseName)
	install.Namespace = ns
	install.CreateNamespace = req.CreateNamespace
	install.Wait = true
	install.Timeout = 5 * time.Minute
	install.SetRegistryClient(regClient)
	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "load_chart", err, "加载 Chart 失败")
	}
	vals := req.Values
	if vals == nil {
		vals = map[string]interface{}{}
	}
	rel, err := install.Run(chartRequested, vals)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "install", err, "Helm 安装失败")
	}
	item := releaseToItem(rel)
	return &item, nil
}

// Upgrade 升级 Release（可指定新 Chart 版本或仅更新 values）。
func (s *K8sHelmService) Upgrade(ctx context.Context, req HelmUpgradeRequest) (*HelmReleaseItem, error) {
	ns := strings.TrimSpace(req.Namespace)
	actionConfig, settings, err := s.newActionConfig(ctx, req.ClusterID, ns)
	if err != nil {
		return nil, err
	}
	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = ns
	upgrade.Wait = true
	upgrade.Timeout = 5 * time.Minute
	// 未传 values 时保留 Release 现有配置，避免升级 Chart 版本时清空业务 values。
	upgrade.ReuseValues = true

	var chartRequested *chart.Chart
	vals := req.Values
	if len(vals) == 0 {
		vals = nil
	}
	chartName := strings.TrimSpace(req.ChartName)
	releaseName := strings.TrimSpace(req.ReleaseName)
	if chartName == "" {
		get := action.NewGet(actionConfig)
		current, gerr := get.Run(releaseName)
		if gerr != nil {
			return nil, bizerrors.Internalf(ctx, "helm", "get_current", gerr, "获取当前 Release 失败")
		}
		if current.Chart != nil && current.Chart.Metadata != nil {
			chartName = current.Chart.Metadata.Name
			if strings.TrimSpace(req.ChartVersion) == "" {
				req.ChartVersion = current.Chart.Metadata.Version
			}
		}
	}
	if chartName != "" {
		harborCfg, herr := s.resolveHarbor(ctx)
		if herr != nil {
			return nil, herr
		}
		chartPath, regClient, lerr := s.locateOCIChart(actionConfig, settings, harborCfg, chartName, req.ChartVersion)
		if lerr != nil {
			return nil, lerr
		}
		upgrade.SetRegistryClient(regClient)
		chartRequested, err = loader.Load(chartPath)
		if err != nil {
			return nil, bizerrors.Internalf(ctx, "helm", "load_chart", err, "加载 Chart 失败")
		}
	}
	rel, err := upgrade.Run(releaseName, chartRequested, vals)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "upgrade", err, "Helm 升级失败")
	}
	item := releaseToItem(rel)
	return &item, nil
}

// Rollback 回滚到指定 revision。
func (s *K8sHelmService) Rollback(ctx context.Context, req HelmRollbackRequest) (*HelmReleaseItem, error) {
	ns := strings.TrimSpace(req.Namespace)
	actionConfig, _, err := s.newActionConfig(ctx, req.ClusterID, ns)
	if err != nil {
		return nil, err
	}
	rollback := action.NewRollback(actionConfig)
	rollback.Version = req.Revision
	rollback.Wait = true
	rollback.Timeout = 5 * time.Minute
	if err := rollback.Run(strings.TrimSpace(req.ReleaseName)); err != nil {
		return nil, bizerrors.Internalf(ctx, "helm", "rollback", err, "Helm 回滚失败")
	}
	q := HelmReleaseNameQuery{ClusterID: req.ClusterID, Namespace: ns, ReleaseName: req.ReleaseName}
	return s.GetRelease(ctx, q)
}

// Uninstall 卸载 Release。
func (s *K8sHelmService) Uninstall(ctx context.Context, q HelmReleaseNameQuery) error {
	actionConfig, _, err := s.newActionConfig(ctx, q.ClusterID, q.Namespace)
	if err != nil {
		return err
	}
	uninstall := action.NewUninstall(actionConfig)
	uninstall.Wait = true
	uninstall.Timeout = 3 * time.Minute
	if _, err := uninstall.Run(strings.TrimSpace(q.ReleaseName)); err != nil {
		return bizerrors.Internalf(ctx, "helm", "uninstall", err, "Helm 卸载失败")
	}
	return nil
}

