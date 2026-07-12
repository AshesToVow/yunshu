package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterDbmgmtRoutes 数据库管理插件（项目作用域）。
func RegisterDbmgmtRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))

	g := projectScoped.Group("/dbmgmt")

	g.GET("/instances", d.dbmgmtHandler.ListInstances)
	g.POST("/instances", d.dbmgmtHandler.CreateInstance)
	g.GET("/instances/:instanceId", d.dbmgmtHandler.GetInstance)
	g.PUT("/instances/:instanceId", d.dbmgmtHandler.UpdateInstance)
	g.DELETE("/instances/:instanceId", d.dbmgmtHandler.DeleteInstance)
	g.POST("/instances/:instanceId/ping", d.dbmgmtHandler.PingInstance)

	g.GET("/instances/:instanceId/metadata/databases", d.dbmgmtHandler.ListDatabases)
	g.GET("/instances/:instanceId/metadata/tables", d.dbmgmtHandler.ListTables)
	g.GET("/instances/:instanceId/metadata/columns", d.dbmgmtHandler.ListColumns)

	g.POST("/instances/:instanceId/query", d.dbmgmtHandler.Query)
	g.POST("/instances/:instanceId/check", d.dbmgmtHandler.CheckSQL)
	g.POST("/instances/:instanceId/execute", d.dbmgmtHandler.Execute)
	g.POST("/instances/:instanceId/import", d.dbmgmtHandler.Import)

	g.GET("/grants", d.dbmgmtHandler.ListGrants)
	g.POST("/grants", d.dbmgmtHandler.CreateGrant)
	g.PUT("/grants/:grantId", d.dbmgmtHandler.UpdateGrant)
	g.DELETE("/grants/:grantId", d.dbmgmtHandler.DeleteGrant)
	g.GET("/grants/effective", d.dbmgmtHandler.GetEffectiveGrant)

	g.GET("/approval-flow", d.dbmgmtHandler.GetApprovalFlow)
	g.PUT("/approval-flow", d.dbmgmtHandler.UpsertApprovalFlow)

	g.GET("/access-requests", d.dbmgmtHandler.ListAccessRequests)
	g.POST("/access-requests", d.dbmgmtHandler.CreateAccessRequest)
	g.POST("/access-requests/:requestId/approve", d.dbmgmtHandler.ApproveAccessRequest)
	g.POST("/access-requests/:requestId/reject", d.dbmgmtHandler.RejectAccessRequest)

	g.GET("/app-user-requests", d.dbmgmtHandler.ListAppUserRequests)
	g.POST("/app-user-requests", d.dbmgmtHandler.CreateAppUserRequest)
	g.POST("/app-user-requests/:requestId/approve", d.dbmgmtHandler.ApproveAppUserRequest)
	g.POST("/app-user-requests/:requestId/reject", d.dbmgmtHandler.RejectAppUserRequest)

	g.GET("/instances/:instanceId/mysql-users", d.dbmgmtHandler.ListInstanceMySQLUsers)
	g.GET("/instances/:instanceId/mysql-user-privileges", d.dbmgmtHandler.GetInstanceMySQLUserPrivileges)
	g.GET("/instances/:instanceId/accounts/:accountId/password", d.dbmgmtHandler.GetInstanceAccountPassword)

	g.GET("/tickets", d.dbmgmtHandler.ListTickets)
	g.GET("/tickets/:ticketId", d.dbmgmtHandler.GetTicket)
	g.GET("/tickets/:ticketId/steps", d.dbmgmtHandler.ListTicketSteps)
	g.GET("/tickets/:ticketId/rollback", d.dbmgmtHandler.GetTicketRollback)
	g.GET("/tickets/:ticketId/rollback/preview", d.dbmgmtHandler.PreviewRollbackTicket)
	g.POST("/tickets/:ticketId/rollback/submit", d.dbmgmtHandler.SubmitRollbackTicket)
	g.GET("/tickets/:ticketId/osc", d.dbmgmtHandler.ListTicketOSCJobs)
	g.GET("/tickets/:ticketId/osc/:sqlsha1", d.dbmgmtHandler.GetTicketOSC)
	g.POST("/tickets/:ticketId/osc/:sqlsha1/control", d.dbmgmtHandler.ControlTicketOSC)
	g.POST("/tickets/:ticketId/approve", d.dbmgmtHandler.ApproveTicket)
	g.POST("/tickets/:ticketId/reject", d.dbmgmtHandler.RejectTicket)
	g.POST("/tickets/:ticketId/execute", d.dbmgmtHandler.ExecuteTicket)

	g.GET("/executions", d.dbmgmtHandler.ListExecutions)
	g.GET("/audit-logs", d.dbmgmtHandler.ListAuditLogs)
}
