package cmd

import (
	"context"

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
		{Resource: "/api/v1/alerts/rule-templates", Action: "GET"},
		{Resource: "/api/v1/alerts/rule-templates", Action: "POST"},
		{Resource: "/api/v1/alerts/rule-templates/:id", Action: "PUT"},
		{Resource: "/api/v1/alerts/rule-templates/:id", Action: "DELETE"},
		// 路径迁移重复项
		{Resource: "/api/v1/admin/banned-ips", Action: "GET"},
		{Resource: "/api/v1/admin/banned-ips/unban", Action: "POST"},
		{Resource: "/api/v1/k8s-policies/grant", Action: "POST"},
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
