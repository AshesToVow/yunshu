package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

// projectCicdOverrides 项目级 CI/CD 覆盖项；空字段表示不覆盖全局/Job 默认。
type projectCicdOverrides struct {
	HarborURL        string
	HarborProject    string
	ApolloMeta       string
	ApolloEnv        string
	ApolloNamespaces string
}

// loadProjectCicdOverrides 读取项目级 Harbor / Apollo；字段为空表示不覆盖。
func (s *Service) loadProjectCicdOverrides(ctx context.Context, projectID uint) projectCicdOverrides {
	if projectID == 0 || s.db == nil {
		return projectCicdOverrides{}
	}
	var p model.Project
	if err := s.db.WithContext(ctx).
		Select(
			"harbor_url", "harbor_project",
			"apollo_meta", "apollo_env", "apollo_namespaces",
		).
		Where("id = ?", projectID).
		First(&p).Error; err != nil {
		return projectCicdOverrides{}
	}
	return projectCicdOverrides{
		HarborURL:        strings.TrimSpace(p.HarborURL),
		HarborProject:    strings.TrimSpace(p.HarborProject),
		ApolloMeta:       strings.TrimSpace(p.ApolloMeta),
		ApolloEnv:        strings.TrimSpace(p.ApolloEnv),
		ApolloNamespaces: strings.TrimSpace(p.ApolloNamespaces),
	}
}

// loadProjectHarbor 读取项目级 Harbor；字段为空表示不覆盖全局。
func (s *Service) loadProjectHarbor(ctx context.Context, projectID uint) (url, project string) {
	o := s.loadProjectCicdOverrides(ctx, projectID)
	return o.HarborURL, o.HarborProject
}
