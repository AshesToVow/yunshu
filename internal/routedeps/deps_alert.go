package routedeps

import (
	"yunshu/internal/handler"
)

type AlertRouteDeps interface {
	RouteMiddleware
	AlertHandler() *handler.AlertHandler
	AlertPlatformHandler() *handler.AlertPlatformHandler
	AlertSubscriptionHandler() *handler.AlertSubscriptionHandler
	AlertInhibitionHandler() *handler.AlertInhibitionHandler
	AlertReceiverGroupHandler() *handler.AlertReceiverGroupHandler
	CloudExpiryRuleHandler() *handler.CloudExpiryRuleHandler
	PlatformFeaturesHandler() *handler.PlatformFeaturesHandler
	WorkflowHandler() *handler.WorkflowHandler
}
