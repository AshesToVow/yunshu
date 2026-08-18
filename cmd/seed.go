package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/model"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/password"
	"yunshu/internal/plugingate"
	"yunshu/internal/service"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

func init() {
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed default admin user, roles and permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.BuildCoreApp(configPath)
		if err != nil {
			return err
		}
		defer app.Close()

		logx.Init(app.Logger)

		ctx := context.Background()
		// 确保新增字段（如 permissions.k8s_scope_enabled）在 seed 前已完成迁移
		if err := bootstrap.AutoMigrateModels(app.DB, &app.Config.Plugins); err != nil {
			return err
		}

		// 历史错误：撤销策略实际路由为 DELETE /api/v1/policies（JSON body），与 /api/v1/policies/:id 不匹配，会导致无法撤销授权
		if err := service.RemovePermissionPolicies(app.Enforcer, "/api/v1/policies/:id", "DELETE"); err != nil {
			return err
		}
		removedStale, err := removeStalePermissions(ctx, app.Enforcer, app.DB)
		if err != nil {
			return fmt.Errorf("remove stale permissions: %w", err)
		}
		if removedStale > 0 {
			slog.Default().With("component", "seed").Info("removed stale permissions", "count", removedStale)
			fmt.Printf("removed %d stale permission records\n", removedStale)
		}

		permissions := defaultPermissions()
		adminRole := model.Role{
			Name:        "Super Admin",
			Code:        "super-admin",
			Description: "Built-in administrator role with full access.",
			Status:      model.StatusEnabled,
		}
		adminEmail := "rootwxd@163.com"
		adminUser := model.User{
			Username: "admin",
			Email:    &adminEmail,
			Nickname: "System Admin",
			Status:   model.StatusEnabled,
		}
		var adminCreated bool

		err = app.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("resource = ? AND action = ?", "/api/v1/policies/:id", "DELETE").
				Delete(&model.Permission{}).Error; err != nil {
				return err
			}
			if err := seedPermissions(ctx, tx, permissions); err != nil {
				return err
			}
			if err := upsertByKey(ctx, tx, &adminRole,
				func(db *gorm.DB) *gorm.DB { return db.Where("code = ?", adminRole.Code) },
				func(existing, incoming *model.Role) {
					existing.Name = incoming.Name
					existing.Description = incoming.Description
					existing.Status = incoming.Status
				},
				nil,
			); err != nil {
				return err
			}
			if err := upsertByKey(ctx, tx, &adminUser,
				func(db *gorm.DB) *gorm.DB { return db.Where("username = ?", adminUser.Username) },
				func(existing, incoming *model.User) {
					existing.Email = incoming.Email
					existing.Nickname = incoming.Nickname
					existing.Status = incoming.Status
				},
				func(incoming *model.User) error {
					adminCreated = true
					hashed, err := password.Hash("Admin@123")
					if err != nil {
						return err
					}
					incoming.Password = hashed
					return nil
				},
			); err != nil {
				return err
			}
			if err := tx.Model(&adminUser).Association("Roles").Replace([]model.Role{adminRole}); err != nil {
				return err
			}
			return seedMenus(ctx, tx, &app.Config.Plugins)
		})
		if err != nil {
			return err
		}

		if err := service.AddRolePolicies(app.Enforcer, adminRole.Code, permissions); err != nil {
			return err
		}
		if err := service.SyncUserRoles(app.Enforcer, adminUser.ID, []model.Role{adminRole}); err != nil {
			return err
		}

		if adminCreated {
			slog.Default().With("component", "seed").Info("Seed completed", "username", adminUser.Username, "email", adminUser.Email, "password", "Admin@123")
			fmt.Println("seed completed: created admin user admin / Admin@123")
		} else {
			slog.Default().With("component", "seed").Info("Seed completed", "username", adminUser.Username, "email", adminUser.Email)
			fmt.Println("seed completed")
		}
		return nil
	},
}

func seedPermissions(ctx context.Context, db *gorm.DB, permissions []model.Permission) error {
	normalized := make([]model.Permission, 0, len(permissions))
	seen := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		p.Resource = strings.TrimSpace(p.Resource)
		p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
		if p.Resource == "" || p.Action == "" {
			continue
		}
		key := p.Resource + "::" + p.Action
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, p)
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "resource"}, {Name: "action"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "description", "k8s_scope_enabled", "updated_at", "deleted_at",
		}),
	}).CreateInBatches(normalized, 200).Error
}

func defaultPermissions() []model.Permission {
	return []model.Permission{
		{Name: "用户列表", Resource: "/api/v1/users", Action: "GET", Description: "View user list"},
		{Name: "创建用户", Resource: "/api/v1/users", Action: "POST", Description: "Create user"},
		{Name: "用户详情", Resource: "/api/v1/users/:id", Action: "GET", Description: "View user detail"},
		{Name: "更新用户", Resource: "/api/v1/users/:id", Action: "PUT", Description: "Update user"},
		{Name: "删除用户", Resource: "/api/v1/users/:id", Action: "DELETE", Description: "Delete user"},
		{Name: "分配用户角色", Resource: "/api/v1/users/:id/roles", Action: "PUT", Description: "Assign roles to user"},
		{Name: "导出用户", Resource: "/api/v1/users/export", Action: "GET", Description: "Export users to Excel"},
		{Name: "用户导入模板", Resource: "/api/v1/users/import-template", Action: "GET", Description: "Download users import template"},
		{Name: "导入用户", Resource: "/api/v1/users/import", Action: "POST", Description: "Import users from Excel"},
		{Name: "角色列表", Resource: "/api/v1/roles", Action: "GET", Description: "View role list"},
		{Name: "创建角色", Resource: "/api/v1/roles", Action: "POST", Description: "Create role"},
		{Name: "角色详情", Resource: "/api/v1/roles/:id", Action: "GET", Description: "View role detail"},
		{Name: "更新角色", Resource: "/api/v1/roles/:id", Action: "PUT", Description: "Update role"},
		{Name: "删除角色", Resource: "/api/v1/roles/:id", Action: "DELETE", Description: "Delete role"},
		{Name: "用户组列表", Resource: "/api/v1/user-groups", Action: "GET", Description: "List user groups"},
		{Name: "创建用户组", Resource: "/api/v1/user-groups", Action: "POST", Description: "Create user group"},
		{Name: "用户组详情", Resource: "/api/v1/user-groups/:id", Action: "GET", Description: "View user group detail"},
		{Name: "更新用户组", Resource: "/api/v1/user-groups/:id", Action: "PUT", Description: "Update user group"},
		{Name: "删除用户组", Resource: "/api/v1/user-groups/:id", Action: "DELETE", Description: "Delete user group"},
		{Name: "分配用户组成员", Resource: "/api/v1/user-groups/:id/users", Action: "PUT", Description: "Replace members of user group"},
		{Name: "部门树", Resource: "/api/v1/departments/tree", Action: "GET", Description: "View department tree"},
		{Name: "部门详情", Resource: "/api/v1/departments/:id", Action: "GET", Description: "View department detail"},
		{Name: "创建部门", Resource: "/api/v1/departments", Action: "POST", Description: "Create department"},
		{Name: "更新部门", Resource: "/api/v1/departments/:id", Action: "PUT", Description: "Update department"},
		{Name: "删除部门", Resource: "/api/v1/departments/:id", Action: "DELETE", Description: "Delete department"},
		{Name: "API列表", Resource: "/api/v1/permissions", Action: "GET", Description: "View permission list"},
		{Name: "创建API", Resource: "/api/v1/permissions", Action: "POST", Description: "Create permission"},
		{Name: "批量更新 K8s 范围校验", Resource: "/api/v1/permissions/k8s-scope/batch", Action: "POST", Description: "Batch enable or disable k8s scope for cluster APIs"},
		{Name: "API详情", Resource: "/api/v1/permissions/:id", Action: "GET", Description: "View permission detail"},
		{Name: "更新API", Resource: "/api/v1/permissions/:id", Action: "PUT", Description: "Update permission"},
		{Name: "删除API", Resource: "/api/v1/permissions/:id", Action: "DELETE", Description: "Delete permission"},
		{Name: "授权列表", Resource: "/api/v1/policies", Action: "GET", Description: "View policy list"},
		{Name: "创建授权策略", Resource: "/api/v1/policies", Action: "POST", Description: "Grant permission to role"},
		{Name: "删除授权策略", Resource: "/api/v1/policies", Action: "DELETE", Description: "Revoke permission from role (JSON body)"},
		{Name: "权限菜单关联", Resource: "/api/v1/policies/menu-links", Action: "GET", Description: "List permission to menu path links"},
		{Name: "策略冲突分析", Resource: "/api/v1/policies/conflicts", Action: "GET", Description: "Analyze role policy conflicts"},
		{Name: "一键补齐入口API", Resource: "/api/v1/policies/conflicts/fix-menu-entry", Action: "POST", Description: "Create missing menu entry permissions and grant to role"},
		{Name: "清理禁用插件策略", Resource: "/api/v1/policies/conflicts/fix-disabled-plugin", Action: "POST", Description: "Revoke Casbin policies whose API belongs to a disabled plugin"},
		{Name: "统一权限树", Resource: "/api/v1/policies/permission-tree", Action: "GET", Description: "Get menu+API permission tree for role"},
		{Name: "策略模拟", Resource: "/api/v1/policies/simulate", Action: "POST", Description: "Simulate API authorization layers"},
		{Name: "K8s 动作码目录", Resource: "/api/v1/k8s-policies/actions", Action: "GET", Description: "List k8s scope action codes (reference)"},
		{Name: "K8s API 路径目录", Resource: "/api/v1/k8s-policies/paths", Action: "GET", Description: "List k8s scope API paths (reference)"},
		{Name: "K8s 集群档位列表", Resource: "/api/v1/k8s-policies", Action: "GET", Description: "List k8s cluster access grants by role"},
		{Name: "K8s 集群档位预设下发", Resource: "/api/v1/k8s-policies/grant-preset", Action: "POST", Description: "Upsert k8s cluster access preset per role/cluster"},
		{Name: "K8s 集群档位删除", Resource: "/api/v1/k8s-policies/cluster-grants/:id", Action: "DELETE", Description: "Delete one k8s cluster access grant"},
		{Name: "K8s 集群已授权矩阵", Resource: "/api/v1/k8s-policies/cluster-auth-matrix", Action: "GET", Description: "List cluster auth matrix expanded by user"},
		{Name: "K8s 用户已授权集群", Resource: "/api/v1/k8s-policies/user-cluster-auth", Action: "GET", Description: "List clusters authorized for a user"},
		{Name: "K8s 我的集群档位", Resource: "/api/v1/k8s-policies/my-access", Action: "GET", Description: "Get current user effective k8s access tier for a cluster"},
		{Name: "K8s 集群档位批量删除", Resource: "/api/v1/k8s-policies/cluster-grants/batch-delete", Action: "POST", Description: "Batch delete k8s cluster access grants"},
		{Name: "K8s 命名空间黑名单列表", Resource: "/api/v1/k8s-namespace-deny-rules", Action: "GET", Description: "List k8s namespace deny rules"},
		{Name: "K8s 命名空间黑名单新增", Resource: "/api/v1/k8s-namespace-deny-rules", Action: "POST", Description: "Create k8s namespace deny rule"},
		{Name: "K8s 命名空间黑名单删除", Resource: "/api/v1/k8s-namespace-deny-rules/:id", Action: "DELETE", Description: "Delete k8s namespace deny rule"},
		{Name: "K8s 命名空间白名单列表", Resource: "/api/v1/k8s-namespace-allow-rules", Action: "GET", Description: "List k8s namespace allow rules"},
		{Name: "K8s 命名空间白名单新增", Resource: "/api/v1/k8s-namespace-allow-rules", Action: "POST", Description: "Create k8s namespace allow rule"},
		{Name: "K8s 命名空间白名单删除", Resource: "/api/v1/k8s-namespace-allow-rules/:id", Action: "DELETE", Description: "Delete k8s namespace allow rule"},
		{Name: "注册审核列表", Resource: "/api/v1/registrations", Action: "GET", Description: "View registration requests"},
		{Name: "审核注册申请", Resource: "/api/v1/registrations/:id/review", Action: "POST", Description: "Review registration request"},
		{Name: "菜单树", Resource: "/api/v1/menus/tree", Action: "GET", Description: "View menu tree"},
		{Name: "创建菜单", Resource: "/api/v1/menus", Action: "POST", Description: "Create menu"},
		{Name: "批量更新菜单状态", Resource: "/api/v1/menus/status", Action: "PUT", Description: "Batch update menu status"},
		{Name: "更新菜单", Resource: "/api/v1/menus/:id", Action: "PUT", Description: "Update menu"},
		{Name: "删除菜单", Resource: "/api/v1/menus/:id", Action: "DELETE", Description: "Delete menu"},
		{Name: "菜单入口权限绑定", Resource: "/api/v1/menus/:id/bindings", Action: "GET", Description: "Get menu entry permission bindings"},
		{Name: "更新菜单入口权限绑定", Resource: "/api/v1/menus/:id/bindings", Action: "PUT", Description: "Replace menu entry permission bindings"},
		{Name: "数据字典列表", Resource: "/api/v1/dict/entries", Action: "GET", Description: "List dict entries"},
		{Name: "数据字典新增", Resource: "/api/v1/dict/entries", Action: "POST", Description: "Create dict entry"},
		{Name: "数据字典更新", Resource: "/api/v1/dict/entries/:id", Action: "PUT", Description: "Update dict entry"},
		{Name: "数据字典删除", Resource: "/api/v1/dict/entries/:id", Action: "DELETE", Description: "Delete dict entry"},
		{Name: "数据字典下拉", Resource: "/api/v1/dict/options/:dictType", Action: "GET", Description: "Dict options for UI"},
		{Name: "数据字典敏感明文", Resource: "/api/v1/dict/entries/:id/reveal-value", Action: "POST", Description: "Reveal sensitive dict entry value"},
		{Name: "告警通道列表", Resource: "/api/v1/alerts/channels", Action: "GET", Description: "List alert channels"},
		{Name: "创建告警通道", Resource: "/api/v1/alerts/channels", Action: "POST", Description: "Create alert channel"},
		{Name: "更新告警通道", Resource: "/api/v1/alerts/channels/:id", Action: "PUT", Description: "Update alert channel"},
		{Name: "删除告警通道", Resource: "/api/v1/alerts/channels/:id", Action: "DELETE", Description: "Delete alert channel"},
		{Name: "测试告警通道", Resource: "/api/v1/alerts/channels/:id/test", Action: "POST", Description: "Send test alert to channel"},
		{Name: "预览告警通道模板", Resource: "/api/v1/alerts/channels/preview-template", Action: "POST", Description: "Preview alert channel template"},
		{Name: "告警路由调试", Resource: "/api/v1/alerts/routing/debug", Action: "POST", Description: "Debug alert routing"},
		{Name: "告警事件列表", Resource: "/api/v1/alerts/events", Action: "GET", Description: "List alert events"},
		{Name: "告警事件分组列表", Resource: "/api/v1/alerts/events/grouped", Action: "GET", Description: "List alert events grouped"},
		{Name: "告警指纹投递追溯", Resource: "/api/v1/alerts/events/by-fingerprint", Action: "GET", Description: "Explain alert delivery by fingerprint"},
		{Name: "告警历史统计", Resource: "/api/v1/alerts/history/stats", Action: "GET", Description: "Alert history stats"},
		{Name: "接收 Alertmanager Webhook", Resource: "/api/v1/alerts/webhook/alertmanager", Action: "POST", Description: "Receive alertmanager webhook"},
		{Name: "告警数据源列表", Resource: "/api/v1/alerts/datasources", Action: "GET", Description: "List alert datasources"},
		{Name: "告警数据源连通性检测", Resource: "/api/v1/alerts/datasources/:id/ping", Action: "GET", Description: "Ping Prometheus datasource"},
		{Name: "创建告警数据源", Resource: "/api/v1/alerts/datasources", Action: "POST", Description: "Create alert datasource"},
		{Name: "更新告警数据源", Resource: "/api/v1/alerts/datasources/:id", Action: "PUT", Description: "Update alert datasource"},
		{Name: "删除告警数据源", Resource: "/api/v1/alerts/datasources/:id", Action: "DELETE", Description: "Delete alert datasource"},
		{Name: "Prometheus 活跃告警快照", Resource: "/api/v1/alerts/datasources/:id/prometheus-alerts", Action: "GET", Description: "GET /api/v1/alerts proxy"},
		{Name: "Alertmanager 静默列表", Resource: "/api/v1/alerts/datasources/:id/alertmanager-silences", Action: "GET", Description: "GET /api/v2/silences proxy"},
		{Name: "PromQL 即时查询", Resource: "/api/v1/alerts/datasources/:id/query", Action: "POST", Description: "Prometheus instant query"},
		{Name: "PromQL 范围查询", Resource: "/api/v1/alerts/datasources/:id/query_range", Action: "POST", Description: "Prometheus range query"},
		{Name: "告警静默列表", Resource: "/api/v1/alerts/silences", Action: "GET", Description: "List alert silences"},
		{Name: "创建告警静默", Resource: "/api/v1/alerts/silences", Action: "POST", Description: "Create alert silence"},
		{Name: "批量创建告警静默", Resource: "/api/v1/alerts/silences/batch", Action: "POST", Description: "Batch create alert silences"},
		{Name: "更新告警静默", Resource: "/api/v1/alerts/silences/:id", Action: "PUT", Description: "Update alert silence"},
		{Name: "删除告警静默", Resource: "/api/v1/alerts/silences/:id", Action: "DELETE", Description: "Delete alert silence"},
		{Name: "维护窗口列表", Resource: "/api/v1/alerts/maintenance-windows", Action: "GET", Description: "List alert maintenance windows"},
		{Name: "创建维护窗口", Resource: "/api/v1/alerts/maintenance-windows", Action: "POST", Description: "Create alert maintenance window"},
		{Name: "更新维护窗口", Resource: "/api/v1/alerts/maintenance-windows/:id", Action: "PUT", Description: "Update alert maintenance window"},
		{Name: "删除维护窗口", Resource: "/api/v1/alerts/maintenance-windows/:id", Action: "DELETE", Description: "Delete alert maintenance window"},
		{Name: "告警抑制规则列表", Resource: "/api/v1/alerts/inhibition-rules", Action: "GET", Description: "List alert inhibition rules"},
		{Name: "创建告警抑制规则", Resource: "/api/v1/alerts/inhibition-rules", Action: "POST", Description: "Create alert inhibition rule"},
		{Name: "更新告警抑制规则", Resource: "/api/v1/alerts/inhibition-rules/:id", Action: "PUT", Description: "Update alert inhibition rule"},
		{Name: "删除告警抑制规则", Resource: "/api/v1/alerts/inhibition-rules/:id", Action: "DELETE", Description: "Delete alert inhibition rule"},
		{Name: "刷新告警抑制缓存", Resource: "/api/v1/alerts/inhibition-rules/refresh-cache", Action: "POST", Description: "Refresh inhibition rule cache"},
		{Name: "监控告警规则列表", Resource: "/api/v1/alerts/monitor-rules", Action: "GET", Description: "List monitor alert rules"},
		{Name: "创建监控告警规则", Resource: "/api/v1/alerts/monitor-rules", Action: "POST", Description: "Create monitor alert rule"},
		{Name: "告警规则模板列表", Resource: "/api/v1/alerts/rule-templates", Action: "GET", Description: "List alert rule templates"},
		{Name: "从模板创建监控规则", Resource: "/api/v1/alerts/monitor-rules/from-template", Action: "POST", Description: "Create monitor rule from template"},
		{Name: "更新监控告警规则", Resource: "/api/v1/alerts/monitor-rules/:id", Action: "PUT", Description: "Update monitor alert rule"},
		{Name: "删除监控告警规则", Resource: "/api/v1/alerts/monitor-rules/:id", Action: "DELETE", Description: "Delete monitor alert rule"},
		{Name: "监控规则处理人", Resource: "/api/v1/alerts/monitor-rules/:id/assignees", Action: "GET", Description: "List rule assignees"},
		{Name: "配置监控规则处理人", Resource: "/api/v1/alerts/monitor-rules/:id/assignees", Action: "PUT", Description: "Upsert rule assignees"},
		{Name: "值班班次列表", Resource: "/api/v1/alerts/duty-blocks", Action: "GET", Description: "List alert duty blocks"},
		{Name: "创建值班班次", Resource: "/api/v1/alerts/duty-blocks", Action: "POST", Description: "Create alert duty block"},
		{Name: "更新值班班次", Resource: "/api/v1/alerts/duty-blocks/:id", Action: "PUT", Description: "Update alert duty block"},
		{Name: "删除值班班次", Resource: "/api/v1/alerts/duty-blocks/:id", Action: "DELETE", Description: "Delete alert duty block"},
		{Name: "值班日历", Resource: "/api/v1/alerts/duty-blocks/calendar", Action: "GET", Description: "List duty calendar"},
		{Name: "校验值班班次", Resource: "/api/v1/alerts/duty-blocks/validate", Action: "POST", Description: "Validate duty blocks"},
		{Name: "值班交接", Resource: "/api/v1/alerts/duty-blocks/:id/handoff", Action: "POST", Description: "Handoff duty block"},
		{Name: "告警订阅节点列表", Resource: "/api/v1/alerts/subscriptions", Action: "GET", Description: "List alert subscription nodes"},
		{Name: "告警订阅树", Resource: "/api/v1/alerts/subscriptions/tree", Action: "GET", Description: "Get alert subscription tree"},
		{Name: "创建告警订阅节点", Resource: "/api/v1/alerts/subscriptions", Action: "POST", Description: "Create alert subscription node"},
		{Name: "更新告警订阅节点", Resource: "/api/v1/alerts/subscriptions/:id", Action: "PUT", Description: "Update alert subscription node"},
		{Name: "删除告警订阅节点", Resource: "/api/v1/alerts/subscriptions/:id", Action: "DELETE", Description: "Delete alert subscription node"},
		{Name: "移动告警订阅节点", Resource: "/api/v1/alerts/subscriptions/:id/move", Action: "POST", Description: "Move alert subscription node"},
		{Name: "从策略迁移订阅", Resource: "/api/v1/alerts/subscriptions/migrate-from-policies", Action: "POST", Description: "Migrate subscriptions from policies"},
		{Name: "克隆项目路由", Resource: "/api/v1/alerts/subscriptions/clone-from-project", Action: "POST", Description: "Clone project routing subscriptions"},
		{Name: "告警路由向导", Resource: "/api/v1/alerts/subscriptions/wizard", Action: "POST", Description: "Create routing node via wizard"},
		{Name: "接收组列表", Resource: "/api/v1/alerts/receiver-groups", Action: "GET", Description: "List alert receiver groups"},
		{Name: "创建接收组", Resource: "/api/v1/alerts/receiver-groups", Action: "POST", Description: "Create alert receiver group"},
		{Name: "更新接收组", Resource: "/api/v1/alerts/receiver-groups/:id", Action: "PUT", Description: "Update alert receiver group"},
		{Name: "删除接收组", Resource: "/api/v1/alerts/receiver-groups/:id", Action: "DELETE", Description: "Delete alert receiver group"},
		{Name: "云到期规则列表", Resource: "/api/v1/alerts/cloud-expiry-rules", Action: "GET", Description: "List cloud expiry rules"},
		{Name: "创建云到期规则", Resource: "/api/v1/alerts/cloud-expiry-rules", Action: "POST", Description: "Create cloud expiry rule"},
		{Name: "更新云到期规则", Resource: "/api/v1/alerts/cloud-expiry-rules/:id", Action: "PUT", Description: "Update cloud expiry rule"},
		{Name: "删除云到期规则", Resource: "/api/v1/alerts/cloud-expiry-rules/:id", Action: "DELETE", Description: "Delete cloud expiry rule"},
		{Name: "立即评估云到期", Resource: "/api/v1/alerts/cloud-expiry-rules/evaluate-now", Action: "POST", Description: "Evaluate cloud expiry rules now"},
		{Name: "登录日志列表", Resource: "/api/v1/login-logs", Action: "GET", Description: "View login logs"},
		{Name: "导出登录日志", Resource: "/api/v1/login-logs/export", Action: "GET", Description: "Export login logs to Excel"},
		{Name: "删除登录日志", Resource: "/api/v1/login-logs/:id", Action: "DELETE", Description: "Delete login log"},
		{Name: "批量删除登录日志", Resource: "/api/v1/login-logs/delete", Action: "POST", Description: "Batch delete login logs"},
		{Name: "操作历史列表", Resource: "/api/v1/operation-logs", Action: "GET", Description: "View operation logs"},
		{Name: "导出操作历史", Resource: "/api/v1/operation-logs/export", Action: "GET", Description: "Export operation logs to Excel"},
		{Name: "删除操作历史", Resource: "/api/v1/operation-logs/:id", Action: "DELETE", Description: "Delete operation log"},
		{Name: "批量删除操作历史", Resource: "/api/v1/operation-logs/delete", Action: "POST", Description: "Batch delete operation logs"},
		{Name: "查看封禁 IP 列表", Resource: "/api/v1/security/banned-ips", Action: "GET", Description: "View banned IPs list"},
		{Name: "解除封禁 IP", Resource: "/api/v1/security/banned-ips/unban", Action: "POST", Description: "Unban IP"},
		{Name: "总览页面", Resource: "/api/v1/overview", Action: "GET", Description: "Get system overview metrics"},
		{Name: "总览项目上线统计", Resource: "/api/v1/overview/project-launches", Action: "GET", Description: "Get project launch stats for last 30 days"},
		{Name: "总览工单按人统计", Resource: "/api/v1/overview/release-by-person", Action: "GET", Description: "Get release runs by submitter for last 30 days"},
		{Name: "插件列表", Resource: "/api/v1/plugins", Action: "GET", Description: "List enabled plugins"},
		{Name: "集群列表", Resource: "/api/v1/clusters", Action: "GET", Description: "View k8s clusters"},
		{Name: "集群详情", Resource: "/api/v1/clusters/:id", Action: "GET", Description: "View k8s cluster detail"},
		{Name: "创建集群", Resource: "/api/v1/clusters", Action: "POST", Description: "Create k8s cluster"},
		{Name: "更新集群", Resource: "/api/v1/clusters/:id", Action: "PUT", Description: "Update k8s cluster"},
		{Name: "删除集群", Resource: "/api/v1/clusters/:id", Action: "DELETE", Description: "Delete k8s cluster"},
		{Name: "启停集群", Resource: "/api/v1/clusters/:id/status", Action: "PUT", Description: "Enable/disable k8s cluster"},
		{Name: "集群连接状态", Resource: "/api/v1/clusters/:id/status", Action: "GET", Description: "Check k8s cluster status"},
		{Name: "集群命名空间", Resource: "/api/v1/clusters/:id/namespaces", Action: "GET", Description: "List cluster namespaces"},
		{Name: "组件状态列表", Resource: "/api/v1/clusters/:id/component-statuses", Action: "GET", Description: "List control plane component statuses"},
		{Name: "集群 API 资源发现", Resource: "/api/v1/clusters/:id/api-resources", Action: "GET", Description: "Discovery API resources like kubectl api-resources"},
		{Name: "K8s 资源 Watch（SSE）", Resource: "/api/v1/k8s/resource-watch/stream", Action: "GET", Description: "Kubernetes watch streamed as Server-Sent Events [k8s-scope=on]", K8sScopeEnabled: true},
		{Name: "K8s 全局搜索", Resource: "/api/v1/k8s/search", Action: "GET", Description: "Search k8s resources"},
		{Name: "K8s 资源拓扑", Resource: "/api/v1/k8s/topology", Action: "GET", Description: "Get k8s resource topology graph"},
		{Name: "K8s Event 转发规则列表", Resource: "/api/v1/k8s/event-forward/rules", Action: "GET", Description: "List k8s event forward rules"},
		{Name: "K8s Event 转发规则详情", Resource: "/api/v1/k8s/event-forward/rules/:id", Action: "GET", Description: "Get k8s event forward rule"},
		{Name: "K8s Event 转发规则创建", Resource: "/api/v1/k8s/event-forward/rules", Action: "POST", Description: "Create k8s event forward rule"},
		{Name: "K8s Event 转发规则更新", Resource: "/api/v1/k8s/event-forward/rules/:id", Action: "PUT", Description: "Update k8s event forward rule"},
		{Name: "K8s Event 转发规则删除", Resource: "/api/v1/k8s/event-forward/rules/:id", Action: "DELETE", Description: "Delete k8s event forward rule"},
		{Name: "K8s Event 转发 Worker 参数", Resource: "/api/v1/k8s/event-forward/settings", Action: "GET", Description: "Get k8s event forward worker settings"},
		{Name: "K8s Event 转发 Worker 更新", Resource: "/api/v1/k8s/event-forward/settings", Action: "PUT", Description: "Update k8s event forward worker settings"},
		{Name: "Pod 列表", Resource: "/api/v1/pods", Action: "GET", Description: "List pods"},
		{Name: "Pod 详情", Resource: "/api/v1/pods/detail", Action: "GET", Description: "Get pod detail"},
		{Name: "Pod 诊断", Resource: "/api/v1/pods/diagnose", Action: "GET", Description: "Diagnose pod issues"},
		{Name: "Pod 事件", Resource: "/api/v1/pods/events", Action: "GET", Description: "List pod events"},
		{Name: "Pod 日志", Resource: "/api/v1/pods/logs", Action: "GET", Description: "Get pod logs"},
		{Name: "Pod 日志下载", Resource: "/api/v1/pods/logs/download", Action: "GET", Description: "Download pod logs"},
		{Name: "Pod 日志流", Resource: "/api/v1/pods/logs/stream", Action: "GET", Description: "Stream pod logs"},
		{Name: "Pod 文件列表", Resource: "/api/v1/pods/files", Action: "GET", Description: "List pod files"},
		{Name: "Pod 文件读取", Resource: "/api/v1/pods/file", Action: "GET", Description: "Read pod file content"},
		{Name: "Pod 文件下载", Resource: "/api/v1/pods/file/download", Action: "GET", Description: "Download pod file"},
		{Name: "Pod 文件上传", Resource: "/api/v1/pods/file/upload", Action: "POST", Description: "Upload file to pod"},
		{Name: "Pod 文件删除", Resource: "/api/v1/pods/file/delete", Action: "POST", Description: "Delete pod file"},
		{Name: "Pod Exec", Resource: "/api/v1/pods/exec", Action: "POST", Description: "Exec command in pod"},
		{Name: "Pod 交互式终端", Resource: "/api/v1/pods/exec/ws", Action: "GET", Description: "Interactive exec terminal via websocket"},
		{Name: "Pod 重启", Resource: "/api/v1/pods/restart", Action: "POST", Description: "Restart pod"},
		{Name: "Pod YAML 创建", Resource: "/api/v1/pods/create/yaml", Action: "POST", Description: "Create pod by yaml"},
		{Name: "Pod 快捷创建", Resource: "/api/v1/pods/create/simple", Action: "POST", Description: "Create pod quickly"},
		{Name: "编辑并重建 Pod", Resource: "/api/v1/pods/update/simple", Action: "POST", Description: "Update pod by recreate"},
		{Name: "删除 Pod", Resource: "/api/v1/pods", Action: "DELETE", Description: "Delete pod"},
		{Name: "命名空间列表", Resource: "/api/v1/namespaces", Action: "GET", Description: "List namespaces"},
		{Name: "命名空间详情", Resource: "/api/v1/namespaces/detail", Action: "GET", Description: "Get namespace detail"},
		{Name: "命名空间应用 YAML", Resource: "/api/v1/namespaces/apply", Action: "POST", Description: "Apply namespace yaml"},
		{Name: "删除命名空间", Resource: "/api/v1/namespaces", Action: "DELETE", Description: "Delete namespace"},
		{Name: "Node 列表", Resource: "/api/v1/nodes", Action: "GET", Description: "List nodes"},
		{Name: "Node 详情", Resource: "/api/v1/nodes/detail", Action: "GET", Description: "Get node detail"},
		{Name: "Node 调度状态", Resource: "/api/v1/nodes/schedulability", Action: "POST", Description: "Cordon or uncordon node"},
		{Name: "Node 污点", Resource: "/api/v1/nodes/taints", Action: "PUT", Description: "Replace node taints"},
		{Name: "Node Drain", Resource: "/api/v1/nodes/drain", Action: "POST", Description: "Drain node workloads"},
		{Name: "Node Drain 进度", Resource: "/api/v1/nodes/drain-status", Action: "GET", Description: "Get node drain status"},

		{Name: "RBAC Role 列表", Resource: "/api/v1/rbac/roles", Action: "GET", Description: "List Roles"},
		{Name: "RBAC RoleBinding 列表", Resource: "/api/v1/rbac/rolebindings", Action: "GET", Description: "List RoleBindings"},
		{Name: "RBAC ClusterRole 列表", Resource: "/api/v1/rbac/clusterroles", Action: "GET", Description: "List ClusterRoles"},
		{Name: "RBAC ClusterRoleBinding 列表", Resource: "/api/v1/rbac/clusterrolebindings", Action: "GET", Description: "List ClusterRoleBindings"},
		{Name: "RBAC 详情", Resource: "/api/v1/rbac/detail", Action: "GET", Description: "Get RBAC detail"},
		{Name: "RBAC 应用 YAML", Resource: "/api/v1/rbac/apply", Action: "POST", Description: "Apply RBAC yaml"},
		{Name: "RBAC 删除", Resource: "/api/v1/rbac", Action: "DELETE", Description: "Delete RBAC resource"},
		{Name: "ServiceAccount 列表", Resource: "/api/v1/serviceaccounts", Action: "GET", Description: "List serviceaccounts"},
		{Name: "ServiceAccount 详情", Resource: "/api/v1/serviceaccounts/detail", Action: "GET", Description: "Get serviceaccount detail"},
		{Name: "ServiceAccount 应用 YAML", Resource: "/api/v1/serviceaccounts/apply", Action: "POST", Description: "Apply serviceaccount yaml"},
		{Name: "ServiceAccount 删除", Resource: "/api/v1/serviceaccounts", Action: "DELETE", Description: "Delete serviceaccount"},

		{Name: "Deployment 列表", Resource: "/api/v1/deployments", Action: "GET", Description: "List deployments"},
		{Name: "Deployment 详情", Resource: "/api/v1/deployments/detail", Action: "GET", Description: "Get deployment detail"},
		{Name: "Deployment 应用 YAML", Resource: "/api/v1/deployments/apply", Action: "POST", Description: "Apply deployment yaml"},
		{Name: "Deployment 扩缩容", Resource: "/api/v1/deployments/scale", Action: "POST", Description: "Scale deployment"},
		{Name: "Deployment 垂直扩缩", Resource: "/api/v1/deployments/container-resources", Action: "POST", Description: "Patch deployment container resources"},
		{Name: "Deployment 重启", Resource: "/api/v1/deployments/restart", Action: "POST", Description: "Restart deployment"},
		{Name: "Deployment 回滚", Resource: "/api/v1/deployments/rollout-undo", Action: "POST", Description: "Undo deployment rollout"},
		{Name: "Deployment 关联 Pods", Resource: "/api/v1/deployments/pods", Action: "GET", Description: "List deployment related pods"},
		{Name: "Deployment 发布状态", Resource: "/api/v1/deployments/rollout-status", Action: "GET", Description: "Get deployment rollout status"},
		{Name: "删除 Deployment", Resource: "/api/v1/deployments", Action: "DELETE", Description: "Delete deployment"},

		{Name: "StatefulSet 列表", Resource: "/api/v1/statefulsets", Action: "GET", Description: "List statefulsets"},
		{Name: "StatefulSet 详情", Resource: "/api/v1/statefulsets/detail", Action: "GET", Description: "Get statefulset detail"},
		{Name: "StatefulSet 应用 YAML", Resource: "/api/v1/statefulsets/apply", Action: "POST", Description: "Apply statefulset yaml"},
		{Name: "StatefulSet 扩缩容", Resource: "/api/v1/statefulsets/scale", Action: "POST", Description: "Scale statefulset"},
		{Name: "StatefulSet 垂直扩缩", Resource: "/api/v1/statefulsets/container-resources", Action: "POST", Description: "Patch statefulset container resources"},
		{Name: "StatefulSet 重启", Resource: "/api/v1/statefulsets/restart", Action: "POST", Description: "Restart statefulset"},
		{Name: "StatefulSet 回滚", Resource: "/api/v1/statefulsets/rollout-undo", Action: "POST", Description: "Undo statefulset rollout"},
		{Name: "StatefulSet 关联 Pods", Resource: "/api/v1/statefulsets/pods", Action: "GET", Description: "List statefulset related pods"},
		{Name: "删除 StatefulSet", Resource: "/api/v1/statefulsets", Action: "DELETE", Description: "Delete statefulset"},

		{Name: "DaemonSet 垂直扩缩", Resource: "/api/v1/daemonsets/container-resources", Action: "POST", Description: "Patch daemonset container resources"},
		{Name: "DaemonSet 列表", Resource: "/api/v1/daemonsets", Action: "GET", Description: "List daemonsets"},
		{Name: "DaemonSet 详情", Resource: "/api/v1/daemonsets/detail", Action: "GET", Description: "Get daemonset detail"},
		{Name: "DaemonSet 应用 YAML", Resource: "/api/v1/daemonsets/apply", Action: "POST", Description: "Apply daemonset yaml"},
		{Name: "DaemonSet 重启", Resource: "/api/v1/daemonsets/restart", Action: "POST", Description: "Restart daemonset"},
		{Name: "DaemonSet 关联 Pods", Resource: "/api/v1/daemonsets/pods", Action: "GET", Description: "List daemonset related pods"},
		{Name: "删除 DaemonSet", Resource: "/api/v1/daemonsets", Action: "DELETE", Description: "Delete daemonset"},

		{Name: "Job 列表", Resource: "/api/v1/jobs", Action: "GET", Description: "List jobs"},
		{Name: "Job 详情", Resource: "/api/v1/jobs/detail", Action: "GET", Description: "Get job detail"},
		{Name: "Job 关联 Pods", Resource: "/api/v1/jobs/pods", Action: "GET", Description: "List job related pods"},
		{Name: "Job 重新执行", Resource: "/api/v1/jobs/rerun", Action: "POST", Description: "Rerun a job"},
		{Name: "Job 垂直扩缩", Resource: "/api/v1/jobs/container-resources", Action: "POST", Description: "Patch job container resources"},
		{Name: "Job 应用 YAML", Resource: "/api/v1/jobs/apply", Action: "POST", Description: "Apply job yaml"},
		{Name: "删除 Job", Resource: "/api/v1/jobs", Action: "DELETE", Description: "Delete job"},

		{Name: "CronJob 列表", Resource: "/api/v1/cronjobs", Action: "GET", Description: "List cronjobs"},
		{Name: "CronJob 列表V2", Resource: "/api/v1/cronjobs/v2", Action: "GET", Description: "List cronjobs with suspend and last schedule"},
		{Name: "CronJob 详情", Resource: "/api/v1/cronjobs/detail", Action: "GET", Description: "Get cronjob detail"},
		{Name: "CronJob 关联 Pods", Resource: "/api/v1/cronjobs/pods", Action: "GET", Description: "List cronjob related pods"},
		{Name: "CronJob 应用 YAML", Resource: "/api/v1/cronjobs/apply", Action: "POST", Description: "Apply cronjob yaml"},
		{Name: "CronJob 垂直扩缩", Resource: "/api/v1/cronjobs/container-resources", Action: "POST", Description: "Patch cronjob container resources"},
		{Name: "CronJob 暂停/恢复", Resource: "/api/v1/cronjobs/suspend", Action: "POST", Description: "Suspend/resume cronjob"},
		{Name: "CronJob 触发执行", Resource: "/api/v1/cronjobs/trigger", Action: "POST", Description: "Trigger cronjob once"},
		{Name: "删除 CronJob", Resource: "/api/v1/cronjobs", Action: "DELETE", Description: "Delete cronjob"},

		{Name: "ConfigMap 列表", Resource: "/api/v1/configmaps", Action: "GET", Description: "List configmaps"},
		{Name: "ConfigMap 详情", Resource: "/api/v1/configmaps/detail", Action: "GET", Description: "Get configmap detail"},
		{Name: "ConfigMap 应用 YAML", Resource: "/api/v1/configmaps/apply", Action: "POST", Description: "Apply configmap yaml"},
		{Name: "删除 ConfigMap", Resource: "/api/v1/configmaps", Action: "DELETE", Description: "Delete configmap"},

		{Name: "Secret 列表", Resource: "/api/v1/secrets", Action: "GET", Description: "List secrets"},
		{Name: "Secret 详情", Resource: "/api/v1/secrets/detail", Action: "GET", Description: "Get secret detail"},
		{Name: "Secret 应用 YAML", Resource: "/api/v1/secrets/apply", Action: "POST", Description: "Apply secret yaml"},
		{Name: "删除 Secret", Resource: "/api/v1/secrets", Action: "DELETE", Description: "Delete secret"},

		{Name: "Service 列表", Resource: "/api/v1/k8s-services", Action: "GET", Description: "List services"},
		{Name: "Service 详情", Resource: "/api/v1/k8s-services/detail", Action: "GET", Description: "Get service detail"},
		{Name: "Service 应用 YAML", Resource: "/api/v1/k8s-services/apply", Action: "POST", Description: "Apply service yaml"},
		{Name: "删除 Service", Resource: "/api/v1/k8s-services", Action: "DELETE", Description: "Delete service"},

		{Name: "PersistentVolume 列表", Resource: "/api/v1/persistentvolumes", Action: "GET", Description: "List persistent volumes"},
		{Name: "PersistentVolume 详情", Resource: "/api/v1/persistentvolumes/detail", Action: "GET", Description: "Get persistent volume detail"},
		{Name: "PersistentVolume 应用 YAML", Resource: "/api/v1/persistentvolumes/apply", Action: "POST", Description: "Apply persistent volume yaml"},
		{Name: "删除 PersistentVolume", Resource: "/api/v1/persistentvolumes", Action: "DELETE", Description: "Delete persistent volume"},

		{Name: "PersistentVolumeClaim 列表", Resource: "/api/v1/persistentvolumeclaims", Action: "GET", Description: "List persistent volume claims"},
		{Name: "PersistentVolumeClaim 详情", Resource: "/api/v1/persistentvolumeclaims/detail", Action: "GET", Description: "Get persistent volume claim detail"},
		{Name: "PersistentVolumeClaim 应用 YAML", Resource: "/api/v1/persistentvolumeclaims/apply", Action: "POST", Description: "Apply persistent volume claim yaml"},
		{Name: "删除 PersistentVolumeClaim", Resource: "/api/v1/persistentvolumeclaims", Action: "DELETE", Description: "Delete persistent volume claim"},

		{Name: "StorageClass 列表", Resource: "/api/v1/storageclasses", Action: "GET", Description: "List storage classes"},
		{Name: "StorageClass 详情", Resource: "/api/v1/storageclasses/detail", Action: "GET", Description: "Get storage class detail"},
		{Name: "StorageClass 应用 YAML", Resource: "/api/v1/storageclasses/apply", Action: "POST", Description: "Apply storage class yaml"},
		{Name: "删除 StorageClass", Resource: "/api/v1/storageclasses", Action: "DELETE", Description: "Delete storage class"},

		{Name: "Ingress 列表", Resource: "/api/v1/ingresses", Action: "GET", Description: "List ingresses"},
		{Name: "Ingress 详情", Resource: "/api/v1/ingresses/detail", Action: "GET", Description: "Get ingress detail"},
		{Name: "Ingress 诊断", Resource: "/api/v1/ingresses/diagnose", Action: "GET", Description: "Diagnose ingress issues"},
		{Name: "Ingress 应用 YAML", Resource: "/api/v1/ingresses/apply", Action: "POST", Description: "Apply ingress yaml"},
		{Name: "IngressClass 列表", Resource: "/api/v1/ingresses/classes", Action: "GET", Description: "List ingress classes"},
		{Name: "IngressClass 详情", Resource: "/api/v1/ingresses/classes/detail", Action: "GET", Description: "Get ingress class detail"},
		{Name: "IngressClass 应用 YAML", Resource: "/api/v1/ingresses/classes/apply", Action: "POST", Description: "Apply ingress class yaml"},
		{Name: "删除 IngressClass", Resource: "/api/v1/ingresses/classes", Action: "DELETE", Description: "Delete ingress class"},
		{Name: "重启 Ingress-Nginx Pods", Resource: "/api/v1/ingresses/nginx/restart", Action: "POST", Description: "Restart ingress-nginx controller pods (requires cluster admin tier + confirm=true)"},
		{Name: "删除 Ingress", Resource: "/api/v1/ingresses", Action: "DELETE", Description: "Delete ingress"},
		{Name: "HPA 列表", Resource: "/api/v1/horizontal-pod-autoscalers", Action: "GET", Description: "List HorizontalPodAutoscaler"},
		{Name: "HPA 详情", Resource: "/api/v1/horizontal-pod-autoscalers/detail", Action: "GET", Description: "Get HPA YAML"},
		{Name: "HPA 应用 YAML", Resource: "/api/v1/horizontal-pod-autoscalers/apply", Action: "POST", Description: "Apply HPA yaml"},
		{Name: "删除 HPA", Resource: "/api/v1/horizontal-pod-autoscalers", Action: "DELETE", Description: "Delete HPA"},

		{Name: "Harbor 信息", Resource: "/api/v1/helm/harbor/info", Action: "GET", Description: "Get Harbor helm repo info"},
		{Name: "Harbor Chart 列表", Resource: "/api/v1/helm/harbor/charts", Action: "GET", Description: "List Harbor helm charts"},
		{Name: "Harbor Chart 版本", Resource: "/api/v1/helm/harbor/charts/versions", Action: "GET", Description: "List Harbor chart versions"},
		{Name: "Helm Release 列表", Resource: "/api/v1/helm/releases", Action: "GET", Description: "List helm releases"},
		{Name: "Helm Release 详情", Resource: "/api/v1/helm/releases/detail", Action: "GET", Description: "Get helm release detail"},
		{Name: "Helm Release 历史", Resource: "/api/v1/helm/releases/history", Action: "GET", Description: "Get helm release history"},
		{Name: "Helm Release Values", Resource: "/api/v1/helm/releases/values", Action: "GET", Description: "Get helm release values"},
		{Name: "Helm 安装", Resource: "/api/v1/helm/releases/install", Action: "POST", Description: "Install helm release from Harbor [k8s-scope=on]", K8sScopeEnabled: true},
		{Name: "Helm 升级", Resource: "/api/v1/helm/releases/upgrade", Action: "POST", Description: "Upgrade helm release [k8s-scope=on]", K8sScopeEnabled: true},
		{Name: "Helm 回滚", Resource: "/api/v1/helm/releases/rollback", Action: "POST", Description: "Rollback helm release [k8s-scope=on]", K8sScopeEnabled: true},
		{Name: "Helm 卸载", Resource: "/api/v1/helm/releases", Action: "DELETE", Description: "Uninstall helm release [k8s-scope=on]", K8sScopeEnabled: true},
		{Name: "网络策略列表", Resource: "/api/v1/network-policies", Action: "GET", Description: "List network policies"},
		{Name: "网络策略详情", Resource: "/api/v1/network-policies/detail", Action: "GET", Description: "Get network policy detail"},
		{Name: "网络策略应用 YAML", Resource: "/api/v1/network-policies/apply", Action: "POST", Description: "Apply network policy yaml"},
		{Name: "删除网络策略", Resource: "/api/v1/network-policies", Action: "DELETE", Description: "Delete network policy"},

		{Name: "项目列表", Resource: "/api/v1/projects", Action: "GET", Description: "List projects"},
		{Name: "创建项目", Resource: "/api/v1/projects", Action: "POST", Description: "Create project"},
		{Name: "更新项目", Resource: "/api/v1/projects/:id", Action: "PUT", Description: "Update project"},
		{Name: "删除项目", Resource: "/api/v1/projects/:id", Action: "DELETE", Description: "Delete project"},
		{Name: "项目成员列表", Resource: "/api/v1/projects/:id/members", Action: "GET", Description: "List project members"},
		{Name: "添加项目成员", Resource: "/api/v1/projects/:id/members", Action: "POST", Description: "Add project member"},
		{Name: "更新项目成员", Resource: "/api/v1/projects/:id/members/:memberId", Action: "PUT", Description: "Update project member"},
		{Name: "移除项目成员", Resource: "/api/v1/projects/:id/members/:memberId", Action: "DELETE", Description: "Remove project member"},
		{Name: "项目服务器列表", Resource: "/api/v1/projects/:id/servers", Action: "GET", Description: "List project servers"},
		{Name: "项目服务器保存", Resource: "/api/v1/projects/:id/servers", Action: "POST", Description: "Upsert project server"},
		{Name: "项目服务器详情", Resource: "/api/v1/projects/:id/servers/:serverId", Action: "GET", Description: "Project server detail"},
		{Name: "项目服务器有效权限", Resource: "/api/v1/projects/:id/servers/:serverId/my-access", Action: "GET", Description: "Current user effective server access"},
		{Name: "删除项目服务器", Resource: "/api/v1/projects/:id/servers/:serverId", Action: "DELETE", Description: "Delete project server"},
		{Name: "项目服务器命令", Resource: "/api/v1/projects/:id/servers/:serverId/exec", Action: "POST", Description: "Exec on project server"},
		{Name: "项目服务器文件列表", Resource: "/api/v1/projects/:id/servers/:serverId/files", Action: "GET", Description: "List remote files on server"},
		{Name: "项目服务器文件上传", Resource: "/api/v1/projects/:id/servers/:serverId/files/upload", Action: "POST", Description: "Upload file to server via SFTP"},
		{Name: "项目服务器文件下载", Resource: "/api/v1/projects/:id/servers/:serverId/files/download", Action: "GET", Description: "Download file from server via SFTP"},
		{Name: "项目服务器文件删除", Resource: "/api/v1/projects/:id/servers/:serverId/files/delete", Action: "POST", Description: "Delete remote file on server"},
		{Name: "项目服务器云操作", Resource: "/api/v1/projects/:id/servers/:serverId/cloud-actions", Action: "POST", Description: "Run cloud provider action on server"},
		{Name: "项目服务器分组树", Resource: "/api/v1/projects/:id/server-groups/tree", Action: "GET", Description: "List server groups tree"},
		{Name: "项目服务器分组创建", Resource: "/api/v1/projects/:id/server-groups", Action: "POST", Description: "Upsert server group"},
		{Name: "项目服务器分组更新", Resource: "/api/v1/projects/:id/server-groups/:groupId", Action: "PUT", Description: "Update server group"},
		{Name: "项目服务器分组删除", Resource: "/api/v1/projects/:id/server-groups/:groupId", Action: "DELETE", Description: "Delete server group"},
		{Name: "项目云账号列表", Resource: "/api/v1/projects/:id/cloud-accounts", Action: "GET", Description: "List cloud accounts"},
		{Name: "项目云账号保存", Resource: "/api/v1/projects/:id/cloud-accounts", Action: "POST", Description: "Upsert cloud account"},
		{Name: "项目云账号更新", Resource: "/api/v1/projects/:id/cloud-accounts/:accountId", Action: "PUT", Description: "Update cloud account"},
		{Name: "项目云账号同步", Resource: "/api/v1/projects/:id/cloud-accounts/:accountId/sync", Action: "PUT", Description: "Sync cloud account"},
		{Name: "项目云账号删除", Resource: "/api/v1/projects/:id/cloud-accounts/:accountId", Action: "DELETE", Description: "Delete cloud account"},
		{Name: "项目服务器导入", Resource: "/api/v1/projects/:id/servers/import", Action: "POST", Description: "Import servers"},
		{Name: "项目服务器导入模板", Resource: "/api/v1/projects/:id/servers/import-template", Action: "GET", Description: "Servers import template"},
		{Name: "项目服务器导出", Resource: "/api/v1/projects/:id/servers/export", Action: "GET", Description: "Export servers"},
		{Name: "项目服务器连通测试", Resource: "/api/v1/projects/:id/servers/test", Action: "POST", Description: "Test server connection"},
		{Name: "项目服务器批量测试", Resource: "/api/v1/projects/:id/servers/test/batch", Action: "POST", Description: "Batch test servers"},
		{Name: "项目服务器同步", Resource: "/api/v1/projects/:id/servers/sync", Action: "POST", Description: "Sync servers"},
		{Name: "服务器授权列表", Resource: "/api/v1/projects/:id/server-access-grants", Action: "GET", Description: "List server access grants"},
		{Name: "服务器授权保存", Resource: "/api/v1/projects/:id/server-access-grants", Action: "POST", Description: "Upsert server access grant"},
		{Name: "服务器授权批量", Resource: "/api/v1/projects/:id/server-access-grants/bulk", Action: "POST", Description: "Bulk upsert server access grants"},
		{Name: "服务器授权迁移", Resource: "/api/v1/projects/:id/server-access-grants/bootstrap", Action: "POST", Description: "Bootstrap server grants for members"},
		{Name: "服务器授权删除", Resource: "/api/v1/projects/:id/server-access-grants/:grantId", Action: "DELETE", Description: "Delete server access grant"},
		{Name: "项目服务列表", Resource: "/api/v1/projects/:id/services", Action: "GET", Description: "List project services"},
		{Name: "项目服务保存", Resource: "/api/v1/projects/:id/services", Action: "POST", Description: "Upsert project service"},
		{Name: "删除项目服务", Resource: "/api/v1/projects/:id/services/:serviceId", Action: "DELETE", Description: "Delete project service"},
		{Name: "服务目录列表", Resource: "/api/v1/projects/:id/service-catalog", Action: "GET", Description: "List service catalog"},
		{Name: "服务目录保存", Resource: "/api/v1/projects/:id/service-catalog", Action: "POST", Description: "Upsert service catalog"},
		{Name: "服务目录详情", Resource: "/api/v1/projects/:id/service-catalog/:catalogId", Action: "GET", Description: "Get service catalog"},
		{Name: "服务画像", Resource: "/api/v1/projects/:id/service-catalog/:catalogId/portrait", Action: "GET", Description: "Service portrait aggregate"},
		{Name: "服务目录删除", Resource: "/api/v1/projects/:id/service-catalog/:catalogId", Action: "DELETE", Description: "Delete service catalog"},
		{Name: "服务目录绑定", Resource: "/api/v1/projects/:id/service-catalog/:catalogId/links", Action: "POST", Description: "Bind service catalog link"},
		{Name: "服务目录解绑", Resource: "/api/v1/projects/:id/service-catalog/:catalogId/links/:linkId", Action: "DELETE", Description: "Unbind service catalog link"},
		{Name: "告警质量报告", Resource: "/api/v1/alerts/quality-report", Action: "GET", Description: "Alert quality report"},
		{Name: "发布后验证", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/verify", Action: "POST", Description: "Verify release run"},
		{Name: "工作负载预检", Resource: "/api/v1/deployments/preview-apply", Action: "POST", Description: "Preview workload apply"},
		{Name: "工作负载快照", Resource: "/api/v1/deployments/snapshots", Action: "GET", Description: "List workload snapshots"},
		{Name: "工作负载回滚", Resource: "/api/v1/deployments/snapshots/rollback", Action: "POST", Description: "Rollback workload snapshot"},
		{Name: "项目日志源列表", Resource: "/api/v1/projects/:id/log-sources", Action: "GET", Description: "List log sources"},
		{Name: "项目日志源保存", Resource: "/api/v1/projects/:id/log-sources", Action: "POST", Description: "Upsert log source"},
		{Name: "删除项目日志源", Resource: "/api/v1/projects/:id/log-sources/:logSourceId", Action: "DELETE", Description: "Delete log source"},
		{Name: "项目日志检索", Resource: "/api/v1/projects/:id/logs/search", Action: "GET", Description: "Search project logs from Elasticsearch"},
		{Name: "项目日志导出", Resource: "/api/v1/projects/:id/logs/export", Action: "GET", Description: "Export project logs from Elasticsearch"},
		{Name: "日志保留策略查询", Resource: "/api/v1/log-platform/retention", Action: "GET", Description: "Get global log retention policy"},
		{Name: "日志保留策略保存", Resource: "/api/v1/log-platform/retention", Action: "PUT", Description: "Upsert global log retention policy"},
		{Name: "日志保留策略列表", Resource: "/api/v1/log-platform/retention/list", Action: "GET", Description: "List log retention policies"},
		{Name: "ES 存储概览", Resource: "/api/v1/log-platform/es-storage", Action: "GET", Description: "Elasticsearch storage stats"},
		{Name: "ES 索引删除", Resource: "/api/v1/log-platform/es-indices/:index", Action: "DELETE", Description: "Delete elasticsearch index"},
		{Name: "Kafka 队列观测", Resource: "/api/v1/log-platform/kafka-stats", Action: "GET", Description: "Kafka lag and consumer stats"},
		{Name: "Kafka 配置预览", Resource: "/api/v1/log-platform/kafka-config", Action: "GET", Description: "Preview kafka config from dict"},
		{Name: "Kafka Topic 删除", Resource: "/api/v1/log-platform/kafka-topics/:topic", Action: "DELETE", Description: "Delete agent kafka topic"},
		{Name: "日志保留手动清理", Resource: "/api/v1/log-platform/retention/cleanup", Action: "POST", Description: "Run log retention cleanup"},
		{Name: "项目日志保留查询", Resource: "/api/v1/projects/:id/log-retention", Action: "GET", Description: "Get project log retention policy"},
		{Name: "项目日志保留保存", Resource: "/api/v1/projects/:id/log-retention", Action: "PUT", Description: "Upsert project log retention policy"},
		{Name: "项目日志保留删除", Resource: "/api/v1/projects/:id/log-retention", Action: "DELETE", Description: "Delete project log retention override"},
		{Name: "Loggie 心跳上报", Resource: "/api/v1/loggie/heartbeat/report", Action: "POST", Description: "Loggie agent heartbeat"},
		{Name: "Loggie 引导", Resource: "/api/v1/projects/:id/loggie/bootstrap", Action: "POST", Description: "Bootstrap loggie agent token"},
		{Name: "Loggie 状态列表", Resource: "/api/v1/projects/:id/loggie/status", Action: "GET", Description: "List loggie status by server"},
		{Name: "Loggie 引导日志源预览", Resource: "/api/v1/projects/:id/loggie/bootstrap-sources", Action: "GET", Description: "Preview log sources for loggie bootstrap"},
		{Name: "Loggie 配置下发", Resource: "/api/v1/projects/:id/loggie/deploy", Action: "POST", Description: "Deploy loggie pipeline over SSH"},
		{Name: "Loggie 安装", Resource: "/api/v1/projects/:id/loggie/install", Action: "POST", Description: "Install loggie agent over SSH"},
		{Name: "Loggie 卸载", Resource: "/api/v1/projects/:id/loggie/uninstall", Action: "POST", Description: "Uninstall loggie agent and clear registration"},
		{Name: "Loggie 启动", Resource: "/api/v1/projects/:id/loggie/start", Action: "POST", Description: "Start loggie service over SSH"},
		{Name: "Loggie 停止", Resource: "/api/v1/projects/:id/loggie/stop", Action: "POST", Description: "Stop loggie service over SSH"},
		{Name: "Loggie 重启", Resource: "/api/v1/projects/:id/loggie/restart", Action: "POST", Description: "Restart loggie service over SSH"},
		{Name: "Loggie 同步下发", Resource: "/api/v1/projects/:id/loggie/sync", Action: "POST", Description: "Sync pipelines from log sources and deploy"},
		{Name: "集群日志规则列表", Resource: "/api/v1/projects/:id/cluster-log/rules", Action: "GET", Description: "List K8s cluster log rules"},
		{Name: "创建集群日志规则", Resource: "/api/v1/projects/:id/cluster-log/rules", Action: "POST", Description: "Create K8s cluster log rule"},
		{Name: "更新集群日志规则", Resource: "/api/v1/projects/:id/cluster-log/rules/:rule_id", Action: "PUT", Description: "Update K8s cluster log rule"},
		{Name: "删除集群日志规则", Resource: "/api/v1/projects/:id/cluster-log/rules/:rule_id", Action: "DELETE", Description: "Delete K8s cluster log rule"},
		{Name: "集群日志 Agent 列表", Resource: "/api/v1/projects/:id/cluster-log/agents", Action: "GET", Description: "List K8s cluster log agents"},
		{Name: "集群日志 Pipeline 预览", Resource: "/api/v1/projects/:id/cluster-log/pipelines/preview", Action: "GET", Description: "Preview K8s cluster pipelines.yml"},
		{Name: "保存集群日志 Pipeline", Resource: "/api/v1/projects/:id/cluster-log/pipelines", Action: "PUT", Description: "Save or reset custom K8s cluster pipelines.yml"},
		{Name: "部署集群日志采集", Resource: "/api/v1/projects/:id/cluster-log/deploy", Action: "POST", Description: "Deploy or sync Loggie DaemonSet"},
		{Name: "集群日志采集状态", Resource: "/api/v1/projects/:id/cluster-log/status", Action: "GET", Description: "Refresh K8s cluster log DaemonSet status"},
		{Name: "Loggie 配置下载", Resource: "/api/v1/projects/:id/loggie/pipeline/download", Action: "GET", Description: "Download loggie pipeline bundle"},
		{Name: "ES 配置预览", Resource: "/api/v1/log-platform/es-config", Action: "GET", Description: "Preview elasticsearch config from dict"},
		{Name: "项目服务器终端 WS", Resource: "/api/v1/projects/:id/servers/:serverId/terminal/ws", Action: "GET", Description: "Server terminal websocket"},

		{Name: "MySQL备份实例列表", Resource: "/api/v1/projects/:id/mysql-backup/instances", Action: "GET", Description: "List MySQL backup instances"},
		{Name: "MySQL备份实例创建", Resource: "/api/v1/projects/:id/mysql-backup/instances", Action: "POST", Description: "Create MySQL backup instance"},
		{Name: "MySQL备份实例更新", Resource: "/api/v1/projects/:id/mysql-backup/instances/:instanceId", Action: "PUT", Description: "Update MySQL backup instance"},
		{Name: "MySQL备份实例删除", Resource: "/api/v1/projects/:id/mysql-backup/instances/:instanceId", Action: "DELETE", Description: "Delete MySQL backup instance"},
		{Name: "MySQL备份连通测试", Resource: "/api/v1/projects/:id/mysql-backup/instances/:instanceId/ping", Action: "POST", Description: "Ping MySQL backup instance"},
		{Name: "MySQL远端备份检查", Resource: "/api/v1/projects/:id/mysql-backup/instances/:instanceId/check-remote", Action: "POST", Description: "Check remote xtrabackup backup"},
		{Name: "MySQL执行备份", Resource: "/api/v1/projects/:id/mysql-backup/instances/:instanceId/run", Action: "POST", Description: "Run MySQL backup job"},
		{Name: "MySQL备份任务列表", Resource: "/api/v1/projects/:id/mysql-backup/jobs", Action: "GET", Description: "List MySQL backup jobs"},
		{Name: "MySQL备份任务停止", Resource: "/api/v1/projects/:id/mysql-backup/jobs/:jobId/stop", Action: "POST", Description: "Stop running MySQL backup job"},
		{Name: "MySQL备份任务删除", Resource: "/api/v1/projects/:id/mysql-backup/jobs/:jobId", Action: "DELETE", Description: "Delete MySQL backup job record"},
		{Name: "MySQL备份预签名下载", Resource: "/api/v1/projects/:id/mysql-backup/jobs/:jobId/presign", Action: "GET", Description: "Presign MySQL backup download URL"},
		{Name: "MySQL备份Dump选项", Resource: "/api/v1/projects/:id/mysql-backup/mysqldump-options", Action: "GET", Description: "List mysqldump backup options"},

		{Name: "巡检计划查询", Resource: "/api/v1/projects/:id/inspect/plan", Action: "GET", Description: "Get project inspect plan"},
		{Name: "巡检计划更新", Resource: "/api/v1/projects/:id/inspect/plan", Action: "PUT", Description: "Update project inspect plan"},
		{Name: "巡检项列表", Resource: "/api/v1/projects/:id/inspect/items", Action: "GET", Description: "List inspect items"},
		{Name: "巡检项创建", Resource: "/api/v1/projects/:id/inspect/items", Action: "POST", Description: "Create inspect item"},
		{Name: "巡检项更新", Resource: "/api/v1/projects/:id/inspect/items/:itemId", Action: "PUT", Description: "Update inspect item"},
		{Name: "巡检项删除", Resource: "/api/v1/projects/:id/inspect/items/:itemId", Action: "DELETE", Description: "Delete inspect item"},
		{Name: "巡检项同步模板", Resource: "/api/v1/projects/:id/inspect/items/sync-template", Action: "POST", Description: "Sync inspect items from global template"},
		{Name: "巡检项重置模板", Resource: "/api/v1/projects/:id/inspect/items/reset-template", Action: "POST", Description: "Reset inspect items from global template"},
		{Name: "巡检记录列表", Resource: "/api/v1/projects/:id/inspect/runs", Action: "GET", Description: "List inspect runs"},
		{Name: "巡检立即执行", Resource: "/api/v1/projects/:id/inspect/runs", Action: "POST", Description: "Start inspect run"},
		{Name: "巡检记录详情", Resource: "/api/v1/projects/:id/inspect/runs/:runId", Action: "GET", Description: "Get inspect run"},
		{Name: "巡检报告HTML", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.html", Action: "GET", Description: "Download inspect HTML report"},
		{Name: "巡检报告PDF", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.pdf", Action: "GET", Description: "Download inspect PDF report"},
		{Name: "巡检报告Excel", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.xlsx", Action: "GET", Description: "Download inspect Excel report"},
		{Name: "巡检报告打印版", Resource: "/api/v1/projects/:id/inspect/runs/:runId/report.print.html", Action: "GET", Description: "Download inspect print HTML"},
		{Name: "巡检邮件重发", Resource: "/api/v1/projects/:id/inspect/runs/:runId/resend-email", Action: "POST", Description: "Resend inspect report email"},
		{Name: "巡检报告模板列表", Resource: "/api/v1/projects/:id/inspect/report-templates", Action: "GET", Description: "List inspect report templates"},
		{Name: "巡检报告模板创建", Resource: "/api/v1/projects/:id/inspect/report-templates", Action: "POST", Description: "Create inspect report template"},
		{Name: "巡检报告模板更新", Resource: "/api/v1/projects/:id/inspect/report-templates/:templateId", Action: "PUT", Description: "Update inspect report template"},
		{Name: "巡检报告模板删除", Resource: "/api/v1/projects/:id/inspect/report-templates/:templateId", Action: "DELETE", Description: "Delete inspect report template"},
		{Name: "巡检报告模板复制", Resource: "/api/v1/projects/:id/inspect/report-templates/copy", Action: "POST", Description: "Copy inspect report template"},
		{Name: "巡检报告模板预览", Resource: "/api/v1/projects/:id/inspect/report-templates/preview", Action: "POST", Description: "Preview inspect report template"},

		{Name: "数据库实例列表", Resource: "/api/v1/projects/:id/dbmgmt/instances", Action: "GET", Description: "List DB instances"},
		{Name: "数据库实例创建", Resource: "/api/v1/projects/:id/dbmgmt/instances", Action: "POST", Description: "Create DB instance"},
		{Name: "数据库实例详情", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId", Action: "GET", Description: "Get DB instance"},
		{Name: "数据库实例更新", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId", Action: "PUT", Description: "Update DB instance"},
		{Name: "数据库实例删除", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId", Action: "DELETE", Description: "Delete DB instance"},
		{Name: "数据库实例探活", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/ping", Action: "POST", Description: "Ping DB instance"},
		{Name: "数据库元数据库列表", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/databases", Action: "GET", Description: "List databases"},
		{Name: "数据库元数据表列表", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/tables", Action: "GET", Description: "List tables"},
		{Name: "数据库元数据列列表", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/metadata/columns", Action: "GET", Description: "List columns"},
		{Name: "数据库只读查询", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/query", Action: "POST", Description: "Execute read-only SQL"},
		{Name: "数据库写操作", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/execute", Action: "POST", Description: "Execute SQL with approval"},
		{Name: "数据库SQL导入", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/import", Action: "POST", Description: "Import SQL file"},
		{Name: "数据库授权列表", Resource: "/api/v1/projects/:id/dbmgmt/grants", Action: "GET", Description: "List DB grants"},
		{Name: "数据库授权创建", Resource: "/api/v1/projects/:id/dbmgmt/grants", Action: "POST", Description: "Create DB grant"},
		{Name: "数据库授权删除", Resource: "/api/v1/projects/:id/dbmgmt/grants/:grantId", Action: "DELETE", Description: "Delete DB grant"},
		{Name: "数据库授权更新", Resource: "/api/v1/projects/:id/dbmgmt/grants/:grantId", Action: "PUT", Description: "Update DB grant"},
		{Name: "数据库SQL预检", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/check", Action: "POST", Description: "Check SQL via goInception"},
		{Name: "数据库工单详情", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId", Action: "GET", Description: "Get DB SQL ticket detail"},
		{Name: "数据库工单审批步骤", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/steps", Action: "GET", Description: "List DB ticket approval steps"},
		{Name: "数据库工单回滚SQL", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback", Action: "GET", Description: "Get DB ticket rollback SQL"},
		{Name: "数据库工单回滚预览", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/preview", Action: "GET", Description: "Preview DB rollback ticket"},
		{Name: "数据库工单回滚提交", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/rollback/submit", Action: "POST", Description: "Submit DB rollback ticket"},
		{Name: "数据库工单OSC列表", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc", Action: "GET", Description: "List DB ticket OSC jobs"},
		{Name: "数据库工单OSC进度", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1", Action: "GET", Description: "Get DB ticket OSC percent"},
		{Name: "数据库工单OSC控制", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/osc/:sqlsha1/control", Action: "POST", Description: "Control DB ticket OSC job"},
		{Name: "数据库有效权限", Resource: "/api/v1/projects/:id/dbmgmt/grants/effective", Action: "GET", Description: "Get effective DB permission"},
		{Name: "数据库审批流查询", Resource: "/api/v1/projects/:id/dbmgmt/approval-flow", Action: "GET", Description: "Get DB approval flow"},
		{Name: "数据库审批流保存", Resource: "/api/v1/projects/:id/dbmgmt/approval-flow", Action: "PUT", Description: "Upsert DB approval flow"},
		{Name: "数据库权限申请列表", Resource: "/api/v1/projects/:id/dbmgmt/access-requests", Action: "GET", Description: "List DB access requests"},
		{Name: "数据库权限申请创建", Resource: "/api/v1/projects/:id/dbmgmt/access-requests", Action: "POST", Description: "Create DB access request"},
		{Name: "数据库权限申请通过", Resource: "/api/v1/projects/:id/dbmgmt/access-requests/:requestId/approve", Action: "POST", Description: "Approve DB access request"},
		{Name: "数据库权限申请拒绝", Resource: "/api/v1/projects/:id/dbmgmt/access-requests/:requestId/reject", Action: "POST", Description: "Reject DB access request"},
		{Name: "应用用户权限申请列表", Resource: "/api/v1/projects/:id/dbmgmt/app-user-requests", Action: "GET", Description: "List DB app user requests"},
		{Name: "应用用户权限申请创建", Resource: "/api/v1/projects/:id/dbmgmt/app-user-requests", Action: "POST", Description: "Create DB app user request"},
		{Name: "应用用户权限申请通过", Resource: "/api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/approve", Action: "POST", Description: "Approve DB app user request"},
		{Name: "应用用户权限申请拒绝", Resource: "/api/v1/projects/:id/dbmgmt/app-user-requests/:requestId/reject", Action: "POST", Description: "Reject DB app user request"},
		{Name: "实例 MySQL 用户列表", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-users", Action: "GET", Description: "List MySQL users on instance"},
		{Name: "实例 MySQL 用户权限查询", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/mysql-user-privileges", Action: "GET", Description: "Get MySQL user privileges for apply form"},
		{Name: "实例账号密码查看", Resource: "/api/v1/projects/:id/dbmgmt/instances/:instanceId/accounts/:accountId/password", Action: "GET", Description: "Reveal platform-managed account password"},
		{Name: "数据库工单列表", Resource: "/api/v1/projects/:id/dbmgmt/tickets", Action: "GET", Description: "List DB SQL tickets"},
		{Name: "数据库工单审批通过", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/approve", Action: "POST", Description: "Approve DB SQL ticket"},
		{Name: "数据库工单审批拒绝", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/reject", Action: "POST", Description: "Reject DB SQL ticket"},
		{Name: "数据库工单执行", Resource: "/api/v1/projects/:id/dbmgmt/tickets/:ticketId/execute", Action: "POST", Description: "Execute approved DB SQL ticket"},
		{Name: "数据库执行历史", Resource: "/api/v1/projects/:id/dbmgmt/executions", Action: "GET", Description: "List DB SQL executions"},
		{Name: "数据库审计日志", Resource: "/api/v1/projects/:id/dbmgmt/audit-logs", Action: "GET", Description: "List DB audit logs"},

		{Name: "CI/CD 应用服务列表", Resource: "/api/v1/projects/:id/cicd/services", Action: "GET", Description: "List CI/CD services"},
		{Name: "CI/CD 授权列表", Resource: "/api/v1/projects/:id/cicd-access-grants", Action: "GET", Description: "List CI/CD access grants"},
		{Name: "CI/CD 授权保存", Resource: "/api/v1/projects/:id/cicd-access-grants", Action: "POST", Description: "Upsert CI/CD access grant"},
		{Name: "CI/CD 授权批量", Resource: "/api/v1/projects/:id/cicd-access-grants/bulk", Action: "POST", Description: "Bulk upsert CI/CD access grants"},
		{Name: "CI/CD 授权迁移", Resource: "/api/v1/projects/:id/cicd-access-grants/bootstrap", Action: "POST", Description: "Bootstrap CI/CD grants for members"},
		{Name: "CI/CD 授权删除", Resource: "/api/v1/projects/:id/cicd-access-grants/:grantId", Action: "DELETE", Description: "Delete CI/CD access grant"},
		{Name: "CI/CD 创建应用服务", Resource: "/api/v1/projects/:id/cicd/services", Action: "POST", Description: "Create CI/CD service"},
		{Name: "CI/CD 应用服务详情", Resource: "/api/v1/projects/:id/cicd/services/:serviceId", Action: "GET", Description: "Get CI/CD service"},
		{Name: "CI/CD 更新应用服务", Resource: "/api/v1/projects/:id/cicd/services/:serviceId", Action: "PUT", Description: "Update CI/CD service"},
		{Name: "CI/CD 删除应用服务", Resource: "/api/v1/projects/:id/cicd/services/:serviceId", Action: "DELETE", Description: "Delete CI/CD service"},
		{Name: "CI/CD CI 配置查询", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/ci-config", Action: "GET", Description: "Get CI config"},
		{Name: "CI/CD CI 配置保存", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/ci-config", Action: "PUT", Description: "Upsert CI config"},
		{Name: "CI/CD 发布配置列表", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs", Action: "GET", Description: "List deploy configs"},
		{Name: "CI/CD 创建发布配置", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs", Action: "POST", Description: "Create deploy config"},
		{Name: "CI/CD 更新发布配置", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId", Action: "PUT", Description: "Update deploy config"},
		{Name: "CI/CD 删除发布配置", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/deploy-configs/:configId", Action: "DELETE", Description: "Delete deploy config"},
		{Name: "CI/CD MinIO 制品列表", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/artifacts", Action: "GET", Description: "List MinIO artifacts"},
		{Name: "CI/CD 下载 Helm 脚手架", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/helm-scaffold", Action: "GET", Description: "Download Helm chart scaffold zip"},
		{Name: "CI/CD 预览 Helm 脚手架", Resource: "/api/v1/projects/:id/cicd/helm-scaffold", Action: "GET", Description: "Preview download Helm chart scaffold zip"},
		{Name: "CI/CD 触发 CI 构建", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/builds", Action: "POST", Description: "Trigger CI build"},
		{Name: "CI/CD 触发 CD 发布", Resource: "/api/v1/projects/:id/cicd/services/:serviceId/releases", Action: "POST", Description: "Trigger CD release"},
		{Name: "CI/CD CI 打包记录列表", Resource: "/api/v1/projects/:id/cicd/build-runs", Action: "GET", Description: "List CI build runs"},
		{Name: "CI/CD CI 打包记录详情", Resource: "/api/v1/projects/:id/cicd/build-runs/:runId", Action: "GET", Description: "Get CI build run"},
		{Name: "CI/CD CI 构建日志", Resource: "/api/v1/projects/:id/cicd/build-runs/:runId/log", Action: "GET", Description: "Get CI build log"},
		{Name: "CI/CD 删除 CI 打包记录", Resource: "/api/v1/projects/:id/cicd/build-runs/:runId", Action: "DELETE", Description: "Delete CI build run"},
		{Name: "CI/CD 审批流查询", Resource: "/api/v1/projects/:id/cicd/approval-flow", Action: "GET", Description: "Get CD approval flow"},
		{Name: "CI/CD 审批流保存", Resource: "/api/v1/projects/:id/cicd/approval-flow", Action: "PUT", Description: "Upsert CD approval flow"},
		{Name: "CI/CD CD 工单列表", Resource: "/api/v1/projects/:id/cicd/release-runs", Action: "GET", Description: "List CD release runs"},
		{Name: "CI/CD CD 工单详情", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId", Action: "GET", Description: "Get CD release run detail"},
		{Name: "CI/CD 工单审批步骤", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/approval-steps", Action: "GET", Description: "List release approval steps"},
		{Name: "CI/CD 审批通过", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/approve", Action: "POST", Description: "Approve release run"},
		{Name: "CI/CD 审批驳回", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/reject", Action: "POST", Description: "Reject release run"},
		{Name: "CI/CD 执行发布", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/execute", Action: "POST", Description: "Execute release run"},
		{Name: "CI/CD 终止发布", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/terminate", Action: "POST", Description: "Terminate release run"},
		{Name: "CI/CD 批量审批通过", Resource: "/api/v1/projects/:id/cicd/release-runs/batch-approve", Action: "POST", Description: "Batch approve release runs"},
		{Name: "CI/CD 批量审批驳回", Resource: "/api/v1/projects/:id/cicd/release-runs/batch-reject", Action: "POST", Description: "Batch reject release runs"},
		{Name: "CI/CD 批量执行发布", Resource: "/api/v1/projects/:id/cicd/release-runs/batch-execute", Action: "POST", Description: "Batch execute release runs"},
		{Name: "CI/CD 批量终止发布", Resource: "/api/v1/projects/:id/cicd/release-runs/batch-terminate", Action: "POST", Description: "Batch terminate release runs"},
		{Name: "CI/CD CD 发布日志", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/log", Action: "GET", Description: "Get CD release log"},
		{Name: "CI/CD 删除 CD 工单", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId", Action: "DELETE", Description: "Delete CD release run"},
		{Name: "CI/CD 构建阶段", Resource: "/api/v1/projects/:id/cicd/build-runs/:runId/stages", Action: "GET", Description: "List CI build stages"},
		{Name: "CI/CD 构建制品元数据", Resource: "/api/v1/projects/:id/cicd/build-runs/:runId/artifacts-meta", Action: "GET", Description: "List CI build artifacts meta"},
		{Name: "CI/CD 平台回滚", Resource: "/api/v1/projects/:id/cicd/release-runs/:runId/platform-rollback", Action: "POST", Description: "Platform rollback release"},

		{Name: "镜像仓库列表", Resource: "/api/v1/registries", Action: "GET", Description: "List container registries"},
		{Name: "镜像仓库创建", Resource: "/api/v1/registries", Action: "POST", Description: "Create container registry"},
		{Name: "镜像仓库详情", Resource: "/api/v1/registries/:registryId", Action: "GET", Description: "Get container registry"},
		{Name: "镜像仓库更新", Resource: "/api/v1/registries/:registryId", Action: "PUT", Description: "Update container registry"},
		{Name: "镜像仓库删除", Resource: "/api/v1/registries/:registryId", Action: "DELETE", Description: "Delete container registry"},
		{Name: "镜像仓库探测", Resource: "/api/v1/registries/:registryId/ping", Action: "POST", Description: "Ping container registry"},
		{Name: "Harbor 项目浏览", Resource: "/api/v1/registries/browse/projects", Action: "GET", Description: "Browse Harbor projects"},
		{Name: "Harbor 仓库浏览", Resource: "/api/v1/registries/browse/repositories", Action: "GET", Description: "Browse Harbor repositories"},
		{Name: "Harbor 制品浏览", Resource: "/api/v1/registries/browse/artifacts", Action: "GET", Description: "Browse Harbor artifacts"},
		{Name: "Harbor 制品删除", Resource: "/api/v1/registries/browse/artifacts/delete", Action: "POST", Description: "Delete Harbor artifact"},
		{Name: "镜像清理策略列表", Resource: "/api/v1/registries/cleanup-policies", Action: "GET", Description: "List registry cleanup policies"},
		{Name: "镜像清理策略创建", Resource: "/api/v1/registries/cleanup-policies", Action: "POST", Description: "Create registry cleanup policy"},
		{Name: "镜像清理策略更新", Resource: "/api/v1/registries/cleanup-policies/:policyId", Action: "PUT", Description: "Update registry cleanup policy"},
		{Name: "镜像清理策略删除", Resource: "/api/v1/registries/cleanup-policies/:policyId", Action: "DELETE", Description: "Delete registry cleanup policy"},
		{Name: "镜像清理策略执行", Resource: "/api/v1/registries/cleanup-policies/:policyId/run", Action: "POST", Description: "Run registry cleanup policy"},
		{Name: "流水线模板列表", Resource: "/api/v1/pipeline-templates", Action: "GET", Description: "List pipeline templates"},
		{Name: "流水线模板创建", Resource: "/api/v1/pipeline-templates", Action: "POST", Description: "Create pipeline template"},
		{Name: "流水线模板更新", Resource: "/api/v1/pipeline-templates/:templateId", Action: "PUT", Description: "Update pipeline template"},
		{Name: "项目仓库绑定查询", Resource: "/api/v1/projects/:id/registry-binding", Action: "GET", Description: "Get project registry binding"},
		{Name: "项目仓库绑定保存", Resource: "/api/v1/projects/:id/registry-binding", Action: "PUT", Description: "Upsert project registry binding"},
		{Name: "项目仓库绑定删除", Resource: "/api/v1/projects/:id/registry-binding", Action: "DELETE", Description: "Delete project registry binding"},

		{Name: "AI 状态查询", Resource: "/api/v1/ai/status", Action: "GET", Description: "Get AI status"},
		{Name: "AI 连通测试", Resource: "/api/v1/ai/ping", Action: "POST", Description: "Ping AI provider"},
		{Name: "AI 运维对话", Resource: "/api/v1/ai/chat", Action: "POST", Description: "AI ops assistant chat"},
		{Name: "AI 会话列表", Resource: "/api/v1/ai/sessions", Action: "GET", Description: "List AI chat sessions"},
		{Name: "AI 创建会话", Resource: "/api/v1/ai/sessions", Action: "POST", Description: "Create AI chat session"},
		{Name: "AI 会话详情", Resource: "/api/v1/ai/sessions/:id", Action: "GET", Description: "Get AI chat session"},
		{Name: "AI 更新会话", Resource: "/api/v1/ai/sessions/:id", Action: "PATCH", Description: "Update AI chat session"},
		{Name: "AI 删除会话", Resource: "/api/v1/ai/sessions/:id", Action: "DELETE", Description: "Delete AI chat session"},
		{Name: "AI 清空会话消息", Resource: "/api/v1/ai/sessions/:id/clear", Action: "POST", Description: "Clear AI chat session messages"},
		{Name: "AI Pod 排障分析", Resource: "/api/v1/ai/k8s/pod-diagnose", Action: "POST", Description: "AI analyze pod diagnose"},
		{Name: "AI CI 构建失败分析", Resource: "/api/v1/ai/cicd/build-fail", Action: "POST", Description: "AI analyze CI build failure"},
		{Name: "AI 告警解释", Resource: "/api/v1/ai/alert/explain", Action: "POST", Description: "AI explain alert fingerprint delivery"},
		{Name: "AI 审批列表", Resource: "/api/v1/ai/approvals", Action: "GET", Description: "List AI tool approvals"},
		{Name: "AI 审批审核", Resource: "/api/v1/ai/approvals/:id/review", Action: "POST", Description: "Review AI tool approval"},
		{Name: "AI 审批执行", Resource: "/api/v1/ai/approvals/:id/execute", Action: "POST", Description: "Execute approved AI tool"},
		{Name: "AI 知识库同步", Resource: "/api/v1/ai/knowledge/sync", Action: "POST", Description: "Sync AI knowledge base to ES"},
		{Name: "AI 能力中心概览", Resource: "/api/v1/ai/center/overview", Action: "GET", Description: "AI capability center overview"},
		{Name: "AI 能力中心重载种子", Resource: "/api/v1/ai/center/reseed", Action: "POST", Description: "Reseed AI center from data/ai"},
		{Name: "AI Prompt 列表", Resource: "/api/v1/ai/center/prompts", Action: "GET", Description: "List AI prompts"},
		{Name: "AI Prompt 版本", Resource: "/api/v1/ai/center/prompts/:id/versions", Action: "GET", Description: "List prompt versions"},
		{Name: "AI Prompt 发布", Resource: "/api/v1/ai/center/prompts/:id/publish", Action: "POST", Description: "Publish prompt version"},
		{Name: "AI Prompt 回滚", Resource: "/api/v1/ai/center/prompts/:id/versions/:vid/rollback", Action: "POST", Description: "Rollback prompt version"},
		{Name: "AI 模型列表", Resource: "/api/v1/ai/center/models", Action: "GET", Description: "List LLM models"},
		{Name: "AI 创建模型", Resource: "/api/v1/ai/center/models", Action: "POST", Description: "Create LLM model"},
		{Name: "AI 更新模型", Resource: "/api/v1/ai/center/models/:id", Action: "PUT", Description: "Update LLM model"},
		{Name: "AI 删除模型", Resource: "/api/v1/ai/center/models/:id", Action: "DELETE", Description: "Delete LLM model"},
		{Name: "AI 设默认模型", Resource: "/api/v1/ai/center/models/:id/default", Action: "POST", Description: "Set default LLM model"},
		{Name: "AI Tool 列表", Resource: "/api/v1/ai/center/tools", Action: "GET", Description: "List AI tools"},
		{Name: "AI Tool 更新", Resource: "/api/v1/ai/center/tools/:id", Action: "PATCH", Description: "Update AI tool"},
		{Name: "AI 故障案例", Resource: "/api/v1/ai/center/cases", Action: "GET", Description: "List incident cases"},
		{Name: "AI SOP 列表", Resource: "/api/v1/ai/center/sops", Action: "GET", Description: "List SOPs"},
		{Name: "AI 知识库列表", Resource: "/api/v1/ai/center/knowledge-bases", Action: "GET", Description: "List knowledge bases"},
		{Name: "AI 评估用例", Resource: "/api/v1/ai/center/eval/cases", Action: "GET", Description: "List eval cases"},
		{Name: "AI 评估运行", Resource: "/api/v1/ai/center/eval/run", Action: "POST", Description: "Run eval suite"},

		{Name: "ES 连接列表", Resource: "/api/v1/esmgmt/connections", Action: "GET", Description: "List Elasticsearch connections"},
		{Name: "ES 创建连接", Resource: "/api/v1/esmgmt/connections", Action: "POST", Description: "Create Elasticsearch connection"},
		{Name: "ES 更新连接", Resource: "/api/v1/esmgmt/connections/:id", Action: "PUT", Description: "Update Elasticsearch connection"},
		{Name: "ES 删除连接", Resource: "/api/v1/esmgmt/connections/:id", Action: "DELETE", Description: "Delete Elasticsearch connection"},
		{Name: "ES 连接探测", Resource: "/api/v1/esmgmt/connections/:id/ping", Action: "POST", Description: "Ping Elasticsearch connection"},
		{Name: "ES 连通测试", Resource: "/api/v1/esmgmt/connections/test", Action: "POST", Description: "Test Elasticsearch credentials without saving"},
		{Name: "ES 集群健康", Resource: "/api/v1/esmgmt/cluster/health", Action: "GET", Description: "Get Elasticsearch cluster health"},
		{Name: "ES 索引列表", Resource: "/api/v1/esmgmt/indices", Action: "GET", Description: "List Elasticsearch indices"},
		{Name: "ES 创建索引", Resource: "/api/v1/esmgmt/indices", Action: "POST", Description: "Create Elasticsearch index"},
		{Name: "ES 删除索引", Resource: "/api/v1/esmgmt/indices/:name", Action: "DELETE", Description: "Delete Elasticsearch index"},
		{Name: "ES 打开索引", Resource: "/api/v1/esmgmt/indices/:name/open", Action: "POST", Description: "Open Elasticsearch index"},
		{Name: "ES 关闭索引", Resource: "/api/v1/esmgmt/indices/:name/close", Action: "POST", Description: "Close Elasticsearch index"},
		{Name: "ES 节点列表", Resource: "/api/v1/esmgmt/nodes", Action: "GET", Description: "List Elasticsearch nodes"},
		{Name: "ES REST 代理", Resource: "/api/v1/esmgmt/proxy", Action: "POST", Description: "Proxy safe Elasticsearch REST calls"},
		{Name: "ES 索引备份", Resource: "/api/v1/esmgmt/backups", Action: "POST", Description: "Backup Elasticsearch index to MinIO"},
		{Name: "ES 备份列表", Resource: "/api/v1/esmgmt/backups", Action: "GET", Description: "List Elasticsearch backup jobs"},
		{Name: "ES 备份详情", Resource: "/api/v1/esmgmt/backups/:id", Action: "GET", Description: "Get Elasticsearch backup job"},
		{Name: "ES 备份下载", Resource: "/api/v1/esmgmt/backups/:id/download", Action: "GET", Description: "Presign Elasticsearch backup download"},
		{Name: "ES 索引恢复", Resource: "/api/v1/esmgmt/restores", Action: "POST", Description: "Restore Elasticsearch index from MinIO backup"},
		{Name: "ES 恢复列表", Resource: "/api/v1/esmgmt/restores", Action: "GET", Description: "List Elasticsearch restore jobs"},
		{Name: "ES 恢复详情", Resource: "/api/v1/esmgmt/restores/:id", Action: "GET", Description: "Get Elasticsearch restore job"},
		{Name: "ES 备份调度列表", Resource: "/api/v1/esmgmt/schedules", Action: "GET", Description: "List Elasticsearch backup schedules"},
		{Name: "ES 创建备份调度", Resource: "/api/v1/esmgmt/schedules", Action: "POST", Description: "Create Elasticsearch backup schedule"},
		{Name: "ES 更新备份调度", Resource: "/api/v1/esmgmt/schedules/:id", Action: "PUT", Description: "Update Elasticsearch backup schedule"},
		{Name: "ES 删除备份调度", Resource: "/api/v1/esmgmt/schedules/:id", Action: "DELETE", Description: "Delete Elasticsearch backup schedule"},

		{Name: "Event 列表", Resource: "/api/v1/events", Action: "GET", Description: "List events"},
		{Name: "Event 分组列表", Resource: "/api/v1/events/grouped", Action: "GET", Description: "List events grouped"},
		{Name: "CRD 列表", Resource: "/api/v1/crds", Action: "GET", Description: "List custom resource definitions"},
		{Name: "CRD 详情", Resource: "/api/v1/crds/detail", Action: "GET", Description: "Get custom resource definition detail"},
		{Name: "CRD 应用 YAML", Resource: "/api/v1/crds/apply", Action: "POST", Description: "Apply custom resource definition yaml"},
		{Name: "删除 CRD", Resource: "/api/v1/crds", Action: "DELETE", Description: "Delete custom resource definition"},
		{Name: "CR 资源类型列表", Resource: "/api/v1/crs/resources", Action: "GET", Description: "List custom resource types"},
		{Name: "CR 实例列表", Resource: "/api/v1/crs", Action: "GET", Description: "List custom resources"},
		{Name: "CR 实例详情", Resource: "/api/v1/crs/detail", Action: "GET", Description: "Get custom resource detail"},
		{Name: "CR 实例应用 YAML", Resource: "/api/v1/crs/apply", Action: "POST", Description: "Apply custom resource yaml"},
		{Name: "删除 CR 实例", Resource: "/api/v1/crs", Action: "DELETE", Description: "Delete custom resource"},
	}
}

func seedMenus(ctx context.Context, db *gorm.DB, cfg *config.PluginsConfig) error {
	if err := menu.Sync(ctx, db); err != nil {
		return err
	}
	return plugingate.SyncMenuVisibility(ctx, db, cfg)
}
