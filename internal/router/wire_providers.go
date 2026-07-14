package router

import (
	"strings"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"log/slog"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/service"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// repositoryFieldNames ??routeRepositories ???????? wire.FieldsOf ?????????
var repositoryFieldNames = wire.FieldsOf(
	new(*routeRepositories),
	"User", "Department", "Role", "Permission", "LoginLog", "OperationLog",
	"ProjectMember", "K8sNsDeny", "K8sNsAllow", "UserGroup", "K8sClusterAccess",
	"Cluster", "Project", "RegRequest", "Menu", "DictEntry",
	"Server", "ServerGroup", "CloudAccount", "Service", "LogSource",
	"LogRetention", "LoggieAgent", "MysqlBackup", "Dbmgmt",
	"AlertEvent", "AlertChannel", "AlertSilence", "AlertMaintenance",
	"AlertInhibitionRule", "AlertSubscription", "AlertDatasource",
	"AlertMonitorRule", "AlertReceiverGroup", "AlertDuty", "AlertRuleAssignee",
	"AlertFiringDelivery", "CloudExpiryRule", "Overview", "K8sEventForward",
)

// AppInfraSet ??bootstrap.App ??????????????
var AppInfraSet = wire.NewSet(
	provideDB,
	provideRedis,
	provideEnforcer,
	provideMailer,
	provideAuthConfig,
	provideAlertConfig,
	provideCicdConfig,
	provideDbmgmtConfig,
	provideAppRouteConfig,
	appRouteConfigFields,
	providePluginsEnabled,
)

// RepositorySet ???????????
var RepositorySet = wire.NewSet(
	newRouteRepositories,
	repositoryFieldNames,
)

func provideAlertStateService(
	redisClient *redis.Client,
	eventRepo interfaces.AlertEventRepository,
	cfg config.AlertConfig,
) alert.AlertStateService {
	return alert.NewRedisAlertStateService(
		redisClient,
		eventRepo,
		cfg.DedupTTLSeconds,
		cfg.AggregateTTLSeconds,
	)
}

func provideAlertServiceOptions(
	silence *service.AlertSilenceService,
	maintenance *service.AlertMaintenanceService,
	assignee *service.AlertRuleAssigneeService,
	duty *service.AlertDutyService,
	cache *service.ReceiverGroupCache,
	encryptionKey SecurityEncryptionKey,
	eventRepo interfaces.AlertEventRepository,
	channelRepo interfaces.AlertChannelRepository,
	monitorRuleRepo interfaces.AlertMonitorRuleRepository,
	datasourceRepo interfaces.AlertDatasourceRepository,
	projectRepo interfaces.ProjectRepository,
	firingDeliveryRepo interfaces.AlertFiringDeliveryRepository,
	cloudExpiryRepo interfaces.CloudExpiryRuleRepository,
	cloudAccountRepo interfaces.CloudAccountRepository,
	stateSvc alert.AlertStateService,
	subscriptionRepo interfaces.AlertSubscriptionRepository,
	inhibitionRuleRepo interfaces.AlertInhibitionRuleRepository,
) *service.AlertServiceOptions {
	return &service.AlertServiceOptions{
		SilenceSvc:         silence,
		MaintenanceSvc:     maintenance,
		AssigneeSvc:        assignee,
		DutySvc:            duty,
		ReceiverGroupCache: cache,
		EncryptionKey:      string(encryptionKey),
		EventRepo:          eventRepo,
		ChannelRepo:        channelRepo,
		MonitorRuleRepo:    monitorRuleRepo,
		DatasourceRepo:     datasourceRepo,
		ProjectRepo:        projectRepo,
		FiringDeliveryRepo: firingDeliveryRepo,
		CloudExpiryRepo:    cloudExpiryRepo,
		CloudAccountRepo:   cloudAccountRepo,
		StateSvc:           stateSvc,
		SubscriptionRepo:   subscriptionRepo,
		InhibitionRuleRepo: inhibitionRuleRepo,
	}
}

func provideAlertService(
	db *gorm.DB,
	redisClient *redis.Client,
	sender mailer.Sender,
	cfg config.AlertConfig,
	opts *service.AlertServiceOptions,
) *service.AlertService {
	if strings.TrimSpace(cfg.WebhookToken) == "" {
		slog.Default().With("component", "router").Warn(
			"Alert webhook token is empty; Alertmanager webhooks will be rejected until configured",
		)
	}
	return service.NewAlertService(db, redisClient, sender, cfg, opts)
}

func provideAuthService(
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.AuthService {
	return service.NewAuthService(userRepo, redisClient, authCfg, sender, string(appName))
}

func provideRegistrationService(
	regRepo interfaces.RegistrationRequestRepository,
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.RegistrationService {
	return service.NewRegistrationService(regRepo, userRepo, redisClient, authCfg, sender, string(appName))
}

func provideCMDBService(
	serverRepo interfaces.ServerRepository,
	serverGroupRepo interfaces.ServerGroupRepository,
	cloudAccountRepo interfaces.CloudAccountRepository,
	encryptionKey SecurityEncryptionKey,
) (*service.CMDBService, error) {
	return service.NewCMDBService(serverRepo, serverGroupRepo, cloudAccountRepo, string(encryptionKey))
}

func provideMysqlBackupService(
	backupRepo interfaces.MysqlBackupRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userRepo interfaces.UserRepository,
	db *gorm.DB,
	encryptionKey SecurityEncryptionKey,
	sender mailer.Sender,
	appName AppDisplayName,
) (*service.MysqlBackupService, error) {
	return service.NewMysqlBackupService(backupRepo, serverRepo, projectRepo, userRepo, db, string(encryptionKey), sender, string(appName))
}

func provideDbmgmtService(
	dbmgmtRepo interfaces.DbmgmtRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	db *gorm.DB,
	encryptionKey SecurityEncryptionKey,
	sender mailer.Sender,
	appName AppDisplayName,
	cfg config.DbmgmtConfig,
) (*dbmgmtsvc.Service, error) {
	return dbmgmtsvc.NewService(dbmgmtRepo, serverRepo, projectRepo, userGroupRepo, userRepo, db, string(encryptionKey), sender, string(appName), cfg)
}

func provideCicdService(
	db *gorm.DB,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	cicdCfg config.CicdConfig,
	sender mailer.Sender,
	appName AppDisplayName,
	k8sNS *service.K8sNamespaceService,
) *cicdsvc.Service {
	return cicdsvc.NewService(db, serverRepo, projectRepo, userGroupRepo, userRepo, cicdCfg, sender, string(appName), k8sNS)
}

func provideK8sHelmService(
	runtime *service.K8sRuntimeService,
	db *gorm.DB,
	cicdCfg config.CicdConfig,
) *service.K8sHelmService {
	return service.NewK8sHelmService(runtime, db, cicdCfg)
}

func provideElasticsearchProvider(app *bootstrap.App) *service.ElasticsearchProvider {
	return service.NewElasticsearchProvider(app.DB, app.Config.Elasticsearch)
}

func provideLogSearchService(es *service.ElasticsearchProvider) *service.LogSearchService {
	return service.NewLogSearchService(es)
}

func provideLogRetentionService(es *service.ElasticsearchProvider, repo interfaces.LogRetentionRepository) *service.LogRetentionService {
	return service.NewLogRetentionService(es, repo)
}

func provideLoggieAgentService(
	repo interfaces.LoggieAgentRepository,
	serverRepo interfaces.ServerRepository,
	logSourceRepo interfaces.LogSourceRepository,
	es *service.ElasticsearchProvider,
	encryptionKey SecurityEncryptionKey,
	k8sRuntime *service.K8sRuntimeService,
	k8sWorkload *service.K8sWorkloadService,
) (*service.LoggieAgentService, error) {
	return service.NewLoggieAgentService(repo, serverRepo, logSourceRepo, es, string(encryptionKey), k8sRuntime, k8sWorkload)
}

var ServiceSet = wire.NewSet(
	// system
	service.NewLoginLogService,
	service.NewOperationLogService,
	provideAuthService,
	service.NewAlertRuleAssigneeService,
	service.NewUserService,
	service.NewDepartmentService,
	service.NewRoleService,
	service.NewPermissionService,
	service.NewPolicyService,
	service.NewK8sScopedPolicyService,
	service.NewK8sNamespaceDenyService,
	service.NewK8sNamespaceAllowService,
	service.NewUserGroupService,
	provideRegistrationService,
	service.NewMenuService,
	service.NewDictEntryService,
	// alert
	service.NewAlertSilenceService,
	service.NewAlertMaintenanceService,
	service.NewAlertDutyService,
	service.NewReceiverGroupCache,
	provideAlertStateService,
	provideAlertServiceOptions,
	provideAlertService,
	service.NewCloudExpiryRuleService,
	service.NewAlertDatasourceService,
	service.NewAlertMonitorRuleService,
	service.NewAlertReceiverGroupService,
	// k8s
	service.NewK8sRuntimeService,
	service.NewK8sClusterService,
	service.NewK8sPodService,
	service.NewK8sNamespaceService,
	service.NewK8sNodeService,
	service.NewK8sWorkloadService,
	service.NewK8sConfigService,
	service.NewK8sStorageService,
	service.NewK8sServiceResourceService,
	service.NewK8sIngressService,
	service.NewK8sNetworkPolicyService,
	service.NewK8sDiscoveryService,
	service.NewK8sHPAService,
	provideK8sHelmService,
	service.NewK8sEventService,
	service.NewK8sCRDService,
	service.NewK8sCRService,
	service.NewK8sRBACService,
	service.NewK8sServiceAccountService,
	service.NewK8sSearchService,
	service.NewK8sEventForwardAdminService,
	// overview / project / cmdb / backup / cicd / log
	service.NewOverviewService,
	provideCMDBService,
	service.NewProjectMgmtService,
	provideMysqlBackupService,
	provideDbmgmtService,
	provideCicdService,
	provideElasticsearchProvider,
	provideLogSearchService,
	provideLogRetentionService,
	provideLoggieAgentService,
	wire.Struct(new(routeServices), "*"),
)
