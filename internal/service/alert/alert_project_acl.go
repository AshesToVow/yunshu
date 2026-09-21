package alert

import (
	"context"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

func projectIDPtr(p *uint) uint {
	if p == nil {
		return 0
	}
	return *p
}

// assertAlertProjectWrite 项目内写操作须为成员；project_id=0 仅超级管理员。
func assertAlertProjectWrite(ctx context.Context, memberRepo interfaces.ProjectMemberRepository, projectID uint) error {
	actor, _ := auth.RequestUserFromContext(ctx)
	if projectID == 0 {
		if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
			return nil
		}
		return constants.ErrForbiddenWithMsg("全局资源（project_id=0）仅超级管理员可操作")
	}
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if memberRepo == nil || actor == nil || actor.ID == 0 {
		return constants.ErrForbidden
	}
	_, err := memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrForbiddenWithMsg("非项目成员，无法操作该项目告警配置")
		}
		return err
	}
	return nil
}
