package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// AlertRouteDeps 告警插件路由依赖。

func (d *RouteDeps) AlertHandler() *handler.AlertHandler {
	if d == nil {
		return nil
	}
	return d.alertHandler
}

func (d *RouteDeps) AlertPlatformHandler() *handler.AlertPlatformHandler {
	if d == nil {
		return nil
	}
	return d.alertPlatformHandler
}

func (d *RouteDeps) AlertSubscriptionHandler() *handler.AlertSubscriptionHandler {
	if d == nil {
		return nil
	}
	return d.alertSubscriptionHandler
}

func (d *RouteDeps) AlertInhibitionHandler() *handler.AlertInhibitionHandler {
	if d == nil {
		return nil
	}
	return d.alertInhibitionHandler
}

func (d *RouteDeps) AlertReceiverGroupHandler() *handler.AlertReceiverGroupHandler {
	if d == nil {
		return nil
	}
	return d.alertReceiverGroupHandler
}

func (d *RouteDeps) CloudExpiryRuleHandler() *handler.CloudExpiryRuleHandler {
	if d == nil {
		return nil
	}
	return d.cloudExpiryRuleHandler
}

func (d *RouteDeps) PlatformFeaturesHandler() *handler.PlatformFeaturesHandler {
	if d == nil {
		return nil
	}
	return d.platformFeatures
}

func (d *RouteDeps) WorkflowHandler() *handler.WorkflowHandler {
	if d == nil {
		return nil
	}
	return d.workflowHandler
}

var _ AlertRouteDeps = (*RouteDeps)(nil)

var _ routedeps.AlertRouteDeps = (*RouteDeps)(nil)
