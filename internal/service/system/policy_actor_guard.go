package system

import (
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"github.com/casbin/casbin/v2"
)

// authorizeRoleTarget 校验操作者能否改动目标角色的策略：超管放行，
// 其余操作者不得改动自己所属的角色，避免给自己提权。
func authorizeRoleTarget(actor *auth.CurrentUser, targetRoleCode string) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	target := strings.TrimSpace(targetRoleCode)
	if target == "" {
		return constants.ErrBadRequestWithMsg("目标角色编码为空")
	}
	for _, code := range actor.RoleCodes {
		if strings.TrimSpace(code) == target {
			return constants.ErrForbiddenWithMsg("不能修改自己所属角色的权限")
		}
	}
	return nil
}

// authorizeRolePolicyChange 在目标角色校验之上，要求非超管只能授予/回收自己已持有的权限。
// holdsPermission 由调用方用 Casbin Enforce 计算；超管不做该限制。
func authorizeRolePolicyChange(actor *auth.CurrentUser, targetRoleCode string, holdsPermission bool) error {
	if err := authorizeRoleTarget(actor, targetRoleCode); err != nil {
		return err
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if !holdsPermission {
		return constants.ErrForbiddenWithMsg("只能授予或回收自己已持有的权限")
	}
	return nil
}

// actorHoldsPermission 判断操作者主体当前是否被授予了指定资源+方法。
func actorHoldsPermission(enforcer *casbin.SyncedEnforcer, actor *auth.CurrentUser, resource, action string) bool {
	if enforcer == nil || actor == nil {
		return false
	}
	allowed, err := enforcer.Enforce(UserSubject(actor.ID), strings.TrimSpace(resource), strings.ToUpper(strings.TrimSpace(action)))
	if err != nil {
		return false
	}
	return allowed
}

// authorizePermissionCatalogChange 权限目录（permission 点）属于鉴权边界本身，仅内置超管可增删。
func authorizePermissionCatalogChange(actor *auth.CurrentUser, action string) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	return constants.ErrForbiddenWithMsg("仅内置超级管理员可以" + action + "权限点")
}
