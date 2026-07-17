package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterLogPlatformRoutes 日志平台全局能力（保留策略、ES 存储概览）。
func RegisterLogPlatformRoutes(api *gin.RouterGroup, d *RouteDeps) {
	g := api.Group("/log-platform")
	g.Use(d.authMiddleware, d.authorize, d.opAudit)
	g.GET("/retention", d.logPlatformHandler.GetGlobalRetention)
	g.PUT("/retention", d.logPlatformHandler.UpsertGlobalRetention)
	g.GET("/retention/list", d.logPlatformHandler.ListRetentionPolicies)
	g.GET("/es-storage", d.logPlatformHandler.StorageStats)
	g.DELETE("/es-indices/:index", d.logPlatformHandler.DeleteESIndex)
	g.GET("/es-config", d.loggieHandler.ESConfigPreview)
	g.GET("/kafka-stats", d.logPlatformHandler.KafkaStats)
	g.GET("/kafka-config", d.logPlatformHandler.KafkaConfigPreview)
	g.DELETE("/kafka-topics/:topic", d.logPlatformHandler.DeleteKafkaTopic)
	g.POST("/retention/cleanup", d.logPlatformHandler.RunCleanup)

	api.POST("/loggie/heartbeat/report", d.loggieHandler.ReportHeartbeat)
}
