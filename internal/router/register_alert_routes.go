package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterAlertRoutes 告警平台 HTTP 路由。
func RegisterAlertRoutes(api *gin.RouterGroup, d AlertRouteDeps) {
	alertHandler := d.AlertHandler()
	alertPlatformHandler := d.AlertPlatformHandler()
	alertSubscriptionHandler := d.AlertSubscriptionHandler()
	alertReceiverGroupHandler := d.AlertReceiverGroupHandler()
	cloudExpiryRuleHandler := d.CloudExpiryRuleHandler()

	// 内部入站：Alertmanager Webhook（平台规则重复标签会跳过，避免双发）+ K8s Event。
	alertIngress := api.Group("/alerts")
	alertIngress.POST("/webhook", alertHandler.ReceiveAlertmanagerWebhook)
	alertIngress.POST("/ingress/k8s-events", alertHandler.ReceiveK8sEventIngress)

	alerts := api.Group("/alerts")
	alerts.Use(d.AuthMiddleware(), d.Authorize(), d.OpAudit())
	alerts.GET("/channels", alertHandler.ListChannels)
	alerts.POST("/channels", alertHandler.CreateChannel)
	alerts.PUT("/channels/:id", alertHandler.UpdateChannel)
	alerts.DELETE("/channels/:id", alertHandler.DeleteChannel)
	alerts.POST("/channels/:id/test", alertHandler.TestChannel)
	alerts.POST("/channels/preview-template", alertHandler.PreviewChannelTemplate)
	alerts.POST("/routing/debug", alertHandler.DebugRouting)
	alerts.GET("/events", alertHandler.ListEvents)
	alerts.GET("/events/grouped", alertHandler.ListEventsGrouped)
	alerts.GET("/events/by-fingerprint", alertHandler.ExplainFingerprintDelivery)
	alerts.GET("/events/evidence", alertHandler.CollectAlertEvidence)
	alerts.GET("/cur-events", alertHandler.ListCurEvents)
	alerts.GET("/his-events", alertHandler.ListHisEvents)
	alerts.GET("/his-events/export.csv", alertHandler.ExportHisEventsCSV)
	alerts.GET("/promql-saved-queries", alertHandler.ListPromqlSavedQueries)
	alerts.POST("/promql-saved-queries", alertHandler.CreatePromqlSavedQuery)
	alerts.DELETE("/promql-saved-queries/:id", alertHandler.DeletePromqlSavedQuery)
	if platformFeatures := d.PlatformFeaturesHandler(); platformFeatures != nil {
		alerts.GET("/monitor-rule-changes", platformFeatures.ListPendingRuleChanges)
		alerts.POST("/monitor-rule-changes", platformFeatures.ProposeRuleChange)
		alerts.POST("/monitor-rule-changes/:id/approve", platformFeatures.ApproveRuleChange)
		alerts.POST("/monitor-rule-changes/:id/reject", platformFeatures.RejectRuleChange)
	}
	alerts.POST("/acks", alertHandler.AcknowledgeAlert)
	alerts.DELETE("/acks", alertHandler.ClearAlertAck)
	alerts.GET("/acks", alertHandler.GetActiveAck)
	alerts.GET("/notes", alertHandler.ListAlertNotes)
	alerts.POST("/notes", alertHandler.CreateAlertNote)
	alerts.GET("/history/stats", alertHandler.HistoryStats)
	alerts.GET("/quality-report", alertHandler.QualityReport)

	alerts.GET("/datasources", alertPlatformHandler.ListDatasources)
	alerts.POST("/datasources", alertPlatformHandler.CreateDatasource)
	alerts.GET("/datasources/health", alertPlatformHandler.ListDatasourceHealth)
	alerts.GET("/datasources/:id/health", alertPlatformHandler.GetDatasourceHealth)
	alerts.POST("/datasources/:id/health-check", alertPlatformHandler.CheckDatasourceHealth)
	alerts.GET("/datasources/:id/ping", alertPlatformHandler.PingDatasource)
	alerts.GET("/datasources/:id/prometheus-alerts", alertPlatformHandler.PromActiveAlerts)
	alerts.POST("/datasources/:id/query", alertPlatformHandler.PromQuery)
	alerts.POST("/datasources/:id/query_range", alertPlatformHandler.PromQueryRange)
	alerts.PUT("/datasources/:id", alertPlatformHandler.UpdateDatasource)
	alerts.DELETE("/datasources/:id", alertPlatformHandler.DeleteDatasource)

	alerts.GET("/consul-endpoints", alertPlatformHandler.ListConsulEndpoints)
	alerts.POST("/consul-endpoints", alertPlatformHandler.CreateConsulEndpoint)
	alerts.PUT("/consul-endpoints/:id", alertPlatformHandler.UpdateConsulEndpoint)
	alerts.DELETE("/consul-endpoints/:id", alertPlatformHandler.DeleteConsulEndpoint)
	alerts.GET("/consul-endpoints/:id/ping", alertPlatformHandler.PingConsulEndpoint)
	alerts.POST("/consul-endpoints/:id/sync", alertPlatformHandler.SyncConsulEndpoint)
	alerts.GET("/monitor-objects", alertPlatformHandler.ListMonitorObjects)

	alerts.GET("/silences", alertPlatformHandler.ListSilences)
	alerts.POST("/silences", alertPlatformHandler.CreateSilence)
	alerts.POST("/silences/batch", alertPlatformHandler.CreateSilenceBatch)
	alerts.PUT("/silences/:id", alertPlatformHandler.UpdateSilence)
	alerts.DELETE("/silences/:id", alertPlatformHandler.DeleteSilence)

	alerts.GET("/maintenance-windows", alertPlatformHandler.ListMaintenanceWindows)
	alerts.POST("/maintenance-windows", alertPlatformHandler.CreateMaintenanceWindow)
	alerts.PUT("/maintenance-windows/:id", alertPlatformHandler.UpdateMaintenanceWindow)
	alerts.DELETE("/maintenance-windows/:id", alertPlatformHandler.DeleteMaintenanceWindow)

	alerts.GET("/monitor-rules", alertPlatformHandler.ListMonitorRules)
	alerts.POST("/monitor-rules", alertPlatformHandler.CreateMonitorRule)
	alerts.POST("/monitor-rules/import-prometheus-yaml", alertPlatformHandler.ImportPrometheusYAML)
	alerts.GET("/rule-templates", alertPlatformHandler.ListRuleTemplates)
	alerts.POST("/monitor-rules/from-template", alertPlatformHandler.CreateMonitorRuleFromTemplate)
	alerts.PUT("/monitor-rules/:id", alertPlatformHandler.UpdateMonitorRule)
	alerts.DELETE("/monitor-rules/:id", alertPlatformHandler.DeleteMonitorRule)
	alerts.GET("/monitor-rules/:id/assignees", alertPlatformHandler.GetMonitorRuleAssignees)
	alerts.PUT("/monitor-rules/:id/assignees", alertPlatformHandler.UpsertMonitorRuleAssignees)
	alerts.GET("/duty-blocks", alertPlatformHandler.ListDutyBlocks)
	alerts.POST("/duty-blocks", alertPlatformHandler.CreateDutyBlock)
	alerts.PUT("/duty-blocks/:id", alertPlatformHandler.UpdateDutyBlock)
	alerts.DELETE("/duty-blocks/:id", alertPlatformHandler.DeleteDutyBlock)
	alerts.GET("/duty-blocks/calendar", alertPlatformHandler.ListDutyCalendar)
	alerts.POST("/duty-blocks/validate", alertPlatformHandler.ValidateDutyBlocks)
	alerts.POST("/duty-blocks/:id/handoff", alertPlatformHandler.HandoffDutyBlock)

	alerts.GET("/subscriptions", alertSubscriptionHandler.ListNodes)
	alerts.GET("/subscriptions/tree", alertSubscriptionHandler.GetNodeTree)
	alerts.POST("/subscriptions", alertSubscriptionHandler.CreateNode)
	alerts.PUT("/subscriptions/:id", alertSubscriptionHandler.UpdateNode)
	alerts.DELETE("/subscriptions/:id", alertSubscriptionHandler.DeleteNode)
	alerts.POST("/subscriptions/:id/move", alertSubscriptionHandler.MoveNode)
	alerts.POST("/subscriptions/wizard", alertSubscriptionHandler.ApplyRoutingWizard)
	alerts.POST("/subscriptions/migrate-from-policies", alertSubscriptionHandler.MigrateFromPolicies)
	alerts.POST("/subscriptions/clone-from-project", alertSubscriptionHandler.CloneProjectRouting)

	if alertInhibitionHandler := d.AlertInhibitionHandler(); alertInhibitionHandler != nil {
		alerts.GET("/inhibition-rules", alertInhibitionHandler.List)
		alerts.POST("/inhibition-rules", alertInhibitionHandler.Create)
		alerts.PUT("/inhibition-rules/:id", alertInhibitionHandler.Update)
		alerts.DELETE("/inhibition-rules/:id", alertInhibitionHandler.Delete)
		alerts.POST("/inhibition-rules/refresh-cache", alertInhibitionHandler.RefreshCache)
	}

	alerts.GET("/receiver-groups", alertReceiverGroupHandler.List)
	alerts.POST("/receiver-groups", alertReceiverGroupHandler.Create)
	alerts.PUT("/receiver-groups/:id", alertReceiverGroupHandler.Update)
	alerts.DELETE("/receiver-groups/:id", alertReceiverGroupHandler.Delete)

	alerts.GET("/cloud-expiry-rules", cloudExpiryRuleHandler.List)
	alerts.POST("/cloud-expiry-rules", cloudExpiryRuleHandler.Create)
	alerts.PUT("/cloud-expiry-rules/:id", cloudExpiryRuleHandler.Update)
	alerts.DELETE("/cloud-expiry-rules/:id", cloudExpiryRuleHandler.Delete)
	alerts.POST("/cloud-expiry-rules/evaluate-now", cloudExpiryRuleHandler.EvaluateNow)

	if workflowHandler := d.WorkflowHandler(); workflowHandler != nil {
		alerts.POST("/events/:alert_event_id/to-ticket", workflowHandler.CreateIncidentFromAlert)
	}
}
