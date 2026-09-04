package router

import (
	"context"
	"strings"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"log/slog"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/objectstore"
	"yunshu/internal/service"
	aisvc "yunshu/internal/service/ai"
	"yunshu/internal/service/alert"
	cicdsvc "yunshu/internal/service/cicd"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
	esmgmtsvc "yunshu/internal/service/esmgmt"
	inspectsvc "yunshu/internal/service/inspect"

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
	"ServiceCatalog", "ChangeEvent",
)

// AppInfraSet extracts infrastructure dependencies from bootstrap.App.
var AppInfraSet = wire.NewSet(
	provideDB,
	provideRedis,
	provideEnforcer,
	provideMailer,
	provideAuthConfig,
	provideAlertConfig,
	provideCicdConfig,
	provideDbmgmtConfig,
	provideAIConfig,
	provideAppRouteConfig,
	appRouteConfigFields,
	providePluginsConfig,
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
	firingDeliveryRepo interfaces.AlertFiringDeliveryRepository,
	cfg config.AlertConfig,
) alert.AlertStateService {
	return alert.NewRedisAlertStateService(
		redisClient,
		eventRepo,
		firingDeliveryRepo,
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

// attachAlertLogSearch 在 LogSearch 就绪后注入证据包依赖（避免 wire 循环顺序问题）。
func attachAlertLogSearch(svc *service.AlertService, logSearch *service.LogSearchService) *service.AlertService {
	if svc != nil {
		svc.SetLogSearch(logSearch)
	}
	return svc
}

func provideAuthService(
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	db *gorm.DB,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.AuthService {
	return service.NewAuthService(userRepo, redisClient, db, authCfg, sender, string(appName))
}

func provideRegistrationService(
	regRepo interfaces.RegistrationRequestRepository,
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	db *gorm.DB,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.RegistrationService {
	return service.NewRegistrationService(regRepo, userRepo, redisClient, db, authCfg, sender, string(appName))
}

func provideK8sRuntimeService(
	repo interfaces.K8sClusterRepository,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
	memberRepo interfaces.ProjectMemberRepository,
	encryptionKey SecurityEncryptionKey,
) (*service.K8sRuntimeService, error) {
	return service.NewK8sRuntimeService(repo, nsDeny, nsAllow, memberRepo, string(encryptionKey))
}

func provideCMDBService(
	db *gorm.DB,
	serverRepo interfaces.ServerRepository,
	serverGroupRepo interfaces.ServerGroupRepository,
	cloudAccountRepo interfaces.CloudAccountRepository,
	memberRepo interfaces.ProjectMemberRepository,
	encryptionKey SecurityEncryptionKey,
) (*service.CMDBService, error) {
	return service.NewCMDBService(db, serverRepo, serverGroupRepo, cloudAccountRepo, memberRepo, string(encryptionKey))
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
	newStore := func(ctx context.Context) (*objectstore.Client, error) {
		return objectstore.NewFromDB(ctx, db)
	}
	resolveSched := func(ctx context.Context) dictconfig.MysqlBackupSchedulerConfig {
		return dictconfig.ResolveMysqlBackupSchedulerConfig(ctx, db, dictconfig.DefaultMysqlBackupSchedulerDictTypes())
	}
	return service.NewMysqlBackupService(backupRepo, serverRepo, projectRepo, userRepo, newStore, resolveSched, string(encryptionKey), sender, string(appName))
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
	memberRepo interfaces.ProjectMemberRepository,
	cicdCfg config.CicdConfig,
	sender mailer.Sender,
	appName AppDisplayName,
	k8sNS *service.K8sNamespaceService,
) *cicdsvc.Service {
	return cicdsvc.NewService(db, serverRepo, projectRepo, userGroupRepo, userRepo, memberRepo, cicdCfg, sender, string(appName), k8sNS)
}

func provideInspectService(
	db *gorm.DB,
	redisClient *redis.Client,
	dsSvc *service.AlertDatasourceService,
	projectRepo interfaces.ProjectRepository,
	sender mailer.Sender,
	appName AppDisplayName,
) *inspectsvc.Service {
	return inspectsvc.NewService(db, redisClient, dsSvc, projectRepo, sender, string(appName))
}

func provideEsmgmtService(
	db *gorm.DB,
	encryptionKey SecurityEncryptionKey,
	es *service.ElasticsearchProvider,
) (*esmgmtsvc.Service, error) {
	newStore := func(ctx context.Context) (*objectstore.Client, error) {
		return objectstore.NewFromDB(ctx, db)
	}
	resolveSched := func(ctx context.Context) dictconfig.EsmgmtBackupSchedulerConfig {
		return dictconfig.ResolveEsmgmtBackupSchedulerConfig(ctx, db, dictconfig.DefaultEsmgmtBackupSchedulerDictTypes())
	}
	svc, err := esmgmtsvc.NewService(db, string(encryptionKey), es, newStore, resolveSched)
	if err != nil {
		return nil, err
	}
	// 日志平台按 connection_id 使用 esmgmt 连接（避免 Wire 循环依赖，装配后注入）
	if es != nil {
		es.SetManagedConnectionLoader(svc)
	}
	return svc, nil
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

func provideKafkaProvider(app *bootstrap.App) *service.KafkaProvider {
	base := config.KafkaConfig{}
	if app != nil && app.Config != nil {
		base = app.Config.Kafka
	}
	return service.NewKafkaProvider(app.DB, base)
}

func provideKafkaToESService(kafka *service.KafkaProvider, es *service.ElasticsearchProvider) *service.KafkaToESService {
	return service.NewKafkaToESService(kafka, es)
}

func provideLogSearchService(db *gorm.DB, es *service.ElasticsearchProvider, serverRepo interfaces.ServerRepository) *service.LogSearchService {
	return service.NewLogSearchService(es, serverRepo, db)
}

func provideLogIntelligenceService(db *gorm.DB, logSearch *service.LogSearchService, projectRepo interfaces.ProjectRepository) *service.LogIntelligenceService {
	return service.NewLogIntelligenceService(db, logSearch, projectRepo)
}

func provideLogRetentionService(es *service.ElasticsearchProvider, repo interfaces.LogRetentionRepository) *service.LogRetentionService {
	return service.NewLogRetentionService(es, repo)
}

func provideLoggieConfig(app *bootstrap.App) config.LoggieConfig {
	if app == nil || app.Config == nil {
		return config.LoggieConfig{}.Normalized()
	}
	return app.Config.Loggie.Normalized()
}

func provideLoggieAgentService(
	repo interfaces.LoggieAgentRepository,
	serverRepo interfaces.ServerRepository,
	logSourceRepo interfaces.LogSourceRepository,
	projectRepo interfaces.ProjectRepository,
	serviceRepo interfaces.ServiceRepository,
	es *service.ElasticsearchProvider,
	kafka *service.KafkaProvider,
	encryptionKey SecurityEncryptionKey,
	loggieCfg config.LoggieConfig,
) (*service.LoggieAgentService, error) {
	return service.NewLoggieAgentService(repo, serverRepo, logSourceRepo, projectRepo, serviceRepo, es, kafka, string(encryptionKey), loggieCfg)
}

func provideClusterLogService(
	db *gorm.DB,
	projectRepo interfaces.ProjectRepository,
	es *service.ElasticsearchProvider,
	kafka *service.KafkaProvider,
	runtime *service.K8sRuntimeService,
	loggieCfg config.LoggieConfig,
) *service.ClusterLogService {
	return service.NewClusterLogService(db, projectRepo, es, kafka, runtime, loggieCfg, loggieCfg.DaemonSetImage)
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
	service.NewPolicyGovernanceService,
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
	service.NewAlertConsulService,
	service.NewAlertMonitorRuleService,
	service.NewAlertReceiverGroupService,
	// k8s
	provideK8sRuntimeService,
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
	service.NewServiceCatalogService,
	service.NewChangeEventService,
	provideMysqlBackupService,
	provideDbmgmtService,
	provideCicdService,
	provideInspectService,
	aisvc.NewService,
	provideEsmgmtService,
	provideElasticsearchProvider,
	provideKafkaProvider,
	provideKafkaToESService,
	provideLogSearchService,
	provideLogIntelligenceService,
	provideLogRetentionService,
	provideLoggieConfig,
	provideLoggieAgentService,
	provideClusterLogService,
	wire.Struct(new(routeServices), "*"),
)
