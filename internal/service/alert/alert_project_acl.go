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

// assertAlertProjectVisible 读路径：请求用户须为成员或超管；无用户上下文（后台任务）放行。
func assertAlertProjectVisible(ctx context.Context, memberRepo interfaces.ProjectMemberRepository, projectID uint) error {
	actor, _ := auth.RequestUserFromContext(ctx)
	if actor == nil || actor.ID == 0 {
		return nil
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if projectID == 0 {
		return constants.ErrForbiddenWithMsg("全局资源（project_id=0）仅超级管理员可查看")
	}
	if memberRepo == nil {
		return constants.ErrForbidden
	}
	_, err := memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrForbiddenWithMsg("非项目成员，无法查看该项目告警配置")
		}
		return err
	}
	return nil
}

// resolveAlertEventScope 解析事件列表/导出的项目范围。
// 指定 project_id 时校验可见性；未指定时非超管用户限制为其所属项目（denyAll 表示无任何项目）。
func resolveAlertEventScope(ctx context.Context, memberRepo interfaces.ProjectMemberRepository, requested uint) (projectID uint, projectIDs []uint, denyAll bool, err error) {
	if requested > 0 {
		if err := assertAlertProjectVisible(ctx, memberRepo, requested); err != nil {
			return 0, nil, false, err
		}
		return requested, nil, false, nil
	}
	actor, _ := auth.RequestUserFromContext(ctx)
	if actor == nil || actor.ID == 0 || auth.IsSuperAdminRole(actor.RoleCodes) {
		return 0, nil, false, nil
	}
	if memberRepo == nil {
		return 0, nil, true, nil
	}
	ids, err := memberRepo.ListProjectIDsByUser(ctx, actor.ID)
	if err != nil {
		return 0, nil, false, err
	}
	if len(ids) == 0 {
		return 0, nil, true, nil
	}
	return 0, ids, false, nil
}
