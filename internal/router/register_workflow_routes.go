package router

import (
	"context"
	"log/slog"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	systemsvc "yunshu/internal/service/system"

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
	permissions.POST("/sync-routes", func(c *gin.Context) {
		syncSvc := systemsvc.NewPermissionSyncService(repository.NewPermissionRepository(d.App().DB), &d.App().Config.Plugins)
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
	syncSvc := systemsvc.NewPermissionSyncService(repository.NewPermissionRepository(d.App().DB), plugins)
	result, err := syncSvc.SyncFromEngine(context.Background(), d.App().Engine)
	log := slog.Default().With("component", "permission.sync")
	if err != nil {
		log.Error("API permission sync failed", "error", err)
		return
	}
	log.Info("API permissions synced from routes", "created", result.Created, "skipped", result.Skipped, "total", result.Total)
}
