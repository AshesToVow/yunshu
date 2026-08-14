package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterProjectRoutes 多租户项目、成员、服务配置与 ES 日志检索。
func RegisterProjectRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectRoutes.GET("", d.projectHandler.List)
	projectRoutes.POST("", d.projectHandler.Create)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))
	projectScoped.PUT("", d.projectHandler.Update)
	projectScoped.DELETE("", d.projectHandler.Delete)
	projectScoped.GET("/members", d.projectHandler.ListProjectMembers)
	projectScoped.POST("/members", d.projectHandler.AddProjectMember)
	projectScoped.PUT("/members/:memberId", d.projectHandler.UpdateProjectMember)
	projectScoped.DELETE("/members/:memberId", d.projectHandler.RemoveProjectMember)
	projectScoped.GET("/services", d.projectHandler.ListServices)
	projectScoped.POST("/services", d.projectHandler.UpsertService)
	projectScoped.DELETE("/services/:serviceId", d.projectHandler.DeleteService)
	projectScoped.GET("/log-sources", d.projectHandler.ListLogSources)
	projectScoped.POST("/log-sources", d.projectHandler.UpsertLogSource)
	projectScoped.DELETE("/log-sources/:logSourceId", d.projectHandler.DeleteLogSource)
	projectScoped.GET("/logs/search", d.projectHandler.SearchLogs)
	projectScoped.GET("/logs/export", d.projectHandler.ExportLogs)
	projectScoped.GET("/log-retention", d.logPlatformHandler.GetProjectRetention)
	projectScoped.PUT("/log-retention", d.logPlatformHandler.UpsertProjectRetention)
	projectScoped.DELETE("/log-retention", d.logPlatformHandler.DeleteProjectRetention)
	projectScoped.POST("/loggie/bootstrap", d.loggieHandler.Bootstrap)
	projectScoped.GET("/loggie/status", d.loggieHandler.ListStatus)
	projectScoped.GET("/loggie/bootstrap-sources", d.loggieHandler.PreviewBootstrapSources)
	projectScoped.GET("/loggie/pipeline/download", d.loggieHandler.DownloadPipeline)
	projectScoped.POST("/loggie/deploy", d.loggieHandler.DeployConfig)
	projectScoped.POST("/loggie/install", d.loggieHandler.InstallLoggie)
	projectScoped.POST("/loggie/uninstall", d.loggieHandler.UninstallLoggie)
	projectScoped.POST("/loggie/start", d.loggieHandler.StartLoggie)
	projectScoped.POST("/loggie/stop", d.loggieHandler.StopLoggie)
	projectScoped.POST("/loggie/restart", d.loggieHandler.RestartLoggie)
	projectScoped.POST("/loggie/sync", d.loggieHandler.SyncFromLogSources)
	if d.clusterLogHandler != nil {
		projectScoped.GET("/cluster-log/rules", d.clusterLogHandler.ListRules)
		projectScoped.POST("/cluster-log/rules", d.clusterLogHandler.CreateRule)
		projectScoped.PUT("/cluster-log/rules/:rule_id", d.clusterLogHandler.UpdateRule)
		projectScoped.DELETE("/cluster-log/rules/:rule_id", d.clusterLogHandler.DeleteRule)
		projectScoped.GET("/cluster-log/agents", d.clusterLogHandler.ListAgents)
		projectScoped.GET("/cluster-log/pipelines/preview", d.clusterLogHandler.PreviewPipelines)
		projectScoped.PUT("/cluster-log/pipelines", d.clusterLogHandler.SavePipelines)
		projectScoped.POST("/cluster-log/deploy", d.clusterLogHandler.Deploy)
		projectScoped.GET("/cluster-log/status", d.clusterLogHandler.RefreshStatus)
	}
	projectScoped.GET("/service-catalog", d.projectCatalogHandler.ListCatalog)
	projectScoped.POST("/service-catalog", d.projectCatalogHandler.UpsertCatalog)
	projectScoped.GET("/service-catalog/:catalogId", d.projectCatalogHandler.GetCatalog)
	projectScoped.GET("/service-catalog/:catalogId/portrait", d.projectCatalogHandler.GetPortrait)
	projectScoped.DELETE("/service-catalog/:catalogId", d.projectCatalogHandler.DeleteCatalog)
	projectScoped.POST("/service-catalog/:catalogId/links", d.projectCatalogHandler.AddLink)
	projectScoped.DELETE("/service-catalog/:catalogId/links/:linkId", d.projectCatalogHandler.DeleteLink)
}
