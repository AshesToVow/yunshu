package cicd

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
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
// Harbor 优先走注册中心绑定（ResolveRegistryForProject），再回退 projects.harbor_*。
func (s *Service) loadProjectCicdOverrides(ctx context.Context, projectID uint) projectCicdOverrides {
	if projectID == 0 || s.repo == nil {
		return projectCicdOverrides{}
	}
	harborURL, harborProject, err := s.repo.GetProjectHarborFields(ctx, projectID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return projectCicdOverrides{}
	}
	out := projectCicdOverrides{
		HarborURL:     harborURL,
		HarborProject: harborProject,
	}
	if s.projectRepo != nil {
		if p, err := s.projectRepo.GetByID(ctx, projectID); err == nil && p != nil {
			out.ApolloMeta = strings.TrimSpace(p.ApolloMeta)
			out.ApolloEnv = strings.TrimSpace(p.ApolloEnv)
			out.ApolloNamespaces = strings.TrimSpace(p.ApolloNamespaces)
		}
	}
	// 绑定注册中心时覆盖 Harbor URL/项目（Apollo 仍读 projects 表）。
	if bind, err := s.repo.GetProjectRegistryBinding(ctx, projectID); err == nil && bind != nil && bind.RegistryID > 0 {
		if reg, err := s.repo.GetImageRegistry(ctx, bind.RegistryID); err == nil && reg != nil {
			out.HarborURL = stripHarborHost(reg.URL)
			if v := strings.TrimSpace(bind.HarborProject); v != "" {
				out.HarborProject = v
			} else if v := strings.TrimSpace(reg.DefaultProject); v != "" {
				out.HarborProject = v
			}
		}
	}
	return out
}

// loadProjectHarbor 读取项目级 Harbor；字段为空表示不覆盖全局。
func (s *Service) loadProjectHarbor(ctx context.Context, projectID uint) (url, project string) {
	o := s.loadProjectCicdOverrides(ctx, projectID)
	return o.HarborURL, o.HarborProject
}
