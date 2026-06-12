package router

import (
	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
	"yunshu/internal/interfaces"
	"yunshu/internal/middleware"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// RouteDeps 聚合路由注册所需的 handler、中间件与共享仓储（插件路由注册入参）。
type RouteDeps struct {
	app *bootstrap.App

	authMiddleware    gin.HandlerFunc
	wsAuthMiddleware  gin.HandlerFunc
	authorize         gin.HandlerFunc
	k8sScopeAuthorize gin.HandlerFunc
	opAudit           gin.HandlerFunc

	projectMemberRepo interfaces.ProjectMemberRepository
	clusterRepo       interfaces.K8sClusterRepository
	k8sRuntimeService *service.K8sRuntimeService

	systemHandler *handler.SystemHandler
	pluginHandler *handler.PluginHandler

	authHandler              *handler.AuthHandler
	loginLogHandler          *handler.LoginLogHandler
	opLogHandler             *handler.OperationLogHandler
	userHandler              *handler.UserHandler
	departmentHandler        *handler.DepartmentHandler
	roleHandler              *handler.RoleHandler
	permissionHandler        *handler.PermissionHandler
	policyHandler            *handler.PolicyHandler
	k8sScopedPolicyHandler   *handler.K8sScopedPolicyHandler
	k8sNamespaceDenyHandler  *handler.K8sNamespaceDenyHandler
	k8sNamespaceAllowHandler *handler.K8sNamespaceAllowHandler
	userGroupHandler         *handler.UserGroupHandler
	regHandler               *handler.RegistrationHandler
	menuHandler              *handler.MenuHandler
	dictEntryHandler         *handler.DictEntryHandler
	adminHandler             *handler.AdminHandler

	alertHandler             *handler.AlertHandler
	alertPlatformHandler     *handler.AlertPlatformHandler
	alertSubscriptionHandler *handler.AlertSubscriptionHandler
	alertInhibitionHandler   *handler.AlertInhibitionHandler
	alertReceiverGroupHandler *handler.AlertReceiverGroupHandler
	cloudExpiryRuleHandler   *handler.CloudExpiryRuleHandler

	clusterHandler           *handler.ClusterHandler
	podHandler               *handler.PodHandler
	namespaceHandler         *handler.NamespaceHandler
	nodeHandler              *handler.NodeHandler
	workloadHandler          *handler.WorkloadHandler
	configHandler            *handler.ConfigHandler
	storageHandler           *handler.StorageHandler
	serviceResourceHandler   *handler.ServiceResourceHandler
	ingressHandler           *handler.IngressHandler
	networkPolicyHandler     *handler.NetworkPolicyHandler
	k8sDiscoveryHandler      *handler.K8sDiscoveryHandler
	k8sHPAHandler            *handler.K8sHPAHandler
	k8sResourceWatchHandler  *handler.K8sResourceWatchHandler
	k8sSearchHandler         *handler.K8sSearchHandler
	k8sEventForwardHandler   *handler.K8sEventForwardHandler
	eventHandler             *handler.EventHandler
	crdHandler               *handler.CRDHandler
	crHandler                *handler.CRHandler
	rbacHandler              *handler.RBACHandler
	serviceAccountHandler    *handler.ServiceAccountHandler
	overviewHandler          *handler.OverviewHandler

	projectHandler         *handler.ProjectHandler
	cmdbHandler            *handler.CMDBHandler
	mysqlBackupSvc         *service.MysqlBackupService
	mysqlBackupHandler     *handler.MysqlBackupHandler
	logAgentHandler        *handler.LogAgentHandler
	agentDiscoveryHandler  *handler.AgentDiscoveryHandler
}

// K8sRuntimeService 供 k8s 插件后台任务使用。
func (d *RouteDeps) K8sRuntimeService() *service.K8sRuntimeService {
	if d == nil {
		return nil
	}
	return d.k8sRuntimeService
}

// MysqlBackupService 供 backup 插件调度器使用。
func (d *RouteDeps) MysqlBackupService() *service.MysqlBackupService {
	if d == nil {
		return nil
	}
	return d.mysqlBackupSvc
}

func assembleRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, repos *routeRepositories, svcs *routeServices) (*RouteDeps, error) {
	if repos == nil {
		repos = newRouteRepositories(app.DB)
	}
	return wireRouteDepsWithRepos(app, runtimeClient, repos, svcs)
}

func wireRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient) (*RouteDeps, error) {
	return assembleRouteDeps(app, runtimeClient, newRouteRepositories(app.DB), nil)
}

func wireRouteDepsWithRepos(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, repos *routeRepositories, svcs *routeServices) (*RouteDeps, error) {
	if svcs == nil {
		var err error
		svcs, err = buildRouteServices(app, repos)
		if err != nil {
			return nil, err
		}
	}

	systemHandler := handler.NewSystemHandler(app.Config.App.Name, app.Config.App.Env)
	pluginHandler := handler.NewPluginHandler(&app.Config.Plugins)
	userRepo := repos.User
	permissionRepo := repos.Permission
	projectMemberRepo := repos.ProjectMember
	k8sNsDenyRepo := repos.K8sNsDeny
	k8sNsAllowRepo := repos.K8sNsAllow
	k8sClusterAccessRepo := repos.K8sClusterAccess
	clusterRepo := repos.Cluster

	k8sNamespaceDenyHandler := handler.NewK8sNamespaceDenyHandler(svcs.K8sNamespaceDeny)
	k8sNamespaceAllowHandler := handler.NewK8sNamespaceAllowHandler(svcs.K8sNamespaceAllow)
	userGroupHandler := handler.NewUserGroupHandler(svcs.UserGroup)

	mysqlBackupHandler := handler.NewMysqlBackupHandler(svcs.MysqlBackup)

	authHandler := handler.NewAuthHandler(svcs.Auth, svcs.LoginLog)
	loginLogHandler := handler.NewLoginLogHandler(svcs.LoginLog)
	opLogHandler := handler.NewOperationLogHandler(svcs.OperationLog)
	userHandler := handler.NewUserHandler(svcs.User)
	departmentHandler := handler.NewDepartmentHandler(svcs.Department)
	roleHandler := handler.NewRoleHandler(svcs.Role)
	permissionHandler := handler.NewPermissionHandler(svcs.Permission)
	policyHandler := handler.NewPolicyHandler(svcs.Policy)
	k8sScopedPolicyHandler := handler.NewK8sScopedPolicyHandler(svcs.K8sScopedPolicy)
	regHandler := handler.NewRegistrationHandler(svcs.Registration)
	menuHandler := handler.NewMenuHandler(svcs.Menu)
	dictEntryHandler := handler.NewDictEntryHandler(svcs.DictEntry)
	alertHandler := handler.NewAlertHandler(svcs.Alert)
	cloudExpiryRuleHandler := handler.NewCloudExpiryRuleHandler(svcs.CloudExpiryRule, svcs.Alert)
	alertPlatformHandler := handler.NewAlertPlatformHandler(svcs.AlertDatasource, svcs.AlertSilence, svcs.AlertMaintenance, svcs.AlertMonitorRule, svcs.AlertAssignee, svcs.AlertDuty)
	alertSubscriptionSvc := svcs.Alert.GetSubscriptionService()
	alertSubscriptionHandler := handler.NewAlertSubscriptionHandler(alertSubscriptionSvc)
	var alertInhibitionHandler *handler.AlertInhibitionHandler
	if inh := svcs.Alert.GetInhibitionService(); inh != nil {
		alertInhibitionHandler = handler.NewAlertInhibitionHandler(inh)
	}
	alertReceiverGroupHandler := handler.NewAlertReceiverGroupHandler(svcs.AlertReceiverGroup)
	adminHandler := handler.NewAdminHandler(app.Redis)
	clusterHandler := handler.NewClusterHandler(svcs.K8sCluster)
	podHandler := handler.NewPodHandler(svcs.K8sPod)
	namespaceHandler := handler.NewNamespaceHandler(svcs.K8sNamespace)
	nodeHandler := handler.NewNodeHandler(svcs.K8sNode)
	workloadHandler := handler.NewWorkloadHandler(svcs.K8sWorkload)
	configHandler := handler.NewConfigHandler(svcs.K8sConfig)
	storageHandler := handler.NewStorageHandler(svcs.K8sStorage)
	serviceResourceHandler := handler.NewServiceResourceHandler(svcs.K8sServiceResource)
	ingressHandler := handler.NewIngressHandler(svcs.K8sIngress)
	networkPolicyHandler := handler.NewNetworkPolicyHandler(svcs.K8sNetworkPolicy)
	k8sDiscoveryHandler := handler.NewK8sDiscoveryHandler(svcs.K8sDiscovery)
	k8sHPAHandler := handler.NewK8sHPAHandler(svcs.K8sHPA)
	k8sResourceWatchHandler := handler.NewK8sResourceWatchHandler(svcs.K8sRuntime)
	k8sSearchHandler := handler.NewK8sSearchHandler(svcs.K8sSearch)
	k8sEventForwardHandler := handler.NewK8sEventForwardHandler(svcs.K8sEventForwardAdmin)
	eventHandler := handler.NewEventHandler(svcs.K8sEvent)
	crdHandler := handler.NewCRDHandler(svcs.K8sCRD)
	crHandler := handler.NewCRHandler(svcs.K8sCR)
	rbacHandler := handler.NewRBACHandler(svcs.K8sRBAC)
	serviceAccountHandler := handler.NewServiceAccountHandler(svcs.K8sServiceAccount)
	overviewHandler := handler.NewOverviewHandler(svcs.Overview)
	projectHandler := handler.NewProjectHandler(svcs.ProjectMgmt, runtimeClient.ProjectSrv, runtimeClient.LogSourceSrv)
	cmdbHandler := handler.NewCMDBHandler(svcs.CMDB)
	logAgentHandler := handler.NewLogAgentHandler(svcs.LogAgent, runtimeClient.AgentSrv)
	agentDiscoveryHandler := handler.NewAgentDiscoveryHandler(svcs.AgentDiscovery, runtimeClient.AgentSrv)

	authMiddleware := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, userRepo, app.Logger)
	wsAuthMiddleware := middleware.WSAuth(app.Redis, userRepo, app.Logger)
	authorize := middleware.Authorize(app.Enforcer, app.Logger, k8sClusterAccessRepo)
	k8sScopeAuthorize := middleware.K8sScopeAuthorize(app.Logger, permissionRepo, k8sClusterAccessRepo, k8sNsDenyRepo, k8sNsAllowRepo)
	opAudit := middleware.OperationAudit(svcs.OperationLog, app.Logger)

	return &RouteDeps{
		app: app,

		authMiddleware:    authMiddleware,
		wsAuthMiddleware:  wsAuthMiddleware,
		authorize:         authorize,
		k8sScopeAuthorize: k8sScopeAuthorize,
		opAudit:           opAudit,

		projectMemberRepo: projectMemberRepo,
		clusterRepo:       clusterRepo,
		k8sRuntimeService: svcs.K8sRuntime,

		systemHandler: systemHandler,
		pluginHandler: pluginHandler,

		authHandler:              authHandler,
		loginLogHandler:          loginLogHandler,
		opLogHandler:             opLogHandler,
		userHandler:              userHandler,
		departmentHandler:        departmentHandler,
		roleHandler:              roleHandler,
		permissionHandler:        permissionHandler,
		policyHandler:            policyHandler,
		k8sScopedPolicyHandler:   k8sScopedPolicyHandler,
		k8sNamespaceDenyHandler:  k8sNamespaceDenyHandler,
		k8sNamespaceAllowHandler: k8sNamespaceAllowHandler,
		userGroupHandler:         userGroupHandler,
		regHandler:               regHandler,
		menuHandler:              menuHandler,
		dictEntryHandler:         dictEntryHandler,
		adminHandler:             adminHandler,

		alertHandler:              alertHandler,
		alertPlatformHandler:      alertPlatformHandler,
		alertSubscriptionHandler:  alertSubscriptionHandler,
		alertInhibitionHandler:    alertInhibitionHandler,
		alertReceiverGroupHandler: alertReceiverGroupHandler,
		cloudExpiryRuleHandler:    cloudExpiryRuleHandler,

		clusterHandler:          clusterHandler,
		podHandler:              podHandler,
		namespaceHandler:        namespaceHandler,
		nodeHandler:             nodeHandler,
		workloadHandler:         workloadHandler,
		configHandler:           configHandler,
		storageHandler:          storageHandler,
		serviceResourceHandler:  serviceResourceHandler,
		ingressHandler:          ingressHandler,
		networkPolicyHandler:    networkPolicyHandler,
		k8sDiscoveryHandler:     k8sDiscoveryHandler,
		k8sHPAHandler:           k8sHPAHandler,
		k8sResourceWatchHandler: k8sResourceWatchHandler,
		k8sSearchHandler:        k8sSearchHandler,
		k8sEventForwardHandler:  k8sEventForwardHandler,
		eventHandler:            eventHandler,
		crdHandler:              crdHandler,
		crHandler:               crHandler,
		rbacHandler:             rbacHandler,
		serviceAccountHandler:   serviceAccountHandler,
		overviewHandler:         overviewHandler,

		projectHandler:        projectHandler,
		cmdbHandler:           cmdbHandler,
		mysqlBackupSvc:        svcs.MysqlBackup,
		mysqlBackupHandler:    mysqlBackupHandler,
		logAgentHandler:       logAgentHandler,
		agentDiscoveryHandler: agentDiscoveryHandler,
	}, nil
}
