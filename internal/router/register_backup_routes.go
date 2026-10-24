package router

import (
	"yunshu/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterBackupRoutes MySQL 备份（挂在项目作用域下）。
func RegisterBackupRoutes(api *gin.RouterGroup, d BackupRouteDeps) {
	projectRoutes := api.Group("/projects")
	projectRoutes.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	projectScoped := projectRoutes.Group("/:id", middleware.RequireProjectMemberAccess(d.ProjectMemberRepo(), d.ProjectRepo(), d.AppLogger()))

	mysqlBackup := projectScoped.Group("/mysql-backup")
	mysqlBackup.GET("/mysqldump-options", d.MysqlBackupHandler().ListMysqldumpOptions)
	mysqlBackup.GET("/instances", d.MysqlBackupHandler().ListInstances)
	mysqlBackup.POST("/instances", d.MysqlBackupHandler().CreateInstance)
	mysqlBackup.PUT("/instances/:instanceId", d.MysqlBackupHandler().UpdateInstance)
	mysqlBackup.DELETE("/instances/:instanceId", d.MysqlBackupHandler().DeleteInstance)
	mysqlBackup.POST("/instances/:instanceId/ping", d.MysqlBackupHandler().PingInstance)
	mysqlBackup.POST("/instances/:instanceId/check-remote", d.MysqlBackupHandler().CheckRemote)
	mysqlBackup.POST("/instances/:instanceId/run", d.MysqlBackupHandler().RunBackup)
	mysqlBackup.GET("/jobs", d.MysqlBackupHandler().ListJobs)
	mysqlBackup.POST("/jobs/:jobId/stop", d.MysqlBackupHandler().StopJob)
	mysqlBackup.DELETE("/jobs/:jobId", d.MysqlBackupHandler().DeleteJob)
	mysqlBackup.GET("/jobs/:jobId/presign", d.MysqlBackupHandler().PresignJob)
}
