package router

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/bootstrap"
	"yunshu/internal/handler"
	"yunshu/internal/interfaces"
	"yunshu/internal/middleware"
	"yunshu/internal/service"
	cicdsvc "yunshu/internal/service/cicd"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
	esmgmtsvc "yunshu/internal/service/esmgmt"
	inspectsvc "yunshu/internal/service/inspect"

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

	alertHandler              *handler.AlertHandler
	alertPlatformHandler      *handler.AlertPlatformHandler
	alertSubscriptionHandler  *handler.AlertSubscriptionHandler
	alertInhibitionHandler    *handler.AlertInhibitionHandler
	alertReceiverGroupHandler *handler.AlertReceiverGroupHandler
	cloudExpiryRuleHandler    *handler.CloudExpiryRuleHandler

	clusterHandler          *handler.ClusterHandler
	podHandler              *handler.PodHandler
	namespaceHandler        *handler.NamespaceHandler
	nodeHandler             *handler.NodeHandler
	workloadHandler         *handler.WorkloadHandler
	configHandler           *handler.ConfigHandler
	storageHandler          *handler.StorageHandler
	serviceResourceHandler  *handler.ServiceResourceHandler
	ingressHandler          *handler.IngressHandler
	networkPolicyHandler    *handler.NetworkPolicyHandler
	k8sDiscoveryHandler     *handler.K8sDiscoveryHandler
	k8sHPAHandler           *handler.K8sHPAHandler
	helmHandler             *handler.HelmHandler
	k8sResourceWatchHandler *handler.K8sResourceWatchHandler
	k8sSearchHandler        *handler.K8sSearchHandler
	k8sEventForwardHandler  *handler.K8sEventForwardHandler
	eventHandler            *handler.EventHandler
	crdHandler              *handler.CRDHandler
	crHandler               *handler.CRHandler
	rbacHandler             *handler.RBACHandler
	serviceAccountHandler   *handler.ServiceAccountHandler
	overviewHandler         *handler.OverviewHandler

	projectHandler     *handler.ProjectHandler
	projectCatalogHandler *handler.ProjectCatalogHandler
	cmdbHandler        *handler.CMDBHandler
	mysqlBackupSvc     *service.MysqlBackupService
	mysqlBackupHandler *handler.MysqlBackupHandler
	dbmgmtSvc          *dbmgmtsvc.Service
	dbmgmtHandler      *handler.DbmgmtHandler
	alertSvc           *service.AlertService
	cicdSvc            *cicdsvc.Service
	cicdHandler        *handler.CicdHandler
	logPlatformHandler *handler.LogPlatformHandler
	loggieHandler      *handler.LoggieHandler
	clusterLogHandler  *handler.ClusterLogHandler
	logRetentionSvc    *service.LogRetentionService
	kafkaToESSvc       *service.KafkaToESService
	inspectSvc         *inspectsvc.Service
	inspectHandler     *handler.InspectHandler
	aiHandler          *handler.AIHandler
	esmgmtSvc          *esmgmtsvc.Service
	esmgmtHandler      *handler.EsmgmtHandler
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

// EsmgmtService 供 esmgmt 插件调度器使用。
func (d *RouteDeps) EsmgmtService() *esmgmtsvc.Service {
	if d == nil {
		return nil
	}
	return d.esmgmtSvc
}

// DbmgmtService 供 dbmgmt 插件后台任务使用。
func (d *RouteDeps) DbmgmtService() *dbmgmtsvc.Service {
	if d == nil {
		return nil
	}
	return d.dbmgmtSvc
}

// CicdService 供 cicd 插件后台任务使用。
func (d *RouteDeps) CicdService() *cicdsvc.Service {
	if d == nil {
		return nil
	}
	return d.cicdSvc
}

// AlertService 供 alert 插件后台任务使用。
func (d *RouteDeps) AlertService() *service.AlertService {
	if d == nil {
		return nil
	}
	return d.alertSvc
}

// LogRetentionService 供 project 插件保留策略调度使用。
func (d *RouteDeps) LogRetentionService() *service.LogRetentionService {
	if d == nil {
		return nil
	}
	return d.logRetentionSvc
}

// KafkaToESService 供 project 插件 Kafka→ES 消费使用。
func (d *RouteDeps) KafkaToESService() *service.KafkaToESService {
	if d == nil {
		return nil
	}
	return d.kafkaToESSvc
}

// InspectService 供 inspect 插件调度器使用。
func (d *RouteDeps) InspectService() *inspectsvc.Service {
	if d == nil {
		return nil
	}
	return d.inspectSvc
}

// assembleRouteDeps 将 Wire 已注入的 repos/services/handlers 与中间件拼成 RouteDeps。
func assembleRouteDeps(
	app *bootstrap.App,
	repos *routeRepositories,
	svcs *routeServices,
	handlers *routeHandlers,
) (*RouteDeps, error) {
	if svcs == nil {
		return nil, fmt.Errorf("route services required (use InitializeRouteDeps)")
	}
	if handlers == nil {
		return nil, fmt.Errorf("route handlers required (use InitializeRouteDeps)")
	}
	if repos == nil {
		repos = newRouteRepositories(app.DB)
	}

	authMiddleware := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, repos.User, app.Logger)
	wsAuthMiddleware := middleware.WSAuth(app.Redis, repos.User, app.Logger)
	authorize := middleware.Authorize(app.Enforcer, app.Logger, repos.K8sClusterAccess)
	k8sScopeAuthorize := middleware.K8sScopeAuthorize(
		app.Logger,
		repos.Permission,
		repos.K8sClusterAccess,
		repos.K8sNsDeny,
		repos.K8sNsAllow,
	)
	opAudit := middleware.OperationAudit(svcs.OperationLog, app.Logger)

	deps := &RouteDeps{
		app: app,

		authMiddleware:    authMiddleware,
		wsAuthMiddleware:  wsAuthMiddleware,
		authorize:         authorize,
		k8sScopeAuthorize: k8sScopeAuthorize,
		opAudit:           opAudit,

		projectMemberRepo: repos.ProjectMember,
		clusterRepo:       repos.Cluster,
		k8sRuntimeService: svcs.K8sRuntime,

		systemHandler: handlers.System,
		pluginHandler: handlers.Plugin,

		authHandler:              handlers.Auth,
		loginLogHandler:          handlers.LoginLog,
		opLogHandler:             handlers.OperationLog,
		userHandler:              handlers.User,
		departmentHandler:        handlers.Department,
		roleHandler:              handlers.Role,
		permissionHandler:        handlers.Permission,
		policyHandler:            handlers.Policy,
		k8sScopedPolicyHandler:   handlers.K8sScopedPolicy,
		k8sNamespaceDenyHandler:  handlers.K8sNamespaceDeny,
		k8sNamespaceAllowHandler: handlers.K8sNamespaceAllow,
		userGroupHandler:         handlers.UserGroup,
		regHandler:               handlers.Registration,
		menuHandler:              handlers.Menu,
		dictEntryHandler:         handlers.DictEntry,
		adminHandler:             handlers.Admin,

		alertHandler:              handlers.Alert,
		alertPlatformHandler:      handlers.AlertPlatform,
		alertSubscriptionHandler:  handlers.AlertSubscription,
		alertInhibitionHandler:    handlers.AlertInhibition,
		alertReceiverGroupHandler: handlers.AlertReceiverGroup,
		cloudExpiryRuleHandler:    handlers.CloudExpiryRule,

		clusterHandler:          handlers.Cluster,
		podHandler:              handlers.Pod,
		namespaceHandler:        handlers.Namespace,
		nodeHandler:             handlers.Node,
		workloadHandler:         handlers.Workload,
		configHandler:           handlers.Config,
		storageHandler:          handlers.Storage,
		serviceResourceHandler:  handlers.ServiceResource,
		ingressHandler:          handlers.Ingress,
		networkPolicyHandler:    handlers.NetworkPolicy,
		k8sDiscoveryHandler:     handlers.K8sDiscovery,
		k8sHPAHandler:           handlers.K8sHPA,
		helmHandler:             handlers.Helm,
		k8sResourceWatchHandler: handlers.K8sResourceWatch,
		k8sSearchHandler:        handlers.K8sSearch,
		k8sEventForwardHandler:  handlers.K8sEventForward,
		eventHandler:            handlers.Event,
		crdHandler:              handlers.CRD,
		crHandler:               handlers.CR,
		rbacHandler:             handlers.RBAC,
		serviceAccountHandler:   handlers.ServiceAccount,
		overviewHandler:         handlers.Overview,

		projectHandler:        handlers.Project,
		projectCatalogHandler: handlers.ProjectCatalog,
		cmdbHandler:           handlers.CMDB,
		mysqlBackupSvc:     svcs.MysqlBackup,
		mysqlBackupHandler: handlers.MysqlBackup,
		dbmgmtSvc:          svcs.Dbmgmt,
		dbmgmtHandler:      handlers.Dbmgmt,
		alertSvc:           svcs.Alert,
		cicdSvc:            svcs.Cicd,
		cicdHandler:        handlers.Cicd,
		logPlatformHandler: handlers.LogPlatform,
		loggieHandler:      handlers.Loggie,
		clusterLogHandler:  handlers.ClusterLog,
		logRetentionSvc:    svcs.LogRetention,
		kafkaToESSvc:       svcs.KafkaToES,
		inspectSvc:         svcs.Inspect,
		inspectHandler:     handlers.Inspect,
		aiHandler:          handlers.AI,
		esmgmtSvc:          svcs.Esmgmt,
		esmgmtHandler:      handlers.Esmgmt,
	}
	wireCicdK8sHooks(deps.cicdSvc, svcs.K8sWorkload)
	return deps, nil
}

func wireCicdK8sHooks(cicdSvc *cicdsvc.Service, wl *service.K8sWorkloadService) {
	if cicdSvc == nil || wl == nil {
		return
	}
	cicdSvc.SetWorkloadReadyCheck(func(ctx context.Context, clusterID, namespace, kind, name string) (*bool, string) {
		return wl.WorkloadReady(ctx, clusterID, namespace, kind, name)
	})
	cicdSvc.SetK8sRolloutUndo(func(ctx context.Context, kind string, clusterID uint, namespace, name string, revision int64) (map[string]any, error) {
		req := service.RolloutUndoRequest{
			ClusterID: clusterID,
			Namespace: namespace,
			Name:      name,
			Revision:  revision,
		}
		var (
			res *service.RolloutUndoResult
			err error
		)
		switch strings.ToLower(strings.TrimSpace(kind)) {
		case "statefulset", "sts", "statefulsets":
			res, err = wl.StatefulSetRolloutUndo(ctx, req)
		default:
			res, err = wl.DeploymentRolloutUndo(ctx, req)
		}
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"kind":          res.Kind,
			"namespace":     res.Namespace,
			"name":          res.Name,
			"from_revision": res.FromRevision,
			"to_revision":   res.ToRevision,
			"message":       res.Message,
		}, nil
	})
}
