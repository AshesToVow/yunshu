package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterAlertRoutes 告警平台 HTTP 路由。
func RegisterAlertRoutes(api *gin.RouterGroup, d *RouteDeps) {
	alertWebhook := api.Group("/alerts")
	alertWebhook.POST("/webhook/alertmanager", d.alertHandler.ReceiveAlertmanager)

	alerts := api.Group("/alerts")
	alerts.Use(d.authMiddleware, d.authorize, d.opAudit)
	alerts.GET("/channels", d.alertHandler.ListChannels)
	alerts.POST("/channels", d.alertHandler.CreateChannel)
	alerts.PUT("/channels/:id", d.alertHandler.UpdateChannel)
	alerts.DELETE("/channels/:id", d.alertHandler.DeleteChannel)
	alerts.POST("/channels/:id/test", d.alertHandler.TestChannel)
	alerts.POST("/channels/preview-template", d.alertHandler.PreviewChannelTemplate)
	alerts.POST("/routing/debug", d.alertHandler.DebugRouting)
	alerts.GET("/events", d.alertHandler.ListEvents)
	alerts.GET("/events/grouped", d.alertHandler.ListEventsGrouped)
	alerts.GET("/events/by-fingerprint", d.alertHandler.ExplainFingerprintDelivery)
	alerts.GET("/history/stats", d.alertHandler.HistoryStats)

	alerts.GET("/datasources", d.alertPlatformHandler.ListDatasources)
	alerts.POST("/datasources", d.alertPlatformHandler.CreateDatasource)
	alerts.GET("/datasources/:id/ping", d.alertPlatformHandler.PingDatasource)
	alerts.GET("/datasources/:id/prometheus-alerts", d.alertPlatformHandler.PromActiveAlerts)
	alerts.GET("/datasources/:id/alertmanager-silences", d.alertPlatformHandler.AlertmanagerSilences)
	alerts.POST("/datasources/:id/query", d.alertPlatformHandler.PromQuery)
	alerts.POST("/datasources/:id/query_range", d.alertPlatformHandler.PromQueryRange)
	alerts.PUT("/datasources/:id", d.alertPlatformHandler.UpdateDatasource)
	alerts.DELETE("/datasources/:id", d.alertPlatformHandler.DeleteDatasource)

	alerts.GET("/silences", d.alertPlatformHandler.ListSilences)
	alerts.POST("/silences", d.alertPlatformHandler.CreateSilence)
	alerts.POST("/silences/batch", d.alertPlatformHandler.CreateSilenceBatch)
	alerts.PUT("/silences/:id", d.alertPlatformHandler.UpdateSilence)
	alerts.DELETE("/silences/:id", d.alertPlatformHandler.DeleteSilence)

	alerts.GET("/maintenance-windows", d.alertPlatformHandler.ListMaintenanceWindows)
	alerts.POST("/maintenance-windows", d.alertPlatformHandler.CreateMaintenanceWindow)
	alerts.PUT("/maintenance-windows/:id", d.alertPlatformHandler.UpdateMaintenanceWindow)
	alerts.DELETE("/maintenance-windows/:id", d.alertPlatformHandler.DeleteMaintenanceWindow)

	alerts.GET("/monitor-rules", d.alertPlatformHandler.ListMonitorRules)
	alerts.POST("/monitor-rules", d.alertPlatformHandler.CreateMonitorRule)
	alerts.PUT("/monitor-rules/:id", d.alertPlatformHandler.UpdateMonitorRule)
	alerts.DELETE("/monitor-rules/:id", d.alertPlatformHandler.DeleteMonitorRule)
	alerts.GET("/monitor-rules/:id/assignees", d.alertPlatformHandler.GetMonitorRuleAssignees)
	alerts.PUT("/monitor-rules/:id/assignees", d.alertPlatformHandler.UpsertMonitorRuleAssignees)
	alerts.GET("/duty-blocks", d.alertPlatformHandler.ListDutyBlocks)
	alerts.POST("/duty-blocks", d.alertPlatformHandler.CreateDutyBlock)
	alerts.PUT("/duty-blocks/:id", d.alertPlatformHandler.UpdateDutyBlock)
	alerts.DELETE("/duty-blocks/:id", d.alertPlatformHandler.DeleteDutyBlock)
	alerts.GET("/duty-blocks/calendar", d.alertPlatformHandler.ListDutyCalendar)
	alerts.POST("/duty-blocks/validate", d.alertPlatformHandler.ValidateDutyBlocks)
	alerts.POST("/duty-blocks/:id/handoff", d.alertPlatformHandler.HandoffDutyBlock)

	alerts.GET("/subscriptions", d.alertSubscriptionHandler.ListNodes)
	alerts.GET("/subscriptions/tree", d.alertSubscriptionHandler.GetNodeTree)
	alerts.POST("/subscriptions", d.alertSubscriptionHandler.CreateNode)
	alerts.PUT("/subscriptions/:id", d.alertSubscriptionHandler.UpdateNode)
	alerts.DELETE("/subscriptions/:id", d.alertSubscriptionHandler.DeleteNode)
	alerts.POST("/subscriptions/:id/move", d.alertSubscriptionHandler.MoveNode)
	alerts.POST("/subscriptions/migrate-from-policies", d.alertSubscriptionHandler.MigrateFromPolicies)
	alerts.POST("/subscriptions/clone-from-project", d.alertSubscriptionHandler.CloneProjectRouting)

	if d.alertInhibitionHandler != nil {
		alerts.GET("/inhibition-rules", d.alertInhibitionHandler.List)
		alerts.POST("/inhibition-rules", d.alertInhibitionHandler.Create)
		alerts.PUT("/inhibition-rules/:id", d.alertInhibitionHandler.Update)
		alerts.DELETE("/inhibition-rules/:id", d.alertInhibitionHandler.Delete)
		alerts.POST("/inhibition-rules/refresh-cache", d.alertInhibitionHandler.RefreshCache)
	}

	alerts.GET("/receiver-groups", d.alertReceiverGroupHandler.List)
	alerts.POST("/receiver-groups", d.alertReceiverGroupHandler.Create)
	alerts.PUT("/receiver-groups/:id", d.alertReceiverGroupHandler.Update)
	alerts.DELETE("/receiver-groups/:id", d.alertReceiverGroupHandler.Delete)

	alerts.GET("/cloud-expiry-rules", d.cloudExpiryRuleHandler.List)
	alerts.POST("/cloud-expiry-rules", d.cloudExpiryRuleHandler.Create)
	alerts.PUT("/cloud-expiry-rules/:id", d.cloudExpiryRuleHandler.Update)
	alerts.DELETE("/cloud-expiry-rules/:id", d.cloudExpiryRuleHandler.Delete)
	alerts.POST("/cloud-expiry-rules/evaluate-now", d.cloudExpiryRuleHandler.EvaluateNow)
}
