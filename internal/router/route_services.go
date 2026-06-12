package router

import (
	"fmt"
	"strings"

	"yunshu/internal/bootstrap"
	"yunshu/internal/pkg/logutil"
	"yunshu/internal/service"
	"yunshu/internal/service/alert"
)

// routeServices aggregates HTTP-layer domain services built from routeRepositories.
type routeServices struct {
	LoginLog            *service.LoginLogService
	OperationLog        *service.OperationLogService
	Auth                *service.AuthService
	User                *service.UserService
	Department          *service.DepartmentService
	Role                *service.RoleService
	Permission          *service.PermissionService
	Policy              *service.PolicyService
	K8sScopedPolicy     *service.K8sScopedPolicyService
	K8sNamespaceDeny    *service.K8sNamespaceDenyService
	K8sNamespaceAllow   *service.K8sNamespaceAllowService
	UserGroup           *service.UserGroupService
	Registration        *service.RegistrationService
	Menu                *service.MenuService
	DictEntry           *service.DictEntryService
	AlertSilence        *service.AlertSilenceService
	AlertDuty           *service.AlertDutyService
	AlertAssignee       *service.AlertRuleAssigneeService
	AlertReceiverCache  *service.ReceiverGroupCache
	Alert               *service.AlertService
	CloudExpiryRule     *service.CloudExpiryRuleService
	AlertDatasource     *service.AlertDatasourceService
	AlertMonitorRule    *service.AlertMonitorRuleService
	K8sRuntime          *service.K8sRuntimeService
	K8sCluster          *service.K8sClusterService
	K8sPod              *service.K8sPodService
	K8sNamespace        *service.K8sNamespaceService
	K8sNode             *service.K8sNodeService
	K8sWorkload         *service.K8sWorkloadService
	K8sConfig           *service.K8sConfigService
	K8sStorage          *service.K8sStorageService
	K8sServiceResource  *service.K8sServiceResourceService
	K8sIngress          *service.K8sIngressService
	K8sNetworkPolicy    *service.K8sNetworkPolicyService
	K8sDiscovery        *service.K8sDiscoveryService
	K8sHPA              *service.K8sHPAService
	K8sEvent            *service.K8sEventService
	K8sCRD              *service.K8sCRDService
	K8sCR               *service.K8sCRService
	K8sRBAC             *service.K8sRBACService
	K8sServiceAccount   *service.K8sServiceAccountService
	Overview            *service.OverviewService
	ProjectMgmt         *service.ProjectMgmtService
	CMDB                *service.CMDBService
	MysqlBackup         *service.MysqlBackupService
	LogAgent            *service.LogAgentService
	AgentDiscovery      *service.AgentDiscoveryService
	AlertReceiverGroup  *service.AlertReceiverGroupService
	K8sEventForwardAdmin *service.K8sEventForwardAdminService
	K8sSearch            *service.K8sSearchService
	AlertMaintenance     *service.AlertMaintenanceService
}

func buildRouteServices(app *bootstrap.App, repos *routeRepositories) (*routeServices, error) {
	if repos == nil {
		return nil, fmt.Errorf("route repositories required")
	}

	userRepo := repos.User
	departmentRepo := repos.Department
	roleRepo := repos.Role
	permissionRepo := repos.Permission
	projectMemberRepo := repos.ProjectMember
	k8sNsDenyRepo := repos.K8sNsDeny
	k8sNsAllowRepo := repos.K8sNsAllow
	userGroupRepo := repos.UserGroup
	k8sClusterAccessRepo := repos.K8sClusterAccess
	clusterRepo := repos.Cluster
	projectRepo := repos.Project
	dictEntryRepo := repos.DictEntry

	loginLogSvc := service.NewLoginLogService(repos.LoginLog)
	opLogSvc := service.NewOperationLogService(repos.OperationLog)
	authService := service.NewAuthService(userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	alertAssigneeSvc := service.NewAlertRuleAssigneeService(repos.AlertRuleAssignee, repos.AlertMonitorRule, repos.AlertDatasource, userRepo, projectMemberRepo, departmentRepo)
	userService := service.NewUserService(userRepo, roleRepo, departmentRepo, app.Enforcer, projectMemberRepo, alertAssigneeSvc)
	departmentService := service.NewDepartmentService(departmentRepo, userRepo, alertAssigneeSvc)
	roleService := service.NewRoleService(roleRepo, app.Enforcer)
	permissionService := service.NewPermissionService(permissionRepo, app.Enforcer)
	policyService := service.NewPolicyService(roleRepo, permissionRepo, app.Enforcer)
	k8sScopedPolicyService := service.NewK8sScopedPolicyService(roleRepo, permissionRepo, k8sClusterAccessRepo, k8sNsDenyRepo, k8sNsAllowRepo, userGroupRepo, userRepo, clusterRepo)
	k8sNamespaceDenySvc := service.NewK8sNamespaceDenyService(k8sNsDenyRepo)
	k8sNamespaceAllowSvc := service.NewK8sNamespaceAllowService(k8sNsAllowRepo)
	userGroupSvc := service.NewUserGroupService(userGroupRepo, userRepo, projectMemberRepo, projectRepo)
	registrationService := service.NewRegistrationService(repos.RegRequest, userRepo, app.Redis, app.Config.Auth, app.Mailer, app.Config.App.Name)
	menuService := service.NewMenuService(repos.Menu)
	dictEntryService := service.NewDictEntryService(repos.DictEntry)

	alertSilenceSvc := service.NewAlertSilenceService(repos.AlertSilence)
	alertMaintenanceSvc := service.NewAlertMaintenanceService(repos.AlertMaintenance)
	alertDutySvc := service.NewAlertDutyService(repos.AlertDuty, repos.AlertMonitorRule, userRepo)
	alertReceiverGroupCache := service.NewReceiverGroupCache(repos.AlertReceiverGroup)

	alertStateSvc := alert.NewRedisAlertStateService(app.Redis, repos.AlertEvent, app.Config.Alert.DedupTTLSeconds, app.Config.Alert.AggregateTTLSeconds)

	alertService := service.NewAlertService(app.DB, app.Redis, app.Mailer, app.Config.Alert, &service.AlertServiceOptions{
		SilenceSvc:         alertSilenceSvc,
		MaintenanceSvc:     alertMaintenanceSvc,
		AssigneeSvc:        alertAssigneeSvc,
		DutySvc:            alertDutySvc,
		ReceiverGroupCache: alertReceiverGroupCache,
		EncryptionKey:      app.Config.Security.EncryptionKey,
		EventRepo:          repos.AlertEvent,
		ChannelRepo:        repos.AlertChannel,
		MonitorRuleRepo:    repos.AlertMonitorRule,
		DatasourceRepo:     repos.AlertDatasource,
		ProjectRepo:        repos.Project,
		FiringDeliveryRepo: repos.AlertFiringDelivery,
		CloudExpiryRepo:    repos.CloudExpiryRule,
		CloudAccountRepo:   repos.CloudAccount,
		StateSvc:           alertStateSvc,
		SubscriptionRepo:   repos.AlertSubscription,
		InhibitionRuleRepo: repos.AlertInhibitionRule,
	})
	if strings.TrimSpace(app.Config.Alert.WebhookToken) == "" {
		logutil.Worker("router").Warnw("Alert webhook token is empty; Alertmanager webhooks will be rejected until configured")
	}

	cloudExpiryRuleSvc := service.NewCloudExpiryRuleService(repos.CloudExpiryRule)
	alertDatasourceSvc := service.NewAlertDatasourceService(repos.AlertDatasource)
	alertMonitorRuleSvc := service.NewAlertMonitorRuleService(repos.AlertMonitorRule, repos.AlertDatasource, app.Redis)

	k8sRuntimeService := service.NewK8sRuntimeService(clusterRepo)
	clusterService := service.NewK8sClusterService(clusterRepo, dictEntryRepo, k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo, projectMemberRepo)
	podService := service.NewK8sPodService(k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo)
	namespaceService := service.NewK8sNamespaceService(k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo)
	nodeService := service.NewK8sNodeService(k8sRuntimeService)
	workloadService := service.NewK8sWorkloadService(k8sRuntimeService)
	configService := service.NewK8sConfigService(k8sRuntimeService)
	storageService := service.NewK8sStorageService(k8sRuntimeService)
	serviceResourceService := service.NewK8sServiceResourceService(k8sRuntimeService)
	ingressService := service.NewK8sIngressService(k8sRuntimeService, k8sClusterAccessRepo)
	networkPolicyService := service.NewK8sNetworkPolicyService(k8sRuntimeService)
	k8sDiscoveryService := service.NewK8sDiscoveryService(k8sRuntimeService)
	k8sHPAService := service.NewK8sHPAService(k8sRuntimeService)
	eventService := service.NewK8sEventService(k8sRuntimeService, k8sNsDenyRepo, k8sNsAllowRepo)
	crdService := service.NewK8sCRDService(k8sRuntimeService)
	crService := service.NewK8sCRService(k8sRuntimeService)
	rbacService := service.NewK8sRBACService(k8sRuntimeService)
	serviceAccountService := service.NewK8sServiceAccountService(k8sRuntimeService)
	overviewService := service.NewOverviewService(repos.Overview, k8sRuntimeService, app.Redis, projectMemberRepo, k8sClusterAccessRepo)

	cmdbService, err := service.NewCMDBService(
		repos.Server, repos.ServerGroup, repos.CloudAccount, app.Config.Security.EncryptionKey,
	)
	if err != nil {
		return nil, fmt.Errorf("cmdb service: %w", err)
	}
	projectMgmtService := service.NewProjectMgmtService(
		projectRepo, repos.Server, repos.ServerGroup, repos.Service, repos.LogSource, projectMemberRepo, userRepo, departmentRepo,
	)
	mysqlBackupSvc, err := service.NewMysqlBackupService(repos.MysqlBackup, repos.Server, projectRepo, app.DB, app.Config.Security.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("mysql backup service: %w", err)
	}
	logAgentService := service.NewLogAgentService(repos.LogAgent, repos.Server, repos.LogSource, app.Config.Agent.RegisterSecret, app.Config.Agent.DiscoveryRoots)
	agentDiscoveryService := service.NewAgentDiscoveryService(repos.AgentDiscovery, repos.LogAgent, repos.Server, repos.LogSource)
	alertReceiverGroupSvc := service.NewAlertReceiverGroupService(repos.AlertReceiverGroup, alertReceiverGroupCache)
	k8sEventForwardAdminSvc := service.NewK8sEventForwardAdminService(repos.K8sEventForward)
	k8sSearchSvc := service.NewK8sSearchService(k8sRuntimeService, clusterRepo, projectMemberRepo, k8sClusterAccessRepo, k8sNsDenyRepo, k8sNsAllowRepo)

	return &routeServices{
		LoginLog:             loginLogSvc,
		OperationLog:         opLogSvc,
		Auth:                 authService,
		User:                 userService,
		Department:           departmentService,
		Role:                 roleService,
		Permission:           permissionService,
		Policy:               policyService,
		K8sScopedPolicy:      k8sScopedPolicyService,
		K8sNamespaceDeny:     k8sNamespaceDenySvc,
		K8sNamespaceAllow:   k8sNamespaceAllowSvc,
		UserGroup:            userGroupSvc,
		Registration:         registrationService,
		Menu:                 menuService,
		DictEntry:            dictEntryService,
		AlertSilence:         alertSilenceSvc,
		AlertDuty:            alertDutySvc,
		AlertAssignee:        alertAssigneeSvc,
		AlertReceiverCache:   alertReceiverGroupCache,
		Alert:                alertService,
		CloudExpiryRule:      cloudExpiryRuleSvc,
		AlertDatasource:      alertDatasourceSvc,
		AlertMonitorRule:     alertMonitorRuleSvc,
		K8sRuntime:           k8sRuntimeService,
		K8sCluster:           clusterService,
		K8sPod:               podService,
		K8sNamespace:         namespaceService,
		K8sNode:              nodeService,
		K8sWorkload:          workloadService,
		K8sConfig:            configService,
		K8sStorage:           storageService,
		K8sServiceResource:   serviceResourceService,
		K8sIngress:           ingressService,
		K8sNetworkPolicy:     networkPolicyService,
		K8sDiscovery:         k8sDiscoveryService,
		K8sHPA:               k8sHPAService,
		K8sEvent:             eventService,
		K8sCRD:               crdService,
		K8sCR:                crService,
		K8sRBAC:              rbacService,
		K8sServiceAccount:    serviceAccountService,
		Overview:             overviewService,
		ProjectMgmt:          projectMgmtService,
		CMDB:                 cmdbService,
		MysqlBackup:          mysqlBackupSvc,
		LogAgent:             logAgentService,
		AgentDiscovery:       agentDiscoveryService,
		AlertReceiverGroup:   alertReceiverGroupSvc,
		K8sEventForwardAdmin: k8sEventForwardAdminSvc,
		K8sSearch:            k8sSearchSvc,
		AlertMaintenance:     alertMaintenanceSvc,
	}, nil
}

func provideRouteServices(app *bootstrap.App, repos *routeRepositories) (*routeServices, error) {
	return buildRouteServices(app, repos)
}
