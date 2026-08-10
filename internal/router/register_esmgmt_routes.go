package router

import "github.com/gin-gonic/gin"

func RegisterEsmgmtRoutes(api *gin.RouterGroup, d *RouteDeps) {
	if d == nil || d.esmgmtHandler == nil {
		return
	}
	g := api.Group("/esmgmt")
	g.Use(d.authMiddleware, d.authorize, d.opAudit)

	g.GET("/connections", d.esmgmtHandler.ListConnections)
	g.POST("/connections", d.esmgmtHandler.CreateConnection)
	g.POST("/connections/test", d.esmgmtHandler.TestConnection)
	g.PUT("/connections/:id", d.esmgmtHandler.UpdateConnection)
	g.DELETE("/connections/:id", d.esmgmtHandler.DeleteConnection)
	g.POST("/connections/:id/ping", d.esmgmtHandler.PingConnection)

	g.GET("/cluster/health", d.esmgmtHandler.ClusterHealth)
	g.GET("/indices", d.esmgmtHandler.ListIndices)
	g.DELETE("/indices/:name", d.esmgmtHandler.DeleteIndex)
	g.POST("/indices/:name/open", d.esmgmtHandler.OpenIndex)
	g.POST("/indices/:name/close", d.esmgmtHandler.CloseIndex)
	g.GET("/nodes", d.esmgmtHandler.CatNodes)
	g.POST("/proxy", d.esmgmtHandler.ProxyREST)

	g.POST("/backups", d.esmgmtHandler.CreateIndexBackup)
	g.GET("/backups", d.esmgmtHandler.ListBackupJobs)
	g.GET("/backups/:id", d.esmgmtHandler.GetBackupJob)
	g.GET("/backups/:id/download", d.esmgmtHandler.DownloadBackup)

	g.POST("/restores", d.esmgmtHandler.CreateIndexRestore)
	g.GET("/restores", d.esmgmtHandler.ListRestoreJobs)
	g.GET("/restores/:id", d.esmgmtHandler.GetRestoreJob)

	g.GET("/schedules", d.esmgmtHandler.ListSchedules)
	g.POST("/schedules", d.esmgmtHandler.CreateSchedule)
	g.PUT("/schedules/:id", d.esmgmtHandler.UpdateSchedule)
	g.DELETE("/schedules/:id", d.esmgmtHandler.DeleteSchedule)
}
