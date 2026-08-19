package cmd

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type permissionPair struct {
	Resource string
	Action   string
}

// stalePermissions 历史误同步或已下线路由，不应出现在 permissions 表。
func stalePermissions() []permissionPair {
	return []permissionPair{
		{Resource: "/api/v1/alerts/duty-schedules", Action: "GET"},
		{Resource: "/api/v1/alerts/duty-schedules", Action: "POST"},
		{Resource: "/api/v1/alerts/duty-schedules/:id", Action: "PUT"},
		{Resource: "/api/v1/alerts/duty-schedules/:id", Action: "DELETE"},
		{Resource: "/api/v1/alerts/policies", Action: "GET"},
		{Resource: "/api/v1/alerts/policies", Action: "POST"},
		{Resource: "/api/v1/alerts/policies/:id", Action: "PUT"},
		{Resource: "/api/v1/alerts/policies/:id", Action: "DELETE"},
		// 内置模板仅提供 GET 列表；旧的 CRUD 路径已下线
		{Resource: "/api/v1/alerts/rule-templates", Action: "POST"},
		{Resource: "/api/v1/alerts/rule-templates/:id", Action: "PUT"},
		{Resource: "/api/v1/alerts/rule-templates/:id", Action: "DELETE"},
		// 路径迁移重复项
		{Resource: "/api/v1/admin/banned-ips", Action: "GET"},
		{Resource: "/api/v1/admin/banned-ips/unban", Action: "POST"},
		{Resource: "/api/v1/k8s-policies/grant", Action: "POST"},
		// 侧栏树已改为登录即可，不再走 Casbin
		{Resource: "/api/v1/menus/tree", Action: "GET"},
		// 不走 Casbin authorize 的公开/登录接口
		{Resource: "/api/v1/health", Action: "GET"},
		{Resource: "/api/v1/auth/verification-code", Action: "POST"},
		{Resource: "/api/v1/auth/login-code", Action: "POST"},
		{Resource: "/api/v1/auth/password-login-code", Action: "POST"},
		{Resource: "/api/v1/auth/login", Action: "POST"},
		{Resource: "/api/v1/auth/email-login", Action: "POST"},
		{Resource: "/api/v1/auth/register", Action: "POST"},
		{Resource: "/api/v1/auth/logout", Action: "POST"},
		{Resource: "/api/v1/auth/ws-ticket", Action: "POST"},
		{Resource: "/api/v1/auth/me", Action: "GET"},
		{Resource: "/api/v1/auth/me", Action: "PUT"},
		{Resource: "/api/v1/auth/password", Action: "PUT"},
		// 已废弃 log-agent / gRPC
		{Resource: "/api/v1/agents/public-register", Action: "POST"},
		{Resource: "/api/v1/agents/register", Action: "POST"},
		{Resource: "/api/v1/agents/health/report", Action: "POST"},
		{Resource: "/api/v1/agents/discovery/report", Action: "POST"},
		{Resource: "/api/v1/agents/runtime-config", Action: "GET"},
		{Resource: "/api/v1/projects/:id/agents", Action: "GET"},
		{Resource: "/api/v1/projects/:id/agents", Action: "POST"},
		{Resource: "/api/v1/projects/:id/agents/:agentId", Action: "DELETE"},
		{Resource: "/api/v1/projects/:id/agents/bootstrap", Action: "POST"},
		{Resource: "/api/v1/projects/:id/agents/status", Action: "GET"},
		{Resource: "/api/v1/projects/:id/agents/discovery", Action: "GET"},
		{Resource: "/api/v1/projects/:id/agents/heartbeat/batch", Action: "POST"},
		{Resource: "/api/v1/projects/:id/logs/stream", Action: "GET"},
		// 已下线 / 改为入站 Token，不走 Casbin
		{Resource: "/api/v1/alerts/webhook/alertmanager", Action: "POST"},
		{Resource: "/api/v1/loggie/heartbeat/report", Action: "POST"},
		// 历史错误路径 / 已改名 API
		{Resource: "/api/v1/k8s-event-forward", Action: "GET"},
		{Resource: "/api/v1/k8s-event-forward", Action: "POST"},
		{Resource: "/api/v1/k8s-event-forward", Action: "PUT"},
		{Resource: "/api/v1/k8s-event-forward", Action: "DELETE"},
		{Resource: "/api/v1/k8s-event-forward/rules", Action: "GET"},
		{Resource: "/api/v1/k8s-event-forward/rules", Action: "POST"},
		{Resource: "/api/v1/k8s-event-forward/settings", Action: "GET"},
		{Resource: "/api/v1/k8s-event-forward/settings", Action: "PUT"},
		{Resource: "/api/v1/component-status", Action: "GET"},
		{Resource: "/api/v1/services", Action: "GET"},
		{Resource: "/api/v1/ingress-classes", Action: "GET"},
	}
}

func removeStalePermissions(ctx context.Context, enforcer *casbin.SyncedEnforcer, db *gorm.DB) (int, error) {
	removed := 0
	for _, stale := range stalePermissions() {
		if err := service.RemovePermissionPolicies(enforcer, stale.Resource, stale.Action); err != nil {
			return removed, err
		}
		res := db.WithContext(ctx).Unscoped().
			Where("resource = ? AND action = ?", stale.Resource, stale.Action).
			Delete(&model.Permission{})
		if res.Error != nil {
			return removed, res.Error
		}
		removed += int(res.RowsAffected)
	}
	return removed, nil
}

func removeWildcardPermissions(ctx context.Context, enforcer *casbin.SyncedEnforcer, db *gorm.DB) (int, error) {
	var rows []model.Permission
	if err := db.WithContext(ctx).Unscoped().
		Where("resource LIKE ?", "%*%").
		Find(&rows).Error; err != nil {
		return 0, err
	}
	removed := 0
	for _, p := range rows {
		if err := service.RemovePermissionPolicies(enforcer, p.Resource, p.Action); err != nil {
			return removed, err
		}
		res := db.WithContext(ctx).Unscoped().
			Where("id = ?", p.ID).
			Delete(&model.Permission{})
		if res.Error != nil {
			return removed, res.Error
		}
		removed += int(res.RowsAffected)
	}
	if enforcer != nil {
		policies := enforcer.GetPolicy()
		seen := map[string]struct{}{}
		for _, p := range policies {
			if len(p) < 2 || !strings.Contains(p[1], "*") {
				continue
			}
			if _, ok := seen[p[1]]; ok {
				continue
			}
			seen[p[1]] = struct{}{}
			if _, err := enforcer.RemoveFilteredPolicy(1, p[1]); err != nil {
				return removed, err
			}
			removed++
		}
	}
	return removed, nil
}
