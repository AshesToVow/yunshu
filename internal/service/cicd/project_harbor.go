package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

// loadProjectHarbor 读取项目级 Harbor；字段为空表示不覆盖全局。
func (s *Service) loadProjectHarbor(ctx context.Context, projectID uint) (url, project string) {
	if projectID == 0 || s.db == nil {
		return "", ""
	}
	var p model.Project
	if err := s.db.WithContext(ctx).
		Select("harbor_url", "harbor_project").
		Where("id = ?", projectID).
		First(&p).Error; err != nil {
		return "", ""
	}
	return strings.TrimSpace(p.HarborURL), strings.TrimSpace(p.HarborProject)
}
