package router

import (
	"context"
	"log/slog"

	"yunshu/internal/config"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"
	systemsvc "yunshu/internal/service/system"

	"github.com/gin-gonic/gin"
)

func registerWorkflowRoutes(api *gin.RouterGroup, d *RouteDeps) {
	if d == nil || d.workflowHandler == nil {
		return
	}
	wf := api.Group("/workflow")
	wf.Use(d.authMiddleware, d.authorize, d.opAudit)
	wf.GET("/definitions/:domain/projects/:project_id", d.workflowHandler.GetDefinition)
	wf.PUT("/definitions/:domain/projects/:project_id", d.workflowHandler.UpsertDefinition)
	wf.GET("/tickets/pending", d.workflowHandler.ListPending)
	wf.GET("/tickets", d.workflowHandler.ListTickets)
	wf.POST("/tickets", d.workflowHandler.CreateTicket)
	wf.GET("/tickets/:id", d.workflowHandler.TicketDetail)
	wf.POST("/tickets/:id/steps/:step_id/review", d.workflowHandler.ReviewStep)
}

func registerPermissionSyncRoute(permissions *gin.RouterGroup, d *RouteDeps) {
	if d == nil || d.app == nil {
		return
	}
	permissions.POST("/sync-routes", func(c *gin.Context) {
		syncSvc := systemsvc.NewPermissionSyncService(repository.NewPermissionRepository(d.app.DB), &d.app.Config.Plugins)
		result, err := syncSvc.SyncFromEngine(c.Request.Context(), d.app.Engine)
		if err != nil {
			response.Error(c, err)
			return
		}
		response.Success(c, result)
	})
}

func syncAPIPermissionsOnBoot(d *RouteDeps, plugins *config.PluginsConfig) {
	if d == nil || d.app == nil || d.app.Engine == nil {
		return
	}
	syncSvc := systemsvc.NewPermissionSyncService(repository.NewPermissionRepository(d.app.DB), plugins)
	result, err := syncSvc.SyncFromEngine(context.Background(), d.app.Engine)
	log := slog.Default().With("component", "permission.sync")
	if err != nil {
		log.Error("API permission sync failed", "error", err)
		return
	}
	log.Info("API permissions synced from routes", "created", result.Created, "skipped", result.Skipped, "total", result.Total)
}
