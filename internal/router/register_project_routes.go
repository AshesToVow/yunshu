package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterProjectRoutes 多租户项目、成员、服务配置、日志 Agent 与日志流。
func RegisterProjectRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectRoutes.GET("", d.projectHandler.List)
	projectRoutes.POST("", d.projectHandler.Create)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))
	projectScoped.PUT("", d.projectHandler.Update)
	projectScoped.DELETE("", d.projectHandler.Delete)
	projectScoped.GET("/application-topology", d.projectHandler.ApplicationTopology)
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
	projectScoped.GET("/agents/list", d.logAgentHandler.List)
	projectScoped.DELETE("/agents/:agentId", d.logAgentHandler.Delete)
	projectScoped.POST("/agents/heartbeat-refresh", d.logAgentHandler.BatchRefreshHeartbeat)
	projectScoped.GET("/agents/status", d.logAgentHandler.Status)
	projectScoped.POST("/agents/bootstrap", d.logAgentHandler.Bootstrap)
	projectScoped.POST("/agents/rotate-token", d.logAgentHandler.RotateToken)
	projectScoped.GET("/agents/discovery", d.agentDiscoveryHandler.List)
	projectScoped.GET("/logs/stream", d.projectHandler.StreamLogs)
	projectScoped.GET("/logs/export", d.projectHandler.ExportLogs)

	agents := api.Group("/agents")
	agents.Use(d.authMiddleware, d.authorize, d.opAudit)
	agents.POST("/register", d.logAgentHandler.Register)
	api.GET("/agents/runtime-config", d.logAgentHandler.RuntimeConfig)
	api.POST("/agents/public-register", d.logAgentHandler.PublicRegister)
	api.POST("/agents/health/report", d.logAgentHandler.ReportHealth)
	api.POST("/agents/discovery/report", d.agentDiscoveryHandler.Report)
}
