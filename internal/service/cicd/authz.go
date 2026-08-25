package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/projectacl"
)

func (s *Service) assertReleaseRunAccess(ctx context.Context, projectID, runID uint, actor *auth.CurrentUser, need string) (*model.CicdReleaseRun, error) {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&release).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	if err := s.AssertCicdAccess(ctx, projectID, release.ServiceID, actor, need); err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) RequireProjectAdmin(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	return s.requireProjectAdmin(ctx, projectID, actor)
}

func (s *Service) requireProjectAdmin(ctx context.Context, projectID uint, actor *auth.CurrentUser) error {
	ok, err := projectacl.CanManageGrants(ctx, s.memberRepo, projectID, actor)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

// forbidSelfApprove 职责分离：提交人不得审批自己的发布（超级管理员可豁免）。
func (s *Service) forbidSelfApprove(ctx context.Context, actor *auth.CurrentUser, submitterUserID uint) error {
	cfg := s.resolvedConfig(ctx)
	if !cfg.ForbidSelfApprove {
		return nil
	}
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if submitterUserID == 0 || actor == nil || actor.ID == 0 {
		return nil
	}
	if actor.ID == submitterUserID {
		return constants.ErrForbiddenWithMsg("职责分离：提交人不可审批自己的发布")
	}
	return nil
}

func (s *Service) enforceProdDeployAudit(ctx context.Context, tenv string, auditEnabled *bool) bool {
	if auditEnabled != nil && *auditEnabled {
		return true
	}
	cfg := s.resolvedConfig(ctx)
	if !cfg.ProdForceAudit {
		if auditEnabled != nil {
			return *auditEnabled
		}
		return false
	}
	if isProdTenv(tenv) {
		return true
	}
	if auditEnabled != nil {
		return *auditEnabled
	}
	return false
}

func isProdTenv(tenv string) bool {
	switch stringsTrimLower(tenv) {
	case "prod", "production", "online", "生产", "生产环境":
		return true
	default:
		return false
	}
}

func stringsTrimLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
