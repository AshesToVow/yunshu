package logplatform

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	k8ssvc "yunshu/internal/service/k8s"

	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
)

const defaultClusterLogNamespace = "yunshu-logging"

// ClusterLogService K8s 集群日志采集（DaemonSet + 规则）。
type ClusterLogService struct {
	db          *gorm.DB
	projectRepo interfaces.ProjectRepository
	esProvider  *ElasticsearchProvider
	kafkaProvider *KafkaProvider
	k8sRuntime  *k8ssvc.K8sRuntimeService
	dyn         *k8ssvc.DynamicResourceService
	loggieCfg   config.LoggieConfig
	daemonImage string
}

// NewClusterLogService 创建集群采集服务。
func NewClusterLogService(
	db *gorm.DB,
	projectRepo interfaces.ProjectRepository,
	esProvider *ElasticsearchProvider,
	kafkaProvider *KafkaProvider,
	k8sRuntime *k8ssvc.K8sRuntimeService,
	loggieCfg config.LoggieConfig,
	daemonImage string,
) *ClusterLogService {
	img := strings.TrimSpace(daemonImage)
	if img == "" {
		img = "ghcr.io/loggie-io/loggie:v1.7.1"
	}
	var dyn *k8ssvc.DynamicResourceService
	if k8sRuntime != nil {
		dyn = k8ssvc.NewDynamicResourceService(k8sRuntime)
	}
	return &ClusterLogService{
		db:            db,
		projectRepo:   projectRepo,
		esProvider:    esProvider,
		kafkaProvider: kafkaProvider,
		k8sRuntime:    k8sRuntime,
		dyn:           dyn,
		loggieCfg:     loggieCfg.Normalized(),
		daemonImage:   img,
	}
}

// ClusterLogRuleUpsert 创建/更新规则。
type ClusterLogRuleUpsert struct {
	ClusterID         uint     `json:"cluster_id"`
	Name              string   `json:"name"`
	MatchNamespaces   []string `json:"match_namespaces"`
	MatchWorkloads    []string `json:"match_workloads"`
	ExcludeNamespaces []string `json:"exclude_namespaces"`
	ParseProfile      string   `json:"parse_profile"`
	RateLimitQPS      *int     `json:"rate_limit_qps"`
	Enabled           *bool    `json:"enabled"`
	Remark            string   `json:"remark"`
}

func (s *ClusterLogService) ensureProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("项目 ID 无效")
	}
	if s.projectRepo == nil {
		return nil
	}
	_, err := s.projectRepo.GetByID(ctx, projectID)
	return err
}

func encodeJSONStrings(list []string) string {
	if len(list) == 0 {
		return "[]"
	}
	b, err := json.Marshal(list)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func (s *ClusterLogService) ListRules(ctx context.Context, projectID, clusterID uint) ([]model.ClusterLogRule, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	q := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc")
	if clusterID > 0 {
		q = q.Where("cluster_id = ?", clusterID)
	}
	var list []model.ClusterLogRule
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ClusterLogService) CreateRule(ctx context.Context, projectID uint, req ClusterLogRuleUpsert) (*model.ClusterLogRule, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if req.ClusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("cluster_id 无效")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("规则名不能为空")
	}
	profile := strings.TrimSpace(req.ParseProfile)
	if profile == "" {
		profile = "cri"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	excludes := req.ExcludeNamespaces
	if len(excludes) == 0 {
		excludes = append([]string{}, defaultExcludeNamespaces...)
	}
	row := &model.ClusterLogRule{
		ProjectID:         projectID,
		ClusterID:         req.ClusterID,
		Name:              name,
		MatchNamespaces:   encodeJSONStrings(req.MatchNamespaces),
		MatchWorkloads:    encodeJSONStrings(req.MatchWorkloads),
		ExcludeNamespaces: encodeJSONStrings(excludes),
		ParseProfile:      profile,
		Enabled:           enabled,
		Remark:            strings.TrimSpace(req.Remark),
	}
	if req.RateLimitQPS != nil && *req.RateLimitQPS > 0 {
		row.RateLimitQPS = *req.RateLimitQPS
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *ClusterLogService) UpdateRule(ctx context.Context, projectID, ruleID uint, req ClusterLogRuleUpsert) (*model.ClusterLogRule, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	var row model.ClusterLogRule
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	if req.ClusterID > 0 {
		row.ClusterID = req.ClusterID
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		row.Name = name
	}
	if req.MatchNamespaces != nil {
		row.MatchNamespaces = encodeJSONStrings(req.MatchNamespaces)
	}
	if req.MatchWorkloads != nil {
		row.MatchWorkloads = encodeJSONStrings(req.MatchWorkloads)
	}
	if req.ExcludeNamespaces != nil {
		row.ExcludeNamespaces = encodeJSONStrings(req.ExcludeNamespaces)
	}
	if p := strings.TrimSpace(req.ParseProfile); p != "" {
		row.ParseProfile = p
	}
	if req.RateLimitQPS != nil {
		if *req.RateLimitQPS < 0 {
			return nil, constants.ErrBadRequestWithMsg("rate_limit_qps 无效")
		}
		row.RateLimitQPS = *req.RateLimitQPS
	}
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	row.Remark = strings.TrimSpace(req.Remark)
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *ClusterLogService) DeleteRule(ctx context.Context, projectID, ruleID uint) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).Delete(&model.ClusterLogRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *ClusterLogService) ListAgents(ctx context.Context, projectID uint) ([]model.ClusterLogAgent, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	var list []model.ClusterLogAgent
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ClusterLogService) PreviewPipelines(ctx context.Context, projectID, clusterID uint) (string, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return "", err
	}
	if clusterID == 0 {
		return "", constants.ErrBadRequestWithMsg("cluster_id 无效")
	}
	rules, err := s.ListRules(ctx, projectID, clusterID)
	if err != nil {
		return "", err
	}
	esCfg, kafkaCfg := s.resolveSinkConfigs(ctx)
	rateQPS := defaultClusterLogRateLimitQPS
	var existing model.ClusterLogAgent
	if err := s.db.WithContext(ctx).Where("project_id = ? AND cluster_id = ?", projectID, clusterID).First(&existing).Error; err == nil && existing.RateLimitQPS > 0 {
		rateQPS = existing.RateLimitQPS
	}
	return BuildClusterPipelinesYAML(projectID, clusterID, rules, esCfg, kafkaCfg, rateQPS), nil
}

func (s *ClusterLogService) resolveSinkConfigs(ctx context.Context) (config.ElasticsearchConfig, config.KafkaConfig) {
	var esCfg config.ElasticsearchConfig
	var kafkaCfg config.KafkaConfig
	if s.esProvider != nil {
		if cfg, err := s.esProvider.Resolve(ctx); err == nil {
			esCfg = cfg
		}
	}
	if s.kafkaProvider != nil {
		if cfg, err := s.kafkaProvider.Resolve(ctx); err == nil {
			kafkaCfg = cfg
		}
	}
	return esCfg, kafkaCfg
}

// DeployOrSync 渲染并 Apply DaemonSet + ConfigMap 到目标集群。
func (s *ClusterLogService) DeployOrSync(ctx context.Context, projectID, clusterID uint, namespace string, rateLimitQPS int) (*model.ClusterLogAgent, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if clusterID == 0 {
		return nil, constants.ErrBadRequestWithMsg("cluster_id 无效")
	}
	if s.k8sRuntime == nil || s.dyn == nil {
		return nil, constants.ErrBadRequestWithMsg("K8s Runtime 不可用")
	}
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		ns = defaultClusterLogNamespace
	}
	if rateLimitQPS < 0 {
		return nil, constants.ErrBadRequestWithMsg("rate_limit_qps 无效")
	}

	rules, err := s.ListRules(ctx, projectID, clusterID)
	if err != nil {
		return nil, err
	}
	var existing *model.ClusterLogAgent
	var prev model.ClusterLogAgent
	if err := s.db.WithContext(ctx).Where("project_id = ? AND cluster_id = ?", projectID, clusterID).First(&prev).Error; err == nil {
		existing = &prev
	}
	qps := resolveProjectRateLimitQPS(rateLimitQPS, existing)
	esCfg, kafkaCfg := s.resolveSinkConfigs(ctx)
	pipelines := BuildClusterPipelinesYAML(projectID, clusterID, rules, esCfg, kafkaCfg, qps)
	systemYAML := RenderClusterSystemConfigYAML(projectID, clusterID, 9196)
	manifest := RenderClusterLogManifest(ClusterLogManifestInput{
		ProjectID:     projectID,
		ClusterID:     clusterID,
		Namespace:     ns,
		Image:         s.daemonImage,
		SystemYAML:    systemYAML,
		PipelinesYAML: pipelines,
	})

	_, k, err := s.k8sRuntime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("连接集群失败: " + err.Error())
	}
	// 平台托管 yunshu-logging，跳过命名空间白名单（否则白名单策略会阻断下发）
	applyCtx := k8sauth.WithSkipNamespacePolicy(ctx)
	if err := s.dyn.ApplyManifest(applyCtx, k, manifest, nil); err != nil {
		_, _ = s.upsertAgentStatus(ctx, projectID, clusterID, ns, "failed", err.Error(), 0, 0, qps)
		return nil, constants.ErrBadRequestWithMsg("下发失败: " + err.Error())
	}

	desired, ready := s.readDaemonSetReplicas(ctx, clusterID, ns)
	agent, err := s.upsertAgentStatus(ctx, projectID, clusterID, ns, "deployed", "", desired, ready, qps)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func resolveProjectRateLimitQPS(requested int, existing *model.ClusterLogAgent) int {
	if requested > 0 {
		return requested
	}
	if existing != nil && existing.RateLimitQPS > 0 {
		return existing.RateLimitQPS
	}
	return defaultClusterLogRateLimitQPS
}

func (s *ClusterLogService) RefreshStatus(ctx context.Context, projectID, clusterID uint) (*model.ClusterLogAgent, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	var agent model.ClusterLogAgent
	err := s.db.WithContext(ctx).Where("project_id = ? AND cluster_id = ?", projectID, clusterID).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg("尚未部署集群采集")
		}
		return nil, err
	}
	desired, ready := s.readDaemonSetReplicas(ctx, clusterID, agent.Namespace)
	status := agent.Status
	if desired > 0 && ready >= desired {
		status = "deployed"
	} else if desired > 0 {
		status = "deploying"
	}
	return s.upsertAgentStatus(ctx, projectID, clusterID, agent.Namespace, status, "", desired, ready, agent.RateLimitQPS)
}

func (s *ClusterLogService) readDaemonSetReplicas(ctx context.Context, clusterID uint, namespace string) (desired, ready int) {
	return s.readDaemonSetViaKom(ctx, clusterID, namespace)
}

func (s *ClusterLogService) readDaemonSetViaKom(ctx context.Context, clusterID uint, namespace string) (desired, ready int) {
	_, k, err := s.k8sRuntime.GetClusterKubectl(ctx, clusterID)
	if err != nil || k == nil {
		return 0, 0
	}
	var ds appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(namespace).Name("yunshu-loggie").Get(&ds).Error; err != nil {
		return 0, 0
	}
	return int(ds.Status.DesiredNumberScheduled), int(ds.Status.NumberReady)
}

func (s *ClusterLogService) upsertAgentStatus(
	ctx context.Context,
	projectID, clusterID uint,
	namespace, status, lastErr string,
	desired, ready, rateLimitQPS int,
) (*model.ClusterLogAgent, error) {
	now := time.Now()
	if rateLimitQPS <= 0 {
		rateLimitQPS = defaultClusterLogRateLimitQPS
	}
	var agent model.ClusterLogAgent
	err := s.db.WithContext(ctx).Where("project_id = ? AND cluster_id = ?", projectID, clusterID).First(&agent).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		agent = model.ClusterLogAgent{
			ProjectID:       projectID,
			ClusterID:       clusterID,
			Namespace:       namespace,
			Status:          status,
			DeployRevision:  1,
			DesiredReplicas: desired,
			ReadyReplicas:   ready,
			RateLimitQPS:    rateLimitQPS,
			LastError:       truncateErr(lastErr),
			LastSyncAt:      &now,
		}
		if err := s.db.WithContext(ctx).Create(&agent).Error; err != nil {
			return nil, err
		}
		return &agent, nil
	}
	if err != nil {
		return nil, err
	}
	agent.Namespace = namespace
	agent.Status = status
	agent.DeployRevision++
	agent.DesiredReplicas = desired
	agent.ReadyReplicas = ready
	agent.RateLimitQPS = rateLimitQPS
	agent.LastError = truncateErr(lastErr)
	agent.LastSyncAt = &now
	if err := s.db.WithContext(ctx).Save(&agent).Error; err != nil {
		return nil, err
	}
	return &agent, nil
}

func truncateErr(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}
