package router

import (
	"yunshu/internal/interfaces"
	"yunshu/internal/repository"

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
	CloudExpiryRule     interfaces.CloudExpiryRuleRepository
	Overview            interfaces.OverviewRepository
	K8sEventForward     interfaces.K8sEventForwardRepository
	ServiceCatalog      interfaces.ServiceCatalogRepository
	ChangeEvent         interfaces.ChangeEventRepository
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
		CloudExpiryRule:     repository.NewCloudExpiryRuleRepository(db),
		Overview:            repository.NewOverviewRepository(db),
		K8sEventForward:     repository.NewK8sEventForwardRepository(db),
		ServiceCatalog:      repository.NewServiceCatalogRepository(db),
		ChangeEvent:         repository.NewChangeEventRepository(db),
	}
}
