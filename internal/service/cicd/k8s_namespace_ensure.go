package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

// K8sNamespaceEnsurer 发布前确保目标命名空间存在（可选注入）。
type K8sNamespaceEnsurer interface {
	EnsureNamespaceExists(ctx context.Context, clusterID uint, name string) error
}

func (s *Service) ensureReleaseNamespace(ctx context.Context, dc *model.CicdDeployConfig) error {
	if s.nsEnsurer == nil || dc == nil {
		return nil
	}
	if !strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		return nil
	}
	ns := strings.TrimSpace(dc.K8sNamespace)
	if ns == "" || dc.K8sClusterID == nil || *dc.K8sClusterID == 0 {
		return nil
	}
	return s.nsEnsurer.EnsureNamespaceExists(ctx, *dc.K8sClusterID, ns)
}
