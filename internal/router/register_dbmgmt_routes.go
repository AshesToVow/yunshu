package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterDbmgmtRoutes 数据库管理插件（项目作用域）。
func RegisterDbmgmtRoutes(api *gin.RouterGroup, d DbmgmtRouteDeps) {
	h := d.DbmgmtHandler()
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))

	g := projectScoped.Group("/dbmgmt")

	g.GET("/instances", h.ListInstances)
	g.POST("/instances", h.CreateInstance)
	g.GET("/instances/:instanceId", h.GetInstance)
	g.PUT("/instances/:instanceId", h.UpdateInstance)
	g.DELETE("/instances/:instanceId", h.DeleteInstance)
	g.POST("/instances/:instanceId/ping", h.PingInstance)

	g.GET("/instances/:instanceId/metadata/databases", h.ListDatabases)
	g.GET("/instances/:instanceId/metadata/tables", h.ListTables)
	g.GET("/instances/:instanceId/metadata/columns", h.ListColumns)

	g.POST("/instances/:instanceId/query", h.Query)
	g.POST("/instances/:instanceId/check", h.CheckSQL)
	g.POST("/instances/:instanceId/execute", h.Execute)
	g.POST("/instances/:instanceId/import", h.Import)

	g.GET("/instances/:instanceId/column-mask-rules", h.ListColumnMaskRules)
	g.POST("/instances/:instanceId/column-mask-rules", h.UpsertColumnMaskRule)
	g.DELETE("/instances/:instanceId/column-mask-rules/:ruleId", h.DeleteColumnMaskRule)

	g.GET("/grants", h.ListGrants)
	g.POST("/grants", h.CreateGrant)
	g.PUT("/grants/:grantId", h.UpdateGrant)
	g.DELETE("/grants/:grantId", h.DeleteGrant)
	g.GET("/grants/effective", h.GetEffectiveGrant)

	g.GET("/approval-flow", h.GetApprovalFlow)
	g.PUT("/approval-flow", h.UpsertApprovalFlow)

	g.GET("/access-requests", h.ListAccessRequests)
	g.POST("/access-requests", h.CreateAccessRequest)
	g.POST("/access-requests/:requestId/approve", h.ApproveAccessRequest)
	g.POST("/access-requests/:requestId/reject", h.RejectAccessRequest)

	g.GET("/app-user-requests", h.ListAppUserRequests)
	g.POST("/app-user-requests", h.CreateAppUserRequest)
	g.POST("/app-user-requests/:requestId/approve", h.ApproveAppUserRequest)
	g.POST("/app-user-requests/:requestId/reject", h.RejectAppUserRequest)

	g.GET("/instances/:instanceId/mysql-users", h.ListInstanceMySQLUsers)
	g.GET("/instances/:instanceId/mysql-user-privileges", h.GetInstanceMySQLUserPrivileges)
	g.GET("/instances/:instanceId/accounts/:accountId/password", h.GetInstanceAccountPassword)

	g.GET("/tickets", h.ListTickets)
	g.GET("/tickets/:ticketId", h.GetTicket)
	g.GET("/tickets/:ticketId/steps", h.ListTicketSteps)
	g.GET("/tickets/:ticketId/rollback", h.GetTicketRollback)
	g.GET("/tickets/:ticketId/rollback/preview", h.PreviewRollbackTicket)
	g.POST("/tickets/:ticketId/rollback/submit", h.SubmitRollbackTicket)
	g.GET("/tickets/:ticketId/osc", h.ListTicketOSCJobs)
	g.GET("/tickets/:ticketId/osc/:sqlsha1", h.GetTicketOSC)
	g.POST("/tickets/:ticketId/osc/:sqlsha1/control", h.ControlTicketOSC)
	g.POST("/tickets/:ticketId/approve", h.ApproveTicket)
	g.POST("/tickets/:ticketId/reject", h.RejectTicket)
	g.POST("/tickets/:ticketId/execute", h.ExecuteTicket)

	g.GET("/executions", h.ListExecutions)
	g.GET("/audit-logs", h.ListAuditLogs)
}
