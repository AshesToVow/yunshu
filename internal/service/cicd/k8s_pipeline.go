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
	var cnt int64
	_ = s.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).
		Where("service_id = ? AND deploy_kind = ?", svc.ID, model.CicdDeployKindContainer).
		Count(&cnt).Error
	return cnt > 0
}

func (s *Service) primaryContainerDeployConfig(ctx context.Context, serviceID uint) *model.CicdDeployConfig {
	var dc model.CicdDeployConfig
	if err := s.db.WithContext(ctx).
		Where("service_id = ? AND deploy_kind = ?", serviceID, model.CicdDeployKindContainer).
		Order("id ASC").
		First(&dc).Error; err != nil {
		return nil
	}
	return &dc
}

func (s *Service) firstDeployConfig(ctx context.Context, serviceID uint) *model.CicdDeployConfig {
	var dc model.CicdDeployConfig
	if err := s.db.WithContext(ctx).
		Where("service_id = ?", serviceID).
		Order("id ASC").
		First(&dc).Error; err != nil {
		return nil
	}
	return &dc
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
