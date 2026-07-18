package system

import (
	"fmt"

	"yunshu/internal/model"

	"github.com/casbin/casbin/v2"
)

func UserSubject(userID uint) string {
	return fmt.Sprintf("user:%d", userID)
}

// SyncUserRoles 重建用户的角色分组关系。已禁用的角色不写入 Casbin，
// 避免禁用角色仍被 Enforce 命中（角色重新启用时由 EnableRoleGroupings 补回）。
func SyncUserRoles(enforcer *casbin.SyncedEnforcer, userID uint, roles []model.Role) error {
	subject := UserSubject(userID)
	if _, err := enforcer.DeleteRolesForUser(subject); err != nil {
		return err
	}
	for _, role := range roles {
		if role.Status == model.StatusDisabled {
			continue
		}
		if _, err := enforcer.AddRoleForUser(subject, role.Code); err != nil {
			return err
		}
	}
	return nil
}

// DisableRoleGroupings 移除该角色下所有 user→role 分组关系，使禁用即时生效。
// 保留 role→permission 的 p 策略，重新启用时无需重建授权。
func DisableRoleGroupings(enforcer *casbin.SyncedEnforcer, roleCode string) error {
	groupings := enforcer.GetFilteredGroupingPolicy(1, roleCode)
	for _, grouping := range groupings {
		if len(grouping) < 2 {
			continue
		}
		if _, err := enforcer.RemoveGroupingPolicy(grouping[0], roleCode); err != nil {
			return err
		}
	}
	return nil
}

// EnableRoleGroupings 依据 DB 中的绑定关系为指定用户补回 user→role 分组关系。
func EnableRoleGroupings(enforcer *casbin.SyncedEnforcer, roleCode string, userIDs []uint) error {
	for _, id := range userIDs {
		if _, err := enforcer.AddRoleForUser(UserSubject(id), roleCode); err != nil {
			return err
		}
	}
	return nil
}

func ReplaceRoleCode(enforcer *casbin.SyncedEnforcer, oldCode, newCode string) error {
	if oldCode == newCode {
		return nil
	}

	policies := enforcer.GetFilteredPolicy(0, oldCode)
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}
		if _, err := enforcer.RemovePolicy(policy[0], policy[1], policy[2]); err != nil {
			return resyncOnError(enforcer, err)
		}
		if _, err := enforcer.AddPolicy(newCode, policy[1], policy[2]); err != nil {
			return resyncOnError(enforcer, err)
		}
	}

	groupings := enforcer.GetFilteredGroupingPolicy(1, oldCode)
	for _, grouping := range groupings {
		if len(grouping) < 2 {
			continue
		}
		if _, err := enforcer.RemoveGroupingPolicy(grouping[0], oldCode); err != nil {
			return resyncOnError(enforcer, err)
		}
		if _, err := enforcer.AddGroupingPolicy(grouping[0], newCode); err != nil {
			return resyncOnError(enforcer, err)
		}
	}
	return nil
}

// resyncOnError 在批量策略改写中途失败时，从 DB 重新加载策略，
// 使内存中的 enforcer 状态与持久化状态一致，避免遗留半改写的策略被 Enforce 命中。
// 返回原始错误（若 reload 也失败则包装两者）。
func resyncOnError(enforcer *casbin.SyncedEnforcer, cause error) error {
	if reloadErr := enforcer.LoadPolicy(); reloadErr != nil {
		return fmt.Errorf("policy update failed: %w; reload after failure also failed: %v", cause, reloadErr)
	}
	return cause
}

func RemoveRolePolicies(enforcer *casbin.SyncedEnforcer, roleCode string) error {
	if _, err := enforcer.DeletePermissionsForUser(roleCode); err != nil {
		return err
	}
	if _, err := enforcer.DeleteRolesForUser(roleCode); err != nil {
		return err
	}
	return nil
}

func ReplacePermissionResource(enforcer *casbin.SyncedEnforcer, oldResource, oldAction, newResource, newAction string) error {
	if oldResource == newResource && oldAction == newAction {
		return nil
	}

	policies := enforcer.GetFilteredPolicy(1, oldResource, oldAction)
	for _, policy := range policies {
		if len(policy) < 3 {
			continue
		}
		if _, err := enforcer.RemovePolicy(policy[0], oldResource, oldAction); err != nil {
			return resyncOnError(enforcer, err)
		}
		if _, err := enforcer.AddPolicy(policy[0], newResource, newAction); err != nil {
			return resyncOnError(enforcer, err)
		}
	}
	return nil
}

func RemovePermissionPolicies(enforcer *casbin.SyncedEnforcer, resource, action string) error {
	_, err := enforcer.RemoveFilteredPolicy(1, resource, action)
	return err
}

// AddRolePolicies 批量写入角色 API 授权（已存在的策略会被 Casbin 跳过）。
func AddRolePolicies(enforcer *casbin.SyncedEnforcer, roleCode string, perms []model.Permission) error {
	if len(perms) == 0 {
		return nil
	}
	rules := make([][]string, len(perms))
	for i, p := range perms {
		rules[i] = []string{roleCode, p.Resource, p.Action}
	}
	_, err := enforcer.AddPolicies(rules)
	return err
}
