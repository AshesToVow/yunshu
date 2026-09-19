package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

func (s *Service) serviceUsesK8sPipeline(ctx context.Context, svc *model.CicdService) bool {
	if svc == nil {
		return false
	}
	if strings.EqualFold(svc.ServiceType, model.CicdServiceTypeMicro) {
		return true
	}
	cnt, _ := s.repo.CountContainerDeploys(ctx, svc.ID)
	return cnt > 0
}

func (s *Service) primaryContainerDeployConfig(ctx context.Context, serviceID uint) *model.CicdDeployConfig {
	dc, err := s.repo.GetFirstContainerDeploy(ctx, serviceID)
	if err != nil {
		return nil
	}
	return dc
}

func (s *Service) firstDeployConfig(ctx context.Context, serviceID uint) *model.CicdDeployConfig {
	dc, err := s.repo.GetFirstDeployConfig(ctx, serviceID)
	if err != nil {
		return nil
	}
	return dc
}

func (s *Service) defaultBuildTenv(ctx context.Context, serviceID uint) string {
	if dc := s.primaryContainerDeployConfig(ctx, serviceID); dc != nil && strings.TrimSpace(dc.Tenv) != "" {
		return strings.TrimSpace(dc.Tenv)
	}
	if dc := s.firstDeployConfig(ctx, serviceID); dc != nil && strings.TrimSpace(dc.Tenv) != "" {
		return strings.TrimSpace(dc.Tenv)
	}
	return "dev"
}
