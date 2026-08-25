package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

// ProgressivePromoteRequest 金丝雀晋级 / 蓝绿切换。
type ProgressivePromoteRequest struct {
	ClusterID   uint   `json:"cluster_id"`
	Namespace   string `json:"namespace"`
	Workload    string `json:"workload"` // 稳定版 Deployment 名
	ServiceName string `json:"service_name"`
	// TargetPercent 金丝雀目标流量占比（相对总副本）；0 表示按步骤自动取下一步
	TargetPercent int `json:"target_percent"`
	// Final 金丝雀最终晋级：新镜像写入稳定版并缩掉 canary
	Final bool `json:"final"`
}

// ProgressiveAbortRequest 中止金丝雀/蓝绿。
type ProgressiveAbortRequest struct {
	ClusterID   uint   `json:"cluster_id"`
	Namespace   string `json:"namespace"`
	Workload    string `json:"workload"`
	ServiceName string `json:"service_name"`
}

type progressiveState struct {
	Strategy      string    `json:"strategy"`
	StepIndex     int       `json:"step_index"`
	CurrentPercent int      `json:"current_percent"`
	Steps         []int     `json:"steps"`
	ActiveColor   string    `json:"active_color,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastAction    string    `json:"last_action"`
	Detail        any       `json:"detail,omitempty"`
}

// K8sProgressiveFns 平台侧渐进式发布能力（由路由层注入）。
type K8sProgressiveFns struct {
	EnsureCanary func(ctx context.Context, clusterID uint, ns, stable, canary, image string, replicas int32) (map[string]any, error)
	Scale        func(ctx context.Context, clusterID uint, ns, name string, replicas int32) error
	PatchImage   func(ctx context.Context, clusterID uint, ns, name, image string) (map[string]any, error)
	SwitchColor  func(ctx context.Context, clusterID uint, ns, svc, color string) (map[string]any, error)
	EnsureColor  func(ctx context.Context, clusterID uint, ns, base, color, image string, replicas int32) (map[string]any, error)
}

func (s *Service) SetK8sProgressive(fns K8sProgressiveFns) {
	s.k8sProgressive = fns
}

// PromoteProgressiveRelease 执行金丝雀晋级或蓝绿切换。
func (s *Service) PromoteProgressiveRelease(ctx context.Context, projectID, runID uint, req ProgressivePromoteRequest, actor *auth.CurrentUser) (map[string]any, error) {
	release, dc, wl, err := s.loadProgressiveContext(ctx, projectID, runID, actor, req.ClusterID, req.Namespace, req.Workload, req.ServiceName)
	if err != nil {
		return nil, err
	}
	strategy := normalizeDeployStrategy(dc.DeployStrategy)
	if strategy == model.CicdDeployStrategyRolling {
		return nil, constants.ErrBadRequestWithMsg("当前发布配置为滚动发布，无需晋级")
	}
	if s.k8sProgressive.EnsureCanary == nil {
		return nil, constants.ErrBadRequestWithMsg("未注入渐进式发布能力")
	}
	st := parseProgressiveState(release.ProgressiveJSON, strategy, dc.CanaryStepsJSON)
	image := strings.TrimSpace(release.ImageAddress)
	out := map[string]any{"strategy": strategy, "release_id": release.ID}

	switch strategy {
	case model.CicdDeployStrategyCanary:
		detail, nextSt, err := s.promoteCanary(ctx, wl, dc, st, image, req)
		if err != nil {
			return nil, err
		}
		st = nextSt
		out["detail"] = detail
	case model.CicdDeployStrategyBlueGreen:
		detail, nextSt, err := s.promoteBlueGreen(ctx, wl, dc, st, image, req)
		if err != nil {
			return nil, err
		}
		st = nextSt
		out["detail"] = detail
	default:
		return nil, constants.ErrBadRequestWithMsg("未知发布策略: " + strategy)
	}
	st.UpdatedAt = time.Now()
	if err := s.saveProgressiveState(ctx, release.ID, st); err != nil {
		return nil, err
	}
	out["state"] = st
	return out, nil
}

// AbortProgressiveRelease 中止渐进式发布。
func (s *Service) AbortProgressiveRelease(ctx context.Context, projectID, runID uint, req ProgressiveAbortRequest, actor *auth.CurrentUser) (map[string]any, error) {
	release, dc, wl, err := s.loadProgressiveContext(ctx, projectID, runID, actor, req.ClusterID, req.Namespace, req.Workload, req.ServiceName)
	if err != nil {
		return nil, err
	}
	strategy := normalizeDeployStrategy(dc.DeployStrategy)
	if strategy == model.CicdDeployStrategyRolling {
		return nil, constants.ErrBadRequestWithMsg("当前发布配置为滚动发布")
	}
	st := parseProgressiveState(release.ProgressiveJSON, strategy, dc.CanaryStepsJSON)
	var detail any
	switch strategy {
	case model.CicdDeployStrategyCanary:
		canary := wl.Name + "-canary"
		if err := s.k8sProgressive.Scale(ctx, wl.ClusterID, wl.Namespace, canary, 0); err != nil {
			return nil, err
		}
		detail = map[string]any{"scaled_to_zero": canary}
		st.LastAction = "abort_canary"
		st.CurrentPercent = 0
		st.StepIndex = 0
	case model.CicdDeployStrategyBlueGreen:
		svc := wl.ServiceName
		if svc == "" {
			svc = wl.Name
		}
		// 切回 blue
		sw, err := s.k8sProgressive.SwitchColor(ctx, wl.ClusterID, wl.Namespace, svc, "blue")
		if err != nil {
			return nil, err
		}
		_ = s.k8sProgressive.Scale(ctx, wl.ClusterID, wl.Namespace, wl.Name+"-green", 0)
		detail = sw
		st.ActiveColor = "blue"
		st.LastAction = "abort_blue_green"
	}
	st.UpdatedAt = time.Now()
	if err := s.saveProgressiveState(ctx, release.ID, st); err != nil {
		return nil, err
	}
	return map[string]any{"strategy": strategy, "state": st, "detail": detail}, nil
}

func (s *Service) promoteCanary(ctx context.Context, wl linkedWorkloadExt, dc *model.CicdDeployConfig, st progressiveState, image string, req ProgressivePromoteRequest) (any, progressiveState, error) {
	stable := wl.Name
	canary := stable + "-canary"
	canaryReplicas := int32(dc.CanaryReplicas)
	if canaryReplicas < 1 {
		canaryReplicas = 1
	}
	total := int32(dc.Replicas)
	if total < 1 {
		total = 1
	}

	ensured, err := s.k8sProgressive.EnsureCanary(ctx, wl.ClusterID, wl.Namespace, stable, canary, image, canaryReplicas)
	if err != nil {
		return nil, st, err
	}

	if req.Final || (len(st.Steps) > 0 && st.StepIndex >= len(st.Steps)-1 && req.TargetPercent == 0 && st.CurrentPercent >= st.Steps[len(st.Steps)-1]) {
		if image != "" {
			if _, err := s.k8sProgressive.PatchImage(ctx, wl.ClusterID, wl.Namespace, stable, image); err != nil {
				return nil, st, err
			}
		}
		if err := s.k8sProgressive.Scale(ctx, wl.ClusterID, wl.Namespace, canary, 0); err != nil {
			return nil, st, err
		}
		st.CurrentPercent = 100
		st.StepIndex = len(st.Steps)
		st.LastAction = "finalize_canary"
		return map[string]any{"ensured": ensured, "finalized": true}, st, nil
	}

	target := req.TargetPercent
	if target <= 0 {
		if st.StepIndex+1 < len(st.Steps) {
			st.StepIndex++
		}
		if st.StepIndex < len(st.Steps) {
			target = st.Steps[st.StepIndex]
		} else {
			target = 100
		}
	}
	if target > 100 {
		target = 100
	}
	canaryN := int32(float64(total) * float64(target) / 100.0)
	if canaryN < 1 && target > 0 {
		canaryN = 1
	}
	stableN := total - canaryN
	if stableN < 0 {
		stableN = 0
	}
	if err := s.k8sProgressive.Scale(ctx, wl.ClusterID, wl.Namespace, canary, canaryN); err != nil {
		return nil, st, err
	}
	if err := s.k8sProgressive.Scale(ctx, wl.ClusterID, wl.Namespace, stable, stableN); err != nil {
		return nil, st, err
	}
	st.CurrentPercent = target
	st.LastAction = fmt.Sprintf("promote_canary_%d", target)
	return map[string]any{
		"ensured":          ensured,
		"canary_replicas":  canaryN,
		"stable_replicas":  stableN,
		"target_percent":   target,
	}, st, nil
}

func (s *Service) promoteBlueGreen(ctx context.Context, wl linkedWorkloadExt, dc *model.CicdDeployConfig, st progressiveState, image string, req ProgressivePromoteRequest) (any, progressiveState, error) {
	base := wl.Name
	replicas := int32(dc.Replicas)
	if replicas < 1 {
		replicas = 1
	}
	// 部署 green
	ensured, err := s.k8sProgressive.EnsureColor(ctx, wl.ClusterID, wl.Namespace, base, "green", image, replicas)
	if err != nil {
		return nil, st, err
	}
	svc := strings.TrimSpace(req.ServiceName)
	if svc == "" {
		svc = wl.ServiceName
	}
	if svc == "" {
		svc = dc.BlueGreenService
	}
	if svc == "" {
		svc = base
	}
	sw, err := s.k8sProgressive.SwitchColor(ctx, wl.ClusterID, wl.Namespace, svc, "green")
	if err != nil {
		return nil, st, err
	}
	st.ActiveColor = "green"
	st.LastAction = "switch_to_green"
	st.CurrentPercent = 100
	return map[string]any{"ensured": ensured, "switch": sw, "service": svc}, st, nil
}

type linkedWorkloadExt struct {
	linkedWorkload
	ServiceName string
}

func (s *Service) loadProgressiveContext(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser, clusterID uint, ns, workload, serviceName string) (*model.CicdReleaseRun, *model.CicdDeployConfig, linkedWorkloadExt, error) {
	release, err := s.assertReleaseRunAccess(ctx, projectID, runID, actor, "release")
	if err != nil {
		return nil, nil, linkedWorkloadExt{}, err
	}
	if !strings.EqualFold(release.ReleaseKind, model.CicdDeployKindContainer) {
		return nil, nil, linkedWorkloadExt{}, constants.ErrBadRequestWithMsg("仅容器化发布支持金丝雀/蓝绿")
	}
	if release.Status != model.CicdRunStatusSuccess && release.Status != model.CicdRunStatusRunning {
		// 允许 success；running 时也可能先晋级（视 Jenkins 是否已部署 canary）
		if release.Status != model.CicdRunStatusPendingExecution {
			// still allow after success primarily
		}
	}
	var dc model.CicdDeployConfig
	if release.DeployConfigID != nil {
		if err := s.db.WithContext(ctx).Where("id = ?", *release.DeployConfigID).First(&dc).Error; err != nil {
			return nil, nil, linkedWorkloadExt{}, constants.ErrBadRequestWithMsg("发布配置不存在")
		}
	} else {
		return nil, nil, linkedWorkloadExt{}, constants.ErrBadRequestWithMsg("工单未关联发布配置")
	}
	wl := linkedWorkloadExt{linkedWorkload: s.lookupLinkedWorkload(ctx, release.ServiceID)}
	if clusterID > 0 {
		wl.ClusterID = clusterID
	}
	if ns != "" {
		wl.Namespace = ns
	}
	if workload != "" {
		wl.Name = workload
		wl.Kind = "Deployment"
	}
	if wl.ClusterID == 0 && dc.K8sClusterID != nil {
		wl.ClusterID = *dc.K8sClusterID
	}
	if wl.Namespace == "" {
		wl.Namespace = dc.K8sNamespace
	}
	if wl.Name == "" {
		wl.Name = strings.TrimSpace(dc.ImageName)
	}
	wl.ServiceName = strings.TrimSpace(serviceName)
	if wl.ServiceName == "" {
		wl.ServiceName = strings.TrimSpace(dc.BlueGreenService)
	}
	if wl.ClusterID == 0 || wl.Namespace == "" || wl.Name == "" {
		return nil, nil, linkedWorkloadExt{}, constants.ErrBadRequestWithMsg("请提供 cluster_id / namespace / workload，或配置 K8s 工作负载关联")
	}
	return release, &dc, wl, nil
}

func (s *Service) saveProgressiveState(ctx context.Context, releaseID uint, st progressiveState) error {
	b, _ := json.Marshal(st)
	return s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).Where("id = ?", releaseID).
		Update("progressive_json", string(b)).Error
}

func parseProgressiveState(raw, strategy, stepsJSON string) progressiveState {
	st := progressiveState{Strategy: strategy, Steps: parseCanarySteps(stepsJSON)}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	if st.Strategy == "" {
		st.Strategy = strategy
	}
	if len(st.Steps) == 0 {
		st.Steps = parseCanarySteps(stepsJSON)
	}
	return st
}

func parseCanarySteps(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int{10, 50, 100}
	}
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n <= 0 {
			continue
		}
		if n > 100 {
			n = 100
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return []int{10, 50, 100}
	}
	return out
}

func normalizeDeployStrategy(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case model.CicdDeployStrategyCanary:
		return model.CicdDeployStrategyCanary
	case model.CicdDeployStrategyBlueGreen, "blue-green", "bluegreen":
		return model.CicdDeployStrategyBlueGreen
	default:
		return model.CicdDeployStrategyRolling
	}
}
