package router

import (
	"context"
	"log/slog"
	"yunshu/internal/service"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
	"yunshu/internal/pkg/response"
	"yunshu/internal/plugingate"
	"yunshu/internal/repository"

	"github.com/gin-gonic/gin"
)

func registerWorkflowRoutes(api *gin.RouterGroup, d WorkflowRouteDeps) {
	if d == nil || d.WorkflowHandler() == nil {
		return
	}
	wf := api.Group("/workflow")
	wf.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	wf.GET("/definitions/:domain/projects/:project_id", d.WorkflowHandler().GetDefinition)
	wf.PUT("/definitions/:domain/projects/:project_id", d.WorkflowHandler().UpsertDefinition)
	wf.GET("/tickets/pending", d.WorkflowHandler().ListPending)
	wf.GET("/tickets", d.WorkflowHandler().ListTickets)
	wf.POST("/tickets", d.WorkflowHandler().CreateTicket)
	wf.GET("/tickets/:id", d.WorkflowHandler().TicketDetail)
	wf.POST("/tickets/:id/steps/:step_id/review", d.WorkflowHandler().ReviewStep)
}

func registerPermissionSyncRoute(permissions *gin.RouterGroup, d interface{ App() *bootstrap.App }) {
	if d == nil || d.App() == nil {
		return
	}
	permissions.GET("/catalog", func(c *gin.Context) {
		syncSvc := service.NewPermissionSyncService(repository.NewPermissionRepository(d.App().DB), &d.App().Config.Plugins)
		result, err := syncSvc.Catalog(c.Request.Context(), d.App().Engine)
		if err != nil {
			response.Error(c, err)
			return
		}
		response.Success(c, result)
	})
	permissions.POST("/sync-routes", func(c *gin.Context) {
		syncSvc := service.NewPermissionSyncService(repository.NewPermissionRepository(d.App().DB), &d.App().Config.Plugins)
		result, err := syncSvc.SyncFromEngine(c.Request.Context(), d.App().Engine)
		if err != nil {
			response.Error(c, err)
			return
		}
		response.Success(c, result)
	})
}

func syncAPIPermissionsOnBoot(d *RouteDeps, plugins *config.PluginsConfig) {
	if d == nil || d.App() == nil || d.App().Engine == nil {
		return
	}
	syncSvc := service.NewPermissionSyncService(repository.NewPermissionRepository(d.App().DB), plugins)
	result, err := syncSvc.SyncFromEngine(context.Background(), d.App().Engine)
	log := slog.Default().With("component", "permission.sync")
	if err != nil {
		log.Error("API permission sync failed", "error", err)
		return
	}
	log.Info("API permissions synced from routes", "created", result.Created, "skipped", result.Skipped, "total", result.Total)
	reclaimOrphanPermissionsOnBoot(syncSvc, d)
	reportRoutePolicyDiffOnBoot(d, plugins)
}

// reclaimOrphanPermissionsOnBoot 路由下线后自动回收对应权限点与孤儿策略。
// 只回收可推导来源（auto-synced / 目录已登记）且无路由的行；禁用插件与超阈值时只报告不删除。
func reclaimOrphanPermissionsOnBoot(syncSvc *service.PermissionSyncService, d *RouteDeps) {
	app := d.App()
	if app == nil || app.Engine == nil || app.DB == nil {
		return
	}
	log := slog.Default().With("component", "permission.sync")
	result, err := syncSvc.ReclaimOrphans(context.Background(), app.Engine, app.Enforcer)
	if err != nil {
		log.Error("orphan permission reclaim failed", "error", err)
		return
	}
	switch {
	case result.Aborted:
		log.Warn("orphan permission reclaim skipped", "reason", result.Reason, "skipped", result.Skipped)
	case result.DeletedPermissions == 0 && result.RemovedPolicies == 0:
		log.Info("orphan permission reclaim: nothing to remove", "skipped", result.Skipped)
	default:
		log.Info("orphan permissions reclaimed",
			"deleted_permissions", result.DeletedPermissions,
			"removed_policies", result.RemovedPolicies,
			"skipped", result.Skipped,
			"samples", result.Samples,
		)
	}
}

// reportRoutePolicyDiffOnBoot 启动期只读比对路由/权限点/Casbin 策略并打印差异，不做任何删除或改写。
// 差异常见于：新接口漏同步、接口下线后权限点残留、权限点被删但策略未清理。
func reportRoutePolicyDiffOnBoot(d *RouteDeps, plugins *config.PluginsConfig) {
	app := d.App()
	if app == nil || app.Engine == nil || app.DB == nil {
		return
	}
	log := slog.Default().With("component", "permission.sync")
	ctx := context.Background()

	entries := make([]apiroute.Entry, 0)
	for _, e := range apiroute.Collect(app.Engine) {
		if plugins != nil && !plugingate.IsAPIResourceAllowed(e.Path, plugins) {
			continue
		}
		entries = append(entries, e)
	}
	permissions, err := repository.NewPermissionRepository(app.DB).ListAll(ctx)
	if err != nil {
		log.Warn("route/permission/policy diff skipped: list permissions failed", "error", err)
		return
	}
	if plugins != nil {
		visible := make([]model.Permission, 0, len(permissions))
		for _, p := range permissions {
			if plugingate.IsAPIResourceAllowed(p.Resource, plugins) {
				visible = append(visible, p)
			}
		}
		permissions = visible
	}
	var policies [][]string
	if app.Enforcer != nil {
		policies = app.Enforcer.GetPolicy()
	}

	diff := service.BuildRoutePolicyDiff(entries, permissions, policies)
	if diff.IsEmpty() {
		log.Info("route/permission/policy diff: clean")
		return
	}
	log.Warn("route/permission/policy diff detected (report only, no auto-clean)",
		"route_without_permission", len(diff.RouteWithoutPermission),
		"permission_without_route", len(diff.PermissionWithoutRoute),
		"policy_without_permission", len(diff.PolicyWithoutPermission),
		"samples", diff.Samples(10),
	)
}
