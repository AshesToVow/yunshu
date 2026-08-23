package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterBackupRoutes MySQL 备份（挂在项目作用域下）。
func RegisterBackupRoutes(api *gin.RouterGroup, d *RouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.authMiddleware, d.authorize, d.opAudit)
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.projectMemberRepo, d.projectRepo, d.app.Logger))

	mysqlBackup := projectScoped.Group("/mysql-backup")
	mysqlBackup.GET("/mysqldump-options", d.mysqlBackupHandler.ListMysqldumpOptions)
	mysqlBackup.GET("/instances", d.mysqlBackupHandler.ListInstances)
	mysqlBackup.POST("/instances", d.mysqlBackupHandler.CreateInstance)
	mysqlBackup.PUT("/instances/:instanceId", d.mysqlBackupHandler.UpdateInstance)
	mysqlBackup.DELETE("/instances/:instanceId", d.mysqlBackupHandler.DeleteInstance)
	mysqlBackup.POST("/instances/:instanceId/ping", d.mysqlBackupHandler.PingInstance)
	mysqlBackup.POST("/instances/:instanceId/check-remote", d.mysqlBackupHandler.CheckRemote)
	mysqlBackup.POST("/instances/:instanceId/run", d.mysqlBackupHandler.RunBackup)
	mysqlBackup.GET("/jobs", d.mysqlBackupHandler.ListJobs)
	mysqlBackup.POST("/jobs/:jobId/stop", d.mysqlBackupHandler.StopJob)
	mysqlBackup.DELETE("/jobs/:jobId", d.mysqlBackupHandler.DeleteJob)
	mysqlBackup.GET("/jobs/:jobId/presign", d.mysqlBackupHandler.PresignJob)
}
