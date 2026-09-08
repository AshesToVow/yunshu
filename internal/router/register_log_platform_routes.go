package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterLogPlatformRoutes 日志平台全局能力（保留策略、ES 存储概览）。
func RegisterLogPlatformRoutes(api *gin.RouterGroup, d LogPlatformRouteDeps) {
	g := api.Group("/log-platform")
	g.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	g.GET("/retention", d.LogPlatformHandler().GetGlobalRetention)
	g.PUT("/retention", d.LogPlatformHandler().UpsertGlobalRetention)
	g.GET("/retention/list", d.LogPlatformHandler().ListRetentionPolicies)
	g.GET("/es-storage", d.LogPlatformHandler().StorageStats)
	g.DELETE("/es-indices/:index", d.LogPlatformHandler().DeleteESIndex)
	g.GET("/es-config", d.LoggieHandler().ESConfigPreview)
	g.PUT("/es-connection", d.LoggieHandler().SetESConnection)
	g.GET("/kafka-stats", d.LogPlatformHandler().KafkaStats)
	g.GET("/kafka-config", d.LogPlatformHandler().KafkaConfigPreview)
	g.DELETE("/kafka-topics/:topic", d.LogPlatformHandler().DeleteKafkaTopic)
	g.POST("/retention/cleanup", d.LogPlatformHandler().RunCleanup)

	api.POST("/loggie/heartbeat/report", d.LoggieHandler().ReportHeartbeat)
}
