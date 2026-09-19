package router

import (
	aisvc "yunshu/internal/service/ai"
	cicdsvc "yunshu/internal/service/cicd"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
	esmgmtsvc "yunshu/internal/service/esmgmt"
	inspectsvc "yunshu/internal/service/inspect"
	"yunshu/internal/service"
)

// routeServices aggregates HTTP-layer domain services built from routeRepositories.
type routeServices struct {
	LoginLog             *service.LoginLogService
	OperationLog         *service.OperationLogService
	Auth                 *service.AuthService
	User                 *service.UserService
	Department           *service.DepartmentService
	Role                 *service.RoleService
	Permission           *service.PermissionService
	Policy               *service.PolicyService
	PolicyGovernance     *service.PolicyGovernanceService
	K8sScopedPolicy      *service.K8sScopedPolicyService
	K8sNamespaceDeny     *service.K8sNamespaceDenyService
	K8sNamespaceAllow    *service.K8sNamespaceAllowService
	UserGroup            *service.UserGroupService
	Registration         *service.RegistrationService
	Menu                 *service.MenuService
	DictEntry            *service.DictEntryService
	AlertSilence         *service.AlertSilenceService
	AlertDuty            *service.AlertDutyService
	AlertAssignee        *service.AlertRuleAssigneeService
	AlertReceiverCache   *service.ReceiverGroupCache
	Alert                *service.AlertService
	CloudExpiryRule      *service.CloudExpiryRuleService
	AlertDatasource      *service.AlertDatasourceService
	AlertMonitorRule     *service.AlertMonitorRuleService
	K8sRuntime           *service.K8sRuntimeService
	K8sCluster           *service.K8sClusterService
	K8sPod               *service.K8sPodService
	K8sNamespace         *service.K8sNamespaceService
	K8sNode              *service.K8sNodeService
	K8sWorkload          *service.K8sWorkloadService
	K8sConfig            *service.K8sConfigService
	K8sStorage           *service.K8sStorageService
	K8sServiceResource   *service.K8sServiceResourceService
	K8sIngress           *service.K8sIngressService
	K8sNetworkPolicy     *service.K8sNetworkPolicyService
	K8sDiscovery         *service.K8sDiscoveryService
	K8sHPA               *service.K8sHPAService
	K8sHelm              *service.K8sHelmService
	K8sEvent             *service.K8sEventService
	K8sCRD               *service.K8sCRDService
	K8sCR                *service.K8sCRService
	K8sRBAC              *service.K8sRBACService
	K8sServiceAccount    *service.K8sServiceAccountService
	Overview             *service.OverviewService
	ProjectMgmt          *service.ProjectMgmtService
	ServiceCatalog       *service.ServiceCatalogService
	ChangeEvent          *service.ChangeEventService
	CMDB                 *service.CMDBService
	Cicd                 *cicdsvc.Service
	MysqlBackup          *service.MysqlBackupService
	Dbmgmt               *dbmgmtsvc.Service
	LogSearch            *service.LogSearchService
	LogIntelligence      *service.LogIntelligenceService
	LogRetention         *service.LogRetentionService
	KafkaToES            *service.KafkaToESService
	LoggieAgent          *service.LoggieAgentService
	ClusterLog           *service.ClusterLogService
	AlertReceiverGroup   *service.AlertReceiverGroupService
	K8sEventForwardAdmin *service.K8sEventForwardAdminService
	K8sSearch            *service.K8sSearchService
	AlertMaintenance     *service.AlertMaintenanceService
	Inspect              *inspectsvc.Service
	AI                   *aisvc.Service
	Esmgmt               *esmgmtsvc.Service
}
