package router

import (
	"yunshu/internal/interfaces"
	"yunshu/internal/repository"
	"yunshu/internal/service/changeevent"

	"gorm.io/gorm"
)

type routeRepositories struct {
	User              interfaces.UserRepository
	Department        interfaces.DepartmentRepository
	Role              interfaces.RoleRepository
	Permission        interfaces.PermissionRepository
	LoginLog          interfaces.LoginLogRepository
	OperationLog      interfaces.OperationLogRepository
	ProjectMember     interfaces.ProjectMemberRepository
	K8sNsDeny         interfaces.K8sNamespaceDenyRepository
	K8sNsAllow        interfaces.K8sNamespaceAllowRepository
	UserGroup         interfaces.UserGroupRepository
	K8sClusterAccess  interfaces.K8sClusterAccessRepository
	Cluster           interfaces.K8sClusterRepository
	Project           interfaces.ProjectRepository
	RegRequest        interfaces.RegistrationRequestRepository
	Menu              interfaces.MenuRepository
	DictEntry         interfaces.DictEntryRepository
	Server            interfaces.ServerRepository
	ServerGroup       interfaces.ServerGroupRepository
	CloudAccount      interfaces.CloudAccountRepository
	Service           interfaces.ServiceRepository
	LogSource         interfaces.LogSourceRepository
	LogRetention      interfaces.LogRetentionRepository
	LoggieAgent       interfaces.LoggieAgentRepository
	LogSavedQuery     interfaces.LogSavedQueryRepository
	LogDropRule       interfaces.LogDropRuleRepository
	LogIntelligence   interfaces.LogIntelligenceRepository
	ClusterLog        interfaces.ClusterLogRepository
	LogPipeline       interfaces.LogPipelineRepository
	MysqlBackup       interfaces.MysqlBackupRepository
	Dbmgmt            interfaces.DbmgmtRepository
	AlertEvent          interfaces.AlertEventRepository
	AlertChannel        interfaces.AlertChannelRepository
	AlertSilence        interfaces.AlertSilenceRepository
	AlertMaintenance    interfaces.AlertMaintenanceWindowRepository
	AlertInhibitionRule interfaces.AlertInhibitionRuleRepository
	AlertSubscription   interfaces.AlertSubscriptionRepository
	AlertDatasource     interfaces.AlertDatasourceRepository
	AlertMonitorRule    interfaces.AlertMonitorRuleRepository
	AlertReceiverGroup  interfaces.AlertReceiverGroupRepository
	AlertDuty           interfaces.AlertDutyRepository
	AlertRuleAssignee   interfaces.AlertRuleAssigneeRepository
	AlertFiringDelivery interfaces.AlertFiringDeliveryRepository
	AlertAck            interfaces.AlertAckRepository
	AlertProgressNote   interfaces.AlertProgressNoteRepository
	AlertCurHis         interfaces.AlertCurHisRepository
	AlertConsul         interfaces.AlertConsulRepository
	AlertRuleChange     interfaces.AlertRuleChangeRepository
	PromqlSavedQuery    interfaces.PromqlSavedQueryRepository
	CloudExpiryRule     interfaces.CloudExpiryRuleRepository
	Overview            interfaces.OverviewRepository
	K8sEventForward     interfaces.K8sEventForwardRepository
	K8sCrTemplate       interfaces.K8sCrTemplateRepository
	K8sWorkloadSnapshot interfaces.K8sWorkloadSnapshotRepository
	HarborMerge         interfaces.HarborMergeRepository
	ServiceCatalog      interfaces.ServiceCatalogRepository
	ServicePortrait     interfaces.ServicePortraitRepository
	ChangeEvent         interfaces.ChangeEventRepository
	ServerAccessGrant   interfaces.ServerAccessGrantRepository
	PlatformTemplate    interfaces.PlatformTemplateRepository
	Workflow            interfaces.WorkflowRepository
	Esmgmt              interfaces.EsmgmtRepository
	Inspect             interfaces.InspectRepository
	Cicd                interfaces.CicdRepository
	Ai                  interfaces.AiRepository
}

func newRouteRepositories(db *gorm.DB) *routeRepositories {
	return &routeRepositories{
		User:             repository.NewUserRepository(db),
		Department:       repository.NewDepartmentRepository(db),
		Role:             repository.NewRoleRepository(db),
		Permission:       repository.NewPermissionRepository(db),
		LoginLog:         repository.NewLoginLogRepository(db),
		OperationLog:     repository.NewOperationLogRepository(db),
		ProjectMember:    repository.NewProjectMemberRepository(db),
		K8sNsDeny:        repository.NewK8sNamespaceDenyRepository(db),
		K8sNsAllow:       repository.NewK8sNamespaceAllowRepository(db),
		UserGroup:        repository.NewUserGroupRepository(db),
		K8sClusterAccess: repository.NewK8sClusterAccessRepository(db),
		Cluster:          repository.NewK8sClusterRepository(db),
		Project:          repository.NewProjectRepository(db),
		RegRequest:       repository.NewRegistrationRequestRepository(db),
		Menu:             repository.NewMenuRepository(db),
		DictEntry:        repository.NewDictEntryRepository(db),
		Server:           repository.NewServerRepository(db),
		ServerGroup:      repository.NewServerGroupRepository(db),
		CloudAccount:     repository.NewCloudAccountRepository(db),
		Service:          repository.NewServiceRepository(db),
		LogSource:        repository.NewLogSourceRepository(db),
		LogRetention:     repository.NewLogRetentionRepository(db),
		LoggieAgent:      repository.NewLoggieAgentRepository(db),
		LogSavedQuery:    repository.NewLogSavedQueryRepository(db),
		LogDropRule:      repository.NewLogDropRuleRepository(db),
		LogIntelligence:  repository.NewLogIntelligenceRepository(db),
		ClusterLog:       repository.NewClusterLogRepository(db),
		LogPipeline:      repository.NewLogPipelineRepository(db),
		MysqlBackup:      repository.NewMysqlBackupRepository(db),
		Dbmgmt:           repository.NewDbmgmtRepository(db),
		AlertEvent:          repository.NewAlertEventRepository(db),
		AlertChannel:        repository.NewAlertChannelRepository(db),
		AlertSilence:        repository.NewAlertSilenceRepository(db),
		AlertMaintenance:    repository.NewAlertMaintenanceWindowRepository(db),
		AlertInhibitionRule: repository.NewAlertInhibitionRuleRepository(db),
		AlertSubscription:   repository.NewAlertSubscriptionRepository(db),
		AlertDatasource:     repository.NewAlertDatasourceRepository(db),
		AlertMonitorRule:    repository.NewAlertMonitorRuleRepository(db),
		AlertReceiverGroup:  repository.NewAlertReceiverGroupRepository(db),
		AlertDuty:           repository.NewAlertDutyRepository(db),
		AlertRuleAssignee:   repository.NewAlertRuleAssigneeRepository(db),
		AlertFiringDelivery: repository.NewAlertFiringDeliveryRepository(db),
		AlertAck:            repository.NewAlertAckRepository(db),
		AlertProgressNote:   repository.NewAlertProgressNoteRepository(db),
		AlertCurHis:         repository.NewAlertCurHisRepository(db),
		AlertConsul:         repository.NewAlertConsulRepository(db),
		AlertRuleChange:     repository.NewAlertRuleChangeRepository(db),
		PromqlSavedQuery:    repository.NewPromqlSavedQueryRepository(db),
		CloudExpiryRule:     repository.NewCloudExpiryRuleRepository(db),
		Overview:            repository.NewOverviewRepository(db),
		K8sEventForward:     repository.NewK8sEventForwardRepository(db),
		K8sCrTemplate:       repository.NewK8sCrTemplateRepository(db),
		K8sWorkloadSnapshot: repository.NewK8sWorkloadSnapshotRepository(db),
		HarborMerge:         repository.NewHarborMergeRepository(db),
		ServiceCatalog:      repository.NewServiceCatalogRepository(db),
		ServicePortrait:     repository.NewServicePortraitRepository(db),
		ChangeEvent:         bindChangeEventRepo(db),
		ServerAccessGrant:   repository.NewServerAccessGrantRepository(db),
		PlatformTemplate:    repository.NewPlatformTemplateRepository(db),
		Workflow:            repository.NewWorkflowRepository(db),
		Esmgmt:              repository.NewEsmgmtRepository(db),
		Inspect:             repository.NewInspectRepository(db),
		Cicd:                repository.NewCicdRepository(db),
		Ai:                  repository.NewAiRepository(db),
	}
}

func bindChangeEventRepo(db *gorm.DB) interfaces.ChangeEventRepository {
	repo := repository.NewChangeEventRepository(db)
	changeevent.BindRepo(repo)
	return repo
}
