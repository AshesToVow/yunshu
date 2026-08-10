package router

import (
	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/handler"
	"yunshu/internal/service"

	"github.com/google/wire"
)

func providePluginsConfig(app *bootstrap.App) *config.PluginsConfig {
	if app == nil || app.Config == nil {
		return &config.PluginsConfig{}
	}
	return &app.Config.Plugins
}

func provideSystemHandler(app *bootstrap.App) *handler.SystemHandler {
	name, env := "", ""
	if app != nil && app.Config != nil {
		name = app.Config.App.Name
		env = app.Config.App.Env
	}
	return handler.NewSystemHandler(name, env)
}

func provideAlertSubscriptionHandler(alertSvc *service.AlertService) *handler.AlertSubscriptionHandler {
	return handler.NewAlertSubscriptionHandler(alertSvc.GetSubscriptionService())
}

func provideAlertInhibitionHandler(alertSvc *service.AlertService) *handler.AlertInhibitionHandler {
	if inh := alertSvc.GetInhibitionService(); inh != nil {
		return handler.NewAlertInhibitionHandler(inh)
	}
	return nil
}

// HandlerSet wires all HTTP handlers into routeHandlers.
var HandlerSet = wire.NewSet(
	provideSystemHandler,
	handler.NewPluginHandler,
	handler.NewAuthHandler,
	handler.NewLoginLogHandler,
	handler.NewOperationLogHandler,
	handler.NewUserHandler,
	handler.NewDepartmentHandler,
	handler.NewRoleHandler,
	handler.NewPermissionHandler,
	handler.NewPolicyHandler,
	handler.NewK8sScopedPolicyHandler,
	handler.NewK8sNamespaceDenyHandler,
	handler.NewK8sNamespaceAllowHandler,
	handler.NewUserGroupHandler,
	handler.NewRegistrationHandler,
	handler.NewMenuHandler,
	handler.NewDictEntryHandler,
	handler.NewAdminHandler,
	handler.NewAlertHandler,
	handler.NewAlertPlatformHandler,
	provideAlertSubscriptionHandler,
	provideAlertInhibitionHandler,
	handler.NewAlertReceiverGroupHandler,
	handler.NewCloudExpiryRuleHandler,
	handler.NewClusterHandler,
	handler.NewPodHandler,
	handler.NewNamespaceHandler,
	handler.NewNodeHandler,
	handler.NewWorkloadHandler,
	handler.NewConfigHandler,
	handler.NewStorageHandler,
	handler.NewServiceResourceHandler,
	handler.NewIngressHandler,
	handler.NewNetworkPolicyHandler,
	handler.NewK8sDiscoveryHandler,
	handler.NewK8sHPAHandler,
	handler.NewHelmHandler,
	handler.NewK8sResourceWatchHandler,
	handler.NewK8sSearchHandler,
	handler.NewK8sEventForwardHandler,
	handler.NewEventHandler,
	handler.NewCRDHandler,
	handler.NewCRHandler,
	handler.NewRBACHandler,
	handler.NewServiceAccountHandler,
	handler.NewOverviewHandler,
	handler.NewProjectHandler,
	handler.NewProjectCatalogHandler,
	handler.NewCMDBHandler,
	handler.NewMysqlBackupHandler,
	handler.NewDbmgmtHandler,
	handler.NewCicdHandler,
	handler.NewLogPlatformHandler,
	handler.NewLoggieHandler,
	handler.NewInspectHandler,
	handler.NewAIHandler,
	handler.NewEsmgmtHandler,
	wire.Struct(new(routeHandlers), "*"),
)
