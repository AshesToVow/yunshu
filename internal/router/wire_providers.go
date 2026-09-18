package router

import (
	"context"
	"path/filepath"
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
	workflowsvc "yunshu/internal/service/workflow"

	"github.com/casbin/casbin/v2"
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
	"LogRetention", "LoggieAgent", "LogSavedQuery", "LogDropRule",
	"LogIntelligence", "ClusterLog", "LogPipeline", "MysqlBackup", "Dbmgmt",
	"AlertEvent", "AlertChannel", "AlertSilence", "AlertMaintenance",
	"AlertInhibitionRule", "AlertSubscription", "AlertDatasource",
	"AlertMonitorRule", "AlertReceiverGroup", "AlertDuty", "AlertRuleAssignee",
	"AlertFiringDelivery", "AlertAck", "AlertProgressNote", "AlertCurHis",
	"AlertConsul", "AlertRuleChange", "PromqlSavedQuery",
	"CloudExpiryRule", "Overview", "K8sEventForward",
	"K8sCrTemplate", "K8sWorkloadSnapshot", "HarborMerge",
	"ServiceCatalog", "ServicePortrait", "ChangeEvent", "ServerAccessGrant",
	"PlatformTemplate", "Workflow", "Esmgmt", "Inspect", "Cicd", "Ai",
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
	ackRepo interfaces.AlertAckRepository,
	progressNoteRepo interfaces.AlertProgressNoteRepository,
	curHisRepo interfaces.AlertCurHisRepository,
	promqlSavedQueryRepo interfaces.PromqlSavedQueryRepository,
	changeEventRepo interfaces.ChangeEventRepository,
	dictEntryRepo interfaces.DictEntryRepository,
	receiverGroupRepo interfaces.AlertReceiverGroupRepository,
	memberRepo interfaces.ProjectMemberRepository,
) *service.AlertServiceOptions {
	return &service.AlertServiceOptions{
		SilenceSvc:           silence,
		MaintenanceSvc:       maintenance,
		AssigneeSvc:          assignee,
		DutySvc:              duty,
		ReceiverGroupCache:   cache,
		EncryptionKey:        string(encryptionKey),
		EventRepo:            eventRepo,
		ChannelRepo:          channelRepo,
		MonitorRuleRepo:      monitorRuleRepo,
		DatasourceRepo:       datasourceRepo,
		ProjectRepo:          projectRepo,
		FiringDeliveryRepo:   firingDeliveryRepo,
		CloudExpiryRepo:      cloudExpiryRepo,
		CloudAccountRepo:     cloudAccountRepo,
		StateSvc:             stateSvc,
		SubscriptionRepo:     subscriptionRepo,
		InhibitionRuleRepo:   inhibitionRuleRepo,
		AckRepo:              ackRepo,
		ProgressNoteRepo:     progressNoteRepo,
		CurHisRepo:           curHisRepo,
		PromqlSavedQueryRepo: promqlSavedQueryRepo,
		ChangeEventRepo:      changeEventRepo,
		DictEntryRepo:        dictEntryRepo,
		ReceiverGroupRepo:    receiverGroupRepo,
		MemberRepo:           memberRepo,
	}
}

func provideAlertService(
	redisClient *redis.Client,
	sender mailer.Sender,
	cfg config.AlertConfig,
	opts *service.AlertServiceOptions,
	logSearch *service.LogSearchService,
) *service.AlertService {
	if strings.TrimSpace(cfg.WebhookToken) == "" {
		slog.Default().With("component", "router").Warn(
			"Alert webhook token is empty; Alertmanager webhooks will be rejected until configured",
		)
	}
	svc := service.NewAlertService(redisClient, sender, cfg, opts)
	return attachAlertLogSearch(svc, logSearch)
}

// attachAlertLogSearch 在 LogSearch 就绪后注入证据包依赖（避免 wire 循环顺序问题）。
func attachAlertLogSearch(svc *service.AlertService, logSearch *service.LogSearchService) *service.AlertService {
	if svc != nil {
		svc.SetLogSearch(logSearch)
	}
	return svc
}

// attachAIOptionalDeps 在组合根注入 AI 跨域可选依赖（替代 assembleRouteDeps 内 Set*Deps）。
func attachAIOptionalDeps(
	svc *aisvc.Service,
	serverRepo interfaces.ServerRepository,
	cmdbSvc *service.CMDBService,
	dbmgmtSvc *dbmgmtsvc.Service,
	esmgmtSvc *esmgmtsvc.Service,
	projectMgmt *service.ProjectMgmtService,
	loggieAgent *service.LoggieAgentService,
	clusterLog *service.ClusterLogService,
	ds *service.AlertDatasourceService,
	changeSvc *service.ChangeEventService,
	silenceSvc *service.AlertSilenceService,
) *aisvc.Service {
	if svc == nil {
		return nil
	}
	svc.SetPlatformDeps(serverRepo, cmdbSvc, dbmgmtSvc, esmgmtSvc)
	svc.SetLogPlatformDeps(projectMgmt, loggieAgent, clusterLog)
	svc.SetMonitorDeps(ds)
	svc.SetOpsDeps(changeSvc, silenceSvc)
	return svc
}

// attachCicdK8sHooks 注入 CICD 发布就绪检查与 rollout undo（替代 assembleRouteDeps 旁路）。
func attachCicdK8sHooks(cicdSvc *cicdsvc.Service, wl *service.K8sWorkloadService) *cicdsvc.Service {
	wireCicdK8sHooks(cicdSvc, wl)
	return cicdSvc
}

func providePasswordPolicyResolver(db *gorm.DB) service.PasswordPolicyResolver {
	return func(ctx context.Context) dictconfig.PasswordPolicyConfig {
		return dictconfig.ResolvePasswordPolicy(ctx, db)
	}
}

func provideAuthService(
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	resolvePolicy service.PasswordPolicyResolver,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.AuthService {
	return service.NewAuthService(userRepo, redisClient, resolvePolicy, authCfg, sender, string(appName))
}

func provideRegistrationService(
	regRepo interfaces.RegistrationRequestRepository,
	userRepo interfaces.UserRepository,
	redisClient *redis.Client,
	resolvePolicy service.PasswordPolicyResolver,
	authCfg config.AuthConfig,
	sender mailer.Sender,
	appName AppDisplayName,
) *service.RegistrationService {
	return service.NewRegistrationService(regRepo, userRepo, redisClient, resolvePolicy, authCfg, sender, string(appName))
}

func provideUserService(
	userRepo interfaces.UserRepository,
	roleRepo interfaces.RoleRepository,
	departmentRepo interfaces.DepartmentRepository,
	enforcer *casbin.SyncedEnforcer,
	projectMemberRepo interfaces.ProjectMemberRepository,
	assigneeSvc *service.AlertRuleAssigneeService,
	resolvePolicy service.PasswordPolicyResolver,
) *service.UserService {
	return service.NewUserService(userRepo, roleRepo, departmentRepo, enforcer, projectMemberRepo, assigneeSvc, resolvePolicy)
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
	serverRepo interfaces.ServerRepository,
	serverGroupRepo interfaces.ServerGroupRepository,
	cloudAccountRepo interfaces.CloudAccountRepository,
	accessGrantRepo interfaces.ServerAccessGrantRepository,
	memberRepo interfaces.ProjectMemberRepository,
	userRepo interfaces.UserRepository,
	dictRepo interfaces.DictEntryRepository,
	encryptionKey SecurityEncryptionKey,
) (*service.CMDBService, error) {
	return service.NewCMDBService(
		serverRepo, serverGroupRepo, cloudAccountRepo, accessGrantRepo,
		memberRepo, userRepo, dictRepo, string(encryptionKey),
	)
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
	memberRepo interfaces.ProjectMemberRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	dutyRepo interfaces.AlertDutyRepository,
	workflowRepo interfaces.WorkflowRepository,
	db *gorm.DB,
	encryptionKey SecurityEncryptionKey,
	sender mailer.Sender,
	appName AppDisplayName,
	cfg config.DbmgmtConfig,
) (*dbmgmtsvc.Service, error) {
	resolveCfg := func(ctx context.Context) config.DbmgmtConfig {
		return dictconfig.ResolveDbmgmtConfig(ctx, db, cfg)
	}
	wf := workflowsvc.NewService(workflowRepo, userGroupRepo, dutyRepo, userRepo)
	return dbmgmtsvc.NewService(
		dbmgmtRepo, serverRepo, projectRepo, memberRepo, userGroupRepo, userRepo, dutyRepo,
		wf, resolveCfg, string(encryptionKey), sender, string(appName), cfg,
	)
}

func provideCicdService(
	cicdRepo interfaces.CicdRepository,
	workflowRepo interfaces.WorkflowRepository,
	serverRepo interfaces.ServerRepository,
	projectRepo interfaces.ProjectRepository,
	userGroupRepo interfaces.UserGroupRepository,
	userRepo interfaces.UserRepository,
	memberRepo interfaces.ProjectMemberRepository,
	dutyRepo interfaces.AlertDutyRepository,
	catalogRepo interfaces.ServiceCatalogRepository,
	db *gorm.DB,
	cicdCfg config.CicdConfig,
	sender mailer.Sender,
	appName AppDisplayName,
	k8sNS *service.K8sNamespaceService,
	wl *service.K8sWorkloadService,
) *cicdsvc.Service {
	resolveCfg := func(ctx context.Context) config.CicdConfig {
		base := cicdCfg
		if base.RunSyncIntervalSeconds <= 0 {
			base = config.DefaultCicdConfig()
			base.Jenkins = cicdCfg.Jenkins
		}
		return dictconfig.ResolveCicdConfig(ctx, db, base, dictconfig.DefaultCicdDictTypes())
	}
	resolveMinio := func(ctx context.Context) dictconfig.MinioConfig {
		return dictconfig.ResolveCicdMinioConfig(ctx, db)
	}
	svc := cicdsvc.NewService(
		cicdRepo, workflowRepo, serverRepo, projectRepo, userGroupRepo, userRepo, memberRepo, dutyRepo, catalogRepo,
		resolveCfg, resolveMinio, sender, string(appName), k8sNS,
	)
	return attachCicdK8sHooks(svc, wl)
}

// provideAIService 构造 AI 服务并注入跨域可选依赖（Wire 单 provider，避免同类型双绑定）。
func provideAIService(
	aiRepo interfaces.AiRepository,
	workflowRepo interfaces.WorkflowRepository,
	db *gorm.DB,
	yamlAI config.AIConfig,
	encryptionKey SecurityEncryptionKey,
	memberRepo interfaces.ProjectMemberRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
	nsDenyRepo interfaces.K8sNamespaceDenyRepository,
	nsAllowRepo interfaces.K8sNamespaceAllowRepository,
	clusterSvc *service.K8sClusterService,
	podSvc *service.K8sPodService,
	workloadSvc *service.K8sWorkloadService,
	nsSvc *service.K8sNamespaceService,
	eventSvc *service.K8sEventService,
	logSearch *service.LogSearchService,
	esProvider *service.ElasticsearchProvider,
	cicdSvc *cicdsvc.Service,
	alertSvc *service.AlertService,
	serverRepo interfaces.ServerRepository,
	cmdbSvc *service.CMDBService,
	dbmgmtSvc *dbmgmtsvc.Service,
	esmgmtSvc *esmgmtsvc.Service,
	projectMgmt *service.ProjectMgmtService,
	loggieAgent *service.LoggieAgentService,
	clusterLog *service.ClusterLogService,
	ds *service.AlertDatasourceService,
	changeSvc *service.ChangeEventService,
	silenceSvc *service.AlertSilenceService,
) *aisvc.Service {
	resolveConfig := func(ctx context.Context) config.AIConfig {
		return dictconfig.ResolveAIConfig(ctx, db, yamlAI, dictconfig.DefaultAIDictTypes())
	}
	svc := aisvc.NewService(
		aiRepo, workflowRepo, resolveConfig, string(encryptionKey),
		memberRepo, accessRepo, nsDenyRepo, nsAllowRepo,
		clusterSvc, podSvc, workloadSvc, nsSvc, eventSvc,
		logSearch, esProvider, cicdSvc, alertSvc,
	)
	return attachAIOptionalDeps(
		svc, serverRepo, cmdbSvc, dbmgmtSvc, esmgmtSvc,
		projectMgmt, loggieAgent, clusterLog, ds, changeSvc, silenceSvc,
	)
}

func provideInspectService(
	inspectRepo interfaces.InspectRepository,
	platformTplRepo interfaces.PlatformTemplateRepository,
	db *gorm.DB,
	redisClient *redis.Client,
	dsSvc *service.AlertDatasourceService,
	projectRepo interfaces.ProjectRepository,
	sender mailer.Sender,
	appName AppDisplayName,
) *inspectsvc.Service {
	localRoot := filepath.Join("logs", "inspect-reports")
	newStore := func(ctx context.Context) inspectsvc.ReportStore {
		return inspectsvc.ResolveReportStore(ctx, db, localRoot)
	}
	storageInfo := func(ctx context.Context) inspectsvc.ReportStorageInfo {
		return inspectsvc.ResolveReportStorageInfo(ctx, db, localRoot)
	}
	return inspectsvc.NewService(
		inspectRepo, platformTplRepo, redisClient, dsSvc, projectRepo,
		sender, string(appName), newStore, storageInfo,
	)
}

func provideEsmgmtService(
	esmgmtRepo interfaces.EsmgmtRepository,
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
	svc, err := esmgmtsvc.NewService(esmgmtRepo, string(encryptionKey), es, newStore, resolveSched)
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
	harborMerge interfaces.HarborMergeRepository,
	db *gorm.DB,
	cicdCfg config.CicdConfig,
) *service.K8sHelmService {
	resolve := func(ctx context.Context) config.CicdConfig {
		return dictconfig.ResolveCicdConfig(ctx, db, cicdCfg, dictconfig.DefaultCicdDictTypes())
	}
	return service.NewK8sHelmService(runtime, harborMerge, resolve, cicdCfg)
}

func provideK8sPodService(
	runtime *service.K8sRuntimeService,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
	db *gorm.DB,
) *service.K8sPodService {
	resolve := func(ctx context.Context) string {
		return dictconfig.ResolvePodDebugImage(ctx, db)
	}
	return service.NewK8sPodService(runtime, nsDeny, nsAllow, resolve)
}

func provideElasticsearchProvider(app *bootstrap.App) *service.ElasticsearchProvider {
	db := app.DB
	base := app.Config.Elasticsearch
	resolve := func(ctx context.Context) config.ElasticsearchConfig {
		return dictconfig.ResolveElasticsearchConfig(ctx, db, base)
	}
	fetch := func(ctx context.Context, dictType string) (string, bool) {
		return dictconfig.FetchEnabledDictValue(ctx, db, dictType)
	}
	upsert := func(ctx context.Context, dictType, label, value, remark string) error {
		return dictconfig.UpsertEnabledDictValue(ctx, db, dictType, label, value, remark)
	}
	return service.NewElasticsearchProvider(resolve, fetch, upsert, base)
}

func provideKafkaProvider(app *bootstrap.App) *service.KafkaProvider {
	base := config.KafkaConfig{}
	if app != nil && app.Config != nil {
		base = app.Config.Kafka
	}
	db := app.DB
	resolve := func(ctx context.Context) config.KafkaConfig {
		return dictconfig.ResolveKafkaConfig(ctx, db, base)
	}
	return service.NewKafkaProvider(resolve, base)
}

func provideKafkaToESService(kafka *service.KafkaProvider, es *service.ElasticsearchProvider) *service.KafkaToESService {
	return service.NewKafkaToESService(kafka, es)
}

func provideLogSearchService(
	es *service.ElasticsearchProvider,
	serverRepo interfaces.ServerRepository,
	dropRuleRepo interfaces.LogDropRuleRepository,
) *service.LogSearchService {
	return service.NewLogSearchService(es, serverRepo, dropRuleRepo)
}

func provideLogIntelligenceService(
	repo interfaces.LogIntelligenceRepository,
	logSearch *service.LogSearchService,
	projectRepo interfaces.ProjectRepository,
) *service.LogIntelligenceService {
	return service.NewLogIntelligenceService(repo, logSearch, projectRepo)
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
	clusterRepo interfaces.ClusterLogRepository,
	pipelineRepo interfaces.LogPipelineRepository,
	savedQueryRepo interfaces.LogSavedQueryRepository,
	dropRuleRepo interfaces.LogDropRuleRepository,
	projectRepo interfaces.ProjectRepository,
	es *service.ElasticsearchProvider,
	kafka *service.KafkaProvider,
	runtime *service.K8sRuntimeService,
	loggieCfg config.LoggieConfig,
) *service.ClusterLogService {
	return service.NewClusterLogService(
		clusterRepo, pipelineRepo, savedQueryRepo, dropRuleRepo,
		projectRepo, es, kafka, runtime, loggieCfg, loggieCfg.DaemonSetImage,
	)
}

var ServiceSet = wire.NewSet(
	// system
	service.NewLoginLogService,
	service.NewOperationLogService,
	provideAuthService,
	service.NewAlertRuleAssigneeService,
	provideUserService,
	providePasswordPolicyResolver,
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
	provideK8sPodService,
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
	provideAIService,
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
