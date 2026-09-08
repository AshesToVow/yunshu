package router

import "github.com/gin-gonic/gin"

func RegisterEsmgmtRoutes(api *gin.RouterGroup, d EsmgmtRouteDeps) {
	if d == nil || d.EsmgmtHandler() == nil {
		return
	}
	h := d.EsmgmtHandler()
	g := api.Group("/esmgmt")
	g.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())

	g.GET("/connections", h.ListConnections)
	g.POST("/connections", h.CreateConnection)
	g.POST("/connections/import-from-dict", h.ImportConnectionFromDict)
	g.POST("/connections/test", h.TestConnection)
	g.PUT("/connections/:id", h.UpdateConnection)
	g.DELETE("/connections/:id", h.DeleteConnection)
	g.POST("/connections/:id/ping", h.PingConnection)

	g.GET("/cluster/health", h.ClusterHealth)
	g.GET("/indices", h.ListIndices)
	g.POST("/indices", h.CreateIndex)
	g.DELETE("/indices/:name", h.DeleteIndex)
	g.POST("/indices/:name/open", h.OpenIndex)
	g.POST("/indices/:name/close", h.CloseIndex)
	g.GET("/nodes", h.CatNodes)
	g.POST("/proxy", h.ProxyREST)

	g.POST("/backups", h.CreateIndexBackup)
	g.GET("/backups", h.ListBackupJobs)
	g.GET("/backups/:id", h.GetBackupJob)
	g.GET("/backups/:id/download", h.DownloadBackup)

	g.POST("/restores", h.CreateIndexRestore)
	g.GET("/restores", h.ListRestoreJobs)
	g.GET("/restores/:id", h.GetRestoreJob)

	g.GET("/schedules", h.ListSchedules)
	g.POST("/schedules", h.CreateSchedule)
	g.PUT("/schedules/:id", h.UpdateSchedule)
	g.DELETE("/schedules/:id", h.DeleteSchedule)
}
