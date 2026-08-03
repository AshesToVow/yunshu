package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCMDBRoutes CMDB 服务器资产：主机、分组、云账号、SSH/Web 终端（仍挂在项目作用域 /projects/:id 下）。
func RegisterCMDBRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))

	projectScoped.GET("/servers", d.cmdbHandler.ListServers)
	projectScoped.POST("/servers", d.cmdbHandler.UpsertServer)
	projectScoped.GET("/servers/:serverId", d.cmdbHandler.ServerDetail)
	projectScoped.GET("/servers/:serverId/my-access", d.cmdbHandler.MyServerAccess)
	projectScoped.DELETE("/servers/:serverId", d.cmdbHandler.DeleteServer)
	projectScoped.POST("/servers/:serverId/exec", d.cmdbHandler.ExecServerCommand)
	projectScoped.GET("/server-groups/tree", d.cmdbHandler.ListServerGroups)
	projectScoped.POST("/server-groups", d.cmdbHandler.UpsertServerGroup)
	projectScoped.PUT("/server-groups/:groupId", d.cmdbHandler.UpdateServerGroup)
	projectScoped.DELETE("/server-groups/:groupId", d.cmdbHandler.DeleteServerGroup)
	projectScoped.GET("/cloud-accounts", d.cmdbHandler.ListCloudAccounts)
	projectScoped.POST("/cloud-accounts", d.cmdbHandler.UpsertCloudAccount)
	projectScoped.PUT("/cloud-accounts/:accountId", d.cmdbHandler.UpdateCloudAccount)
	projectScoped.DELETE("/cloud-accounts/:accountId", d.cmdbHandler.DeleteCloudAccount)
	projectScoped.PUT("/cloud-accounts/:accountId/sync", d.cmdbHandler.SyncCloudAccount)
	projectScoped.POST("/servers/import", d.cmdbHandler.ImportServers)
	projectScoped.GET("/servers/import-template", d.cmdbHandler.ServersImportTemplate)
	projectScoped.GET("/servers/export", d.cmdbHandler.ExportServers)
	projectScoped.POST("/servers/test", d.cmdbHandler.TestServer)
	projectScoped.POST("/servers/test/batch", d.cmdbHandler.BatchTestServers)
	projectScoped.POST("/servers/:serverId/cloud-actions", d.cmdbHandler.CloudServerAction)
	projectScoped.POST("/servers/sync", d.cmdbHandler.SyncServers)
	projectScoped.GET("/server-access-grants", d.cmdbHandler.ListServerGrants)
	projectScoped.POST("/server-access-grants", d.cmdbHandler.UpsertServerGrant)
	projectScoped.POST("/server-access-grants/bulk", d.cmdbHandler.BulkUpsertServerGrants)
	projectScoped.POST("/server-access-grants/bootstrap", d.cmdbHandler.BootstrapServerGrants)
	projectScoped.DELETE("/server-access-grants/:grantId", d.cmdbHandler.DeleteServerGrant)

	projectsWS := api.Group("/projects")
	projectsWS.Use(d.wsAuthMiddleware, d.authorize, d.opAudit, middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.app.Logger))
	projectsWS.GET("/:id/servers/:serverId/terminal/ws", d.cmdbHandler.ServerTerminalWS)
}
