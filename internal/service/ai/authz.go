package ai

import (
	"context"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// assertProjectMember 校验当前用户对 projectID 的成员资格（超级管理员放行）。
func (s *Service) assertProjectMember(ctx context.Context, actor *auth.CurrentUser, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project_id 必填")
	}
	if actor == nil {
		if u, ok := auth.RequestUserFromContext(ctx); ok {
			actor = u
		}
	}
	if actor == nil || actor.ID == 0 {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrForbiddenWithMsg("无权访问该项目")
	}
	if err != nil {
		return err
	}
	return nil
}

// resolveActor 优先用传入 actor，其次从 RequestContext 取完整用户；禁止仅用 ID 的残缺主体。
func resolveActor(ctx context.Context, actor *auth.CurrentUser) *auth.CurrentUser {
	if actor != nil && actor.ID > 0 {
		return actor
	}
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && u.ID > 0 {
		return u
	}
	return nil
}

// canReviewApprovals 超级管理员或具备审批查看全局的运维角色。
func canReviewApprovals(actor *auth.CurrentUser) bool {
	if actor == nil {
		return false
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true
	}
	for _, c := range actor.RoleCodes {
		switch c {
		case "admin", "ops-admin", "ai-approver":
			return true
		}
	}
	return false
}
