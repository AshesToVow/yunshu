package cicd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

// PlatformRollbackRequest 容器模式平台侧 Deployment/STS 回滚。
type PlatformRollbackRequest struct {
	ClusterID uint   `json:"cluster_id"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"` // Deployment|StatefulSet
	Name      string `json:"name"`
	Revision  int64  `json:"revision"`
}

// K8sRolloutUndoFn 由路由层注入，避免 cicd 包直接依赖 k8s service。
type K8sRolloutUndoFn func(ctx context.Context, kind string, clusterID uint, namespace, name string, revision int64) (map[string]any, error)

func (s *Service) SetK8sRolloutUndo(fn K8sRolloutUndoFn) {
	s.k8sRolloutUndo = fn
}

// PlatformRollbackRelease 对容器发布工单执行平台回滚。
func (s *Service) PlatformRollbackRelease(ctx context.Context, projectID, runID uint, req PlatformRollbackRequest, actor *auth.CurrentUser) (map[string]any, error) {
	release, err := s.assertReleaseRunAccess(ctx, projectID, runID, actor, "release")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(release.ReleaseKind, model.CicdDeployKindContainer) {
		return nil, constants.ErrBadRequestWithMsg("仅容器化发布支持平台回滚")
	}
	if s.k8sRolloutUndo == nil {
		return nil, constants.ErrBadRequestWithMsg("未注入 K8s 回滚能力")
	}
	kind := strings.TrimSpace(req.Kind)
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	clusterID := req.ClusterID
	if ns == "" || name == "" || kind == "" || clusterID == 0 {
		parsed := s.lookupLinkedWorkload(ctx, release.ServiceID)
		if ns == "" {
			ns = parsed.Namespace
		}
		if name == "" {
			name = parsed.Name
		}
		if kind == "" {
			kind = parsed.Kind
		}
		if clusterID == 0 {
			clusterID = parsed.ClusterID
		}
	}
	if clusterID == 0 || ns == "" || name == "" {
		return nil, constants.ErrBadRequestWithMsg("请提供 cluster_id / namespace / kind / name，或配置 K8s 工作负载关联")
	}
	out, err := s.k8sRolloutUndo(ctx, kind, clusterID, ns, name, req.Revision)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(map[string]any{
		"platform_rollback": out,
		"release_id":        release.ID,
	})
	_ = s.db.WithContext(ctx).Model(&release).Updates(map[string]any{
		"verify_json": string(b),
	}).Error
	return out, nil
}

type linkedWorkload struct {
	ClusterID uint
	Namespace string
	Kind      string
	Name      string
}

func (s *Service) lookupLinkedWorkload(ctx context.Context, cicdServiceID uint) linkedWorkload {
	var catalogLink model.ServiceLink
	if err := s.db.WithContext(ctx).
		Where("link_type = ? AND ref_id = ?", model.ServiceLinkCicdService, cicdServiceID).
		First(&catalogLink).Error; err != nil {
		return linkedWorkload{}
	}
	var wl model.ServiceLink
	if err := s.db.WithContext(ctx).
		Where("service_id = ? AND link_type = ?", catalogLink.ServiceID, model.ServiceLinkK8sWorkload).
		First(&wl).Error; err != nil {
		return linkedWorkload{}
	}
	parts := strings.Split(strings.TrimSpace(wl.RefKey), "/")
	if len(parts) < 4 {
		return linkedWorkload{}
	}
	var cid uint
	_, _ = fmt.Sscanf(parts[0], "%d", &cid)
	return linkedWorkload{
		ClusterID: cid,
		Namespace: parts[1],
		Kind:      parts[2],
		Name:      parts[3],
	}
}
