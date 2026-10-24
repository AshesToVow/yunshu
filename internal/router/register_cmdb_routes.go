package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCMDBRoutes CMDB 服务器资产：主机、分组、云账号、SSH/Web 终端（仍挂在项目作用域 /projects/:id 下）。
func RegisterCMDBRoutes(api *gin.RouterGroup, d CMDBRouteDeps) {
	h := d.CMDBHandler()
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))

	projectScoped.GET("/servers", h.ListServers)
	projectScoped.POST("/servers", h.UpsertServer)
	projectScoped.GET("/servers/:serverId", h.ServerDetail)
	projectScoped.GET("/servers/:serverId/my-access", h.MyServerAccess)
	projectScoped.DELETE("/servers/:serverId", h.DeleteServer)
	projectScoped.POST("/servers/:serverId/exec", h.ExecServerCommand)
	projectScoped.POST("/servers/:serverId/probe", h.ProbeServer)
	projectScoped.GET("/servers/:serverId/files", h.ListServerFiles)
	projectScoped.POST("/servers/:serverId/files/upload", h.UploadServerFile)
	projectScoped.GET("/servers/:serverId/files/download", h.DownloadServerFile)
	projectScoped.POST("/servers/:serverId/files/delete", h.DeleteServerFile)
	projectScoped.GET("/server-groups/tree", h.ListServerGroups)
	projectScoped.POST("/server-groups", h.UpsertServerGroup)
	projectScoped.PUT("/server-groups/:groupId", h.UpdateServerGroup)
	projectScoped.DELETE("/server-groups/:groupId", h.DeleteServerGroup)
	projectScoped.GET("/cloud-accounts", h.ListCloudAccounts)
	projectScoped.POST("/cloud-accounts", h.UpsertCloudAccount)
	projectScoped.PUT("/cloud-accounts/:accountId", h.UpdateCloudAccount)
	projectScoped.DELETE("/cloud-accounts/:accountId", h.DeleteCloudAccount)
	projectScoped.PUT("/cloud-accounts/:accountId/sync", h.SyncCloudAccount)
	projectScoped.POST("/servers/import", h.ImportServers)
	projectScoped.GET("/servers/import-template", h.ServersImportTemplate)
	projectScoped.GET("/servers/export", h.ExportServers)
	projectScoped.POST("/servers/test", h.TestServer)
	projectScoped.POST("/servers/test/batch", h.BatchTestServers)
	projectScoped.POST("/servers/:serverId/cloud-actions", h.CloudServerAction)
	projectScoped.POST("/servers/sync", h.SyncServers)
	projectScoped.GET("/server-access-grants", h.ListServerGrants)
	projectScoped.POST("/server-access-grants", h.UpsertServerGrant)
	projectScoped.POST("/server-access-grants/bulk", h.BulkUpsertServerGrants)
	projectScoped.POST("/server-access-grants/bootstrap", h.BootstrapServerGrants)
	projectScoped.DELETE("/server-access-grants/:grantId", h.DeleteServerGrant)

	projectsWS := api.Group("/projects")
	projectsWS.Use(d.WSAuthMiddleware(), d.Authorize(), d.OpAudit(), middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))
	projectsWS.GET("/:id/servers/:serverId/terminal/ws", h.ServerTerminalWS)
}
