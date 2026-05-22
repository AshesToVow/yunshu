// Package service re-exports domain services from subpackages for handlers and router wiring.
package service

import (
	"yunshu/internal/service/alert"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/k8s/eventforward"
	"yunshu/internal/service/logplatform"
	"yunshu/internal/service/mysqlbackup"
	"yunshu/internal/service/overview"
	"yunshu/internal/service/project"
	"yunshu/internal/service/system"
)

// --- system ---
type (
	LoginLogService     = system.LoginLogService
	OperationLogService = system.OperationLogService
	AuthService         = system.AuthService
	UserService         = system.UserService
	DepartmentService   = system.DepartmentService
	RoleService         = system.RoleService
	PermissionService   = system.PermissionService
	PolicyService       = system.PolicyService
	UserGroupService    = system.UserGroupService
	RegistrationService = system.RegistrationService
	MenuService         = system.MenuService
	DictEntryService    = system.DictEntryService
	LoginRequest        = system.LoginRequest
	SendEmailCodeRequest         = system.SendEmailCodeRequest
	SendEmailCodeResponse        = system.SendEmailCodeResponse
	SendEmailCodeWithIPRequest   = system.SendEmailCodeWithIPRequest
	SendPasswordLoginCodeRequest = system.SendPasswordLoginCodeRequest
	SendPasswordLoginCodeResponse = system.SendPasswordLoginCodeResponse
	EmailLoginRequest            = system.EmailLoginRequest
	RegisterRequest              = system.RegisterRequest
	RegisterResponse             = system.RegisterResponse
	LoginResponse                = system.LoginResponse
	UpdateProfileRequest         = system.UpdateProfileRequest
	ChangePasswordRequest        = system.ChangePasswordRequest
)

var (
	NewLoginLogService     = system.NewLoginLogService
	NewOperationLogService = system.NewOperationLogService
	NewAuthService         = system.NewAuthService
	NewUserService         = system.NewUserService
	NewDepartmentService   = system.NewDepartmentService
	NewRoleService         = system.NewRoleService
	NewPermissionService   = system.NewPermissionService
	NewPolicyService       = system.NewPolicyService
	NewUserGroupService    = system.NewUserGroupService
	NewRegistrationService = system.NewRegistrationService
	NewMenuService         = system.NewMenuService
	NewDictEntryService    = system.NewDictEntryService
	UserSubject            = system.UserSubject
	SyncUserRoles          = system.SyncUserRoles
	ReplaceRoleCode        = system.ReplaceRoleCode
	RemoveRolePolicies     = system.RemoveRolePolicies
	ReplacePermissionResource = system.ReplacePermissionResource
	RemovePermissionPolicies  = system.RemovePermissionPolicies
	NewRoleItem            = system.NewRoleItem
	NewPermissionItem      = system.NewPermissionItem
	NewUserDetailResponse  = system.NewUserDetailResponse
	NewUserGroupItem       = system.NewUserGroupItem
)

type (
	UserDetailResponse       = system.UserDetailResponse
	RoleItem                 = system.RoleItem
	PermissionItem           = system.PermissionItem
	UserCreateRequest        = system.UserCreateRequest
	UserUpdateRequest        = system.UserUpdateRequest
	UserAssignRolesRequest   = system.UserAssignRolesRequest
	UserListQuery            = system.UserListQuery
	DepartmentCreateRequest  = system.DepartmentCreateRequest
	DepartmentUpdateRequest  = system.DepartmentUpdateRequest
	DepartmentDetailResponse = system.DepartmentDetailResponse
	ApplyRegisterRequest     = system.ApplyRegisterRequest
	ReviewRequest            = system.ReviewRequest
	PermissionCreateRequest  = system.PermissionCreateRequest
	PermissionUpdateRequest  = system.PermissionUpdateRequest
	PermissionListQuery      = system.PermissionListQuery
	RoleCreateRequest        = system.RoleCreateRequest
	RoleUpdateRequest        = system.RoleUpdateRequest
	RoleListQuery            = system.RoleListQuery
	UserGroupCreateRequest   = system.UserGroupCreateRequest
	UserGroupUpdateRequest   = system.UserGroupUpdateRequest
	UserGroupListQuery       = system.UserGroupListQuery
	UserGroupAssignUsersRequest = system.UserGroupAssignUsersRequest
	UserGroupItem            = system.UserGroupItem
	UserGroupDetailResponse  = system.UserGroupDetailResponse
	PolicyGrantRequest       = system.PolicyGrantRequest
	PolicyItemResponse       = system.PolicyItemResponse
	OperationLogListQuery    = system.OperationLogListQuery
	LoginLogListQuery        = system.LoginLogListQuery
	MenuCreatePayload        = system.MenuCreatePayload
	MenuUpdatePayload        = system.MenuUpdatePayload
	MenuBatchStatusPayload   = system.MenuBatchStatusPayload
	DictEntryListQuery       = system.DictEntryListQuery
	DictEntryCreateRequest   = system.DictEntryCreateRequest
	DictEntryUpdateRequest   = system.DictEntryUpdateRequest
	DictEntryOption          = system.DictEntryOption
)

// --- overview ---
type OverviewService = overview.OverviewService

var NewOverviewService = overview.NewOverviewService

// --- project ---
type (
	ProjectMgmtService    = project.ProjectMgmtService
	CloudExpiryRuleService = project.CloudExpiryRuleService
)

var (
	NewProjectMgmtService    = project.NewProjectMgmtService
	NewCloudExpiryRuleService = project.NewCloudExpiryRuleService
	ValidateCloudExpiryCronSpec = project.ValidateCloudExpiryCronSpec
)

type (
	ProjectItem              = project.ProjectItem
	ProjectListQuery         = project.ProjectListQuery
	ProjectCreateRequest     = project.ProjectCreateRequest
	ProjectUpdateRequest     = project.ProjectUpdateRequest
	ServerItem               = project.ServerItem
	ServerListQuery          = project.ServerListQuery
	ServerUpsertRequest      = project.ServerUpsertRequest
	ServerDetailItem         = project.ServerDetailItem
	ServerExecRequest        = project.ServerExecRequest
	ServerExecResult         = project.ServerExecResult
	ServerGroupItem          = project.ServerGroupItem
	ServerGroupUpsertRequest = project.ServerGroupUpsertRequest
	ServerGroupTreeQuery     = project.ServerGroupTreeQuery
	CloudAccountItem         = project.CloudAccountItem
	CloudAccountListQuery    = project.CloudAccountListQuery
	CloudAccountUpsertRequest = project.CloudAccountUpsertRequest
	CloudSyncRequest         = project.CloudSyncRequest
	CloudSyncResult          = project.CloudSyncResult
	CloudServerActionRequest = project.CloudServerActionRequest
	CloudServerActionResult  = project.CloudServerActionResult
	BatchServerTestRequest   = project.BatchServerTestRequest
	BatchServerTestResult    = project.BatchServerTestResult
	ServerSyncRequest        = project.ServerSyncRequest
	ServerSyncResult         = project.ServerSyncResult
	ServerTestRequest        = project.ServerTestRequest
	ServerTestResult         = project.ServerTestResult
	LogSourceItem            = project.LogSourceItem
	LogSourceListQuery       = project.LogSourceListQuery
	LogSourceUpsertRequest   = project.LogSourceUpsertRequest
	LogStreamQuery           = project.LogStreamQuery
	LogExportQuery           = project.LogExportQuery
	RemoteLogFileQuery       = project.RemoteLogFileQuery
	RemoteLogUnitQuery       = project.RemoteLogUnitQuery
	ProjectMemberItem        = project.ProjectMemberItem
	ProjectMemberAddRequest  = project.ProjectMemberAddRequest
	ProjectMemberUpdateRequest = project.ProjectMemberUpdateRequest
	CloudExpiryRuleListQuery = project.CloudExpiryRuleListQuery
	CloudExpiryRuleUpsertRequest = project.CloudExpiryRuleUpsertRequest
	ServiceItem              = project.ServiceItem
	ServiceListQuery         = project.ServiceListQuery
	ServiceUpsertRequest     = project.ServiceUpsertRequest
)

// --- k8s ---
type (
	K8sRuntimeService         = k8s.K8sRuntimeService
	K8sClusterService         = k8s.K8sClusterService
	K8sPodService             = k8s.K8sPodService
	K8sNamespaceService       = k8s.K8sNamespaceService
	K8sNodeService            = k8s.K8sNodeService
	K8sWorkloadService        = k8s.K8sWorkloadService
	K8sConfigService          = k8s.K8sConfigService
	K8sStorageService         = k8s.K8sStorageService
	K8sServiceResourceService = k8s.K8sServiceResourceService
	K8sIngressService         = k8s.K8sIngressService
	K8sNetworkPolicyService   = k8s.K8sNetworkPolicyService
	K8sDiscoveryService       = k8s.K8sDiscoveryService
	K8sHPAService             = k8s.K8sHPAService
	K8sEventService           = k8s.K8sEventService
	K8sCRDService             = k8s.K8sCRDService
	K8sCRService              = k8s.K8sCRService
	K8sRBACService            = k8s.K8sRBACService
	K8sServiceAccountService  = k8s.K8sServiceAccountService
	K8sScopedPolicyService    = k8s.K8sScopedPolicyService
	K8sNamespaceDenyService   = k8s.K8sNamespaceDenyService
	K8sNamespaceAllowService  = k8s.K8sNamespaceAllowService
	K8sResourceWatchQuery     = k8s.K8sResourceWatchQuery
	K8sEventForwardAdminService = eventforward.K8sEventForwardAdminService
)

var (
	NewK8sRuntimeService         = k8s.NewK8sRuntimeService
	NewK8sClusterService         = k8s.NewK8sClusterService
	NewK8sPodService             = k8s.NewK8sPodService
	NewK8sNamespaceService       = k8s.NewK8sNamespaceService
	NewK8sNodeService            = k8s.NewK8sNodeService
	NewK8sWorkloadService        = k8s.NewK8sWorkloadService
	NewK8sConfigService          = k8s.NewK8sConfigService
	NewK8sStorageService         = k8s.NewK8sStorageService
	NewK8sServiceResourceService = k8s.NewK8sServiceResourceService
	NewK8sIngressService         = k8s.NewK8sIngressService
	NewK8sNetworkPolicyService   = k8s.NewK8sNetworkPolicyService
	NewK8sDiscoveryService       = k8s.NewK8sDiscoveryService
	NewK8sHPAService             = k8s.NewK8sHPAService
	NewK8sEventService           = k8s.NewK8sEventService
	NewK8sCRDService             = k8s.NewK8sCRDService
	NewK8sCRService              = k8s.NewK8sCRService
	NewK8sRBACService            = k8s.NewK8sRBACService
	NewK8sServiceAccountService  = k8s.NewK8sServiceAccountService
	NewK8sScopedPolicyService    = k8s.NewK8sScopedPolicyService
	NewK8sNamespaceDenyService   = k8s.NewK8sNamespaceDenyService
	NewK8sNamespaceAllowService  = k8s.NewK8sNamespaceAllowService
	NewK8sEventForwardAdminService = eventforward.NewK8sEventForwardAdminService
	ResolveWatchTarget           = k8s.ResolveWatchTarget
	WatchResourceLabel           = k8s.WatchResourceLabel
	IsK8sReadAPIPath             = k8s.IsK8sReadAPIPath
	IsK8sNginxRestartRoute       = k8s.IsK8sNginxRestartRoute
	RequiredK8sAccessRank        = k8s.RequiredK8sAccessRank
	K8sScopeRouteKey             = k8s.K8sScopeRouteKey
	BuildK8sScopeMappings        = k8s.BuildK8sScopeMappings
)

const (
	K8sAccessRankReadonly     = k8s.K8sAccessRankReadonly
	K8sAccessRankReadonlyExec = k8s.K8sAccessRankReadonlyExec
	K8sAccessRankAdmin        = k8s.K8sAccessRankAdmin
)

type (
	RbacListQuery              = k8s.RbacListQuery
	CronJobTriggerRequest      = k8s.CronJobTriggerRequest
	JobRerunRequest            = k8s.JobRerunRequest
	ClusterKeywordQuery        = k8s.ClusterKeywordQuery
	ClusterNameQuery           = k8s.ClusterNameQuery
	ClusterNamespaceKeywordQuery = k8s.ClusterNamespaceKeywordQuery
	ClusterManifestApplyRequest = k8s.ClusterManifestApplyRequest
	ClusterNamespaceNameQuery  = k8s.ClusterNamespaceNameQuery
	K8sClusterItem             = k8s.K8sClusterItem
	K8sClusterSetStatusRequest = k8s.K8sClusterSetStatusRequest
	K8sClusterAccessItem       = k8s.K8sClusterAccessItem
	K8sAuthMatrixRow           = k8s.K8sAuthMatrixRow
	K8sUserClusterAuthRow      = k8s.K8sUserClusterAuthRow
	K8sEventForwardRuleUpsertRequest = eventforward.K8sEventForwardRuleUpsertRequest
)

// --- logplatform ---
type (
	LogAgentService       = logplatform.LogAgentService
	AgentDiscoveryService = logplatform.AgentDiscoveryService
)

var (
	NewLogAgentService       = logplatform.NewLogAgentService
	NewAgentDiscoveryService = logplatform.NewAgentDiscoveryService
	BuildLogStreamKey        = logplatform.BuildLogStreamKey
)

var AgentLogBroker = logplatform.AgentLogBroker

const MaxLogHistoryPerStream = logplatform.MaxLogHistoryPerStream

type (
	AgentBootstrapRequest            = logplatform.AgentBootstrapRequest
	AgentBootstrapResult             = logplatform.AgentBootstrapResult
	AgentRuntimeSource               = logplatform.AgentRuntimeSource
	AgentRuntimeConfigResult         = logplatform.AgentRuntimeConfigResult
	AgentBatchHeartbeatRefreshRequest = logplatform.AgentBatchHeartbeatRefreshRequest
	AgentBatchHeartbeatRefreshResult  = logplatform.AgentBatchHeartbeatRefreshResult
	AgentDiscoveryItem               = logplatform.AgentDiscoveryItem
	AgentDiscoveryReportRequest      = logplatform.AgentDiscoveryReportRequest
	AgentDiscoveryReportResult       = logplatform.AgentDiscoveryReportResult
	AgentDiscoveryListQuery          = logplatform.AgentDiscoveryListQuery
	AgentDiscoveryListItem           = logplatform.AgentDiscoveryListItem
	LogAgentRegisterRequest          = logplatform.LogAgentRegisterRequest
	LogAgentPublicRegisterRequest    = logplatform.LogAgentPublicRegisterRequest
	LogAgentRegisterResult           = logplatform.LogAgentRegisterResult
	LogAgentHealthReportRequest      = logplatform.LogAgentHealthReportRequest
	LogAgentStatusResult             = logplatform.LogAgentStatusResult
	LogAgentListQuery                = logplatform.LogAgentListQuery
	LogAgentListItem                 = logplatform.LogAgentListItem
	AgentLogEvent                    = logplatform.AgentLogEvent
)

// --- mysqlbackup ---
type (
	MysqlBackupService              = mysqlbackup.MysqlBackupService
	MysqlBackupInstanceItem         = mysqlbackup.MysqlBackupInstanceItem
	MysqlBackupInstanceListQuery    = mysqlbackup.MysqlBackupInstanceListQuery
	MysqlBackupInstanceUpsertRequest = mysqlbackup.MysqlBackupInstanceUpsertRequest
)

var NewMysqlBackupService = mysqlbackup.NewMysqlBackupService

// --- alert ---
type (
	AlertService              = alert.AlertService
	AlertServiceOptions       = alert.AlertServiceOptions
	AlertSilenceService       = alert.AlertSilenceService
	AlertDutyService          = alert.AlertDutyService
	AlertRuleAssigneeService  = alert.AlertRuleAssigneeService
	ReceiverGroupCache        = alert.ReceiverGroupCache
	AlertDatasourceService    = alert.AlertDatasourceService
	AlertMonitorRuleService   = alert.AlertMonitorRuleService
	AlertReceiverGroupService = alert.AlertReceiverGroupService
	AlertInhibitionService    = alert.AlertInhibitionService
	AlertSubscriptionService  = alert.AlertSubscriptionService
	AlertManagerPayload       = alert.AlertManagerPayload
	AlertManagerAlert         = alert.AlertManagerAlert
	AlertChannelListQuery     = alert.AlertChannelListQuery
	AlertEventListQuery       = alert.AlertEventListQuery
	AlertChannelUpsertRequest = alert.AlertChannelUpsertRequest
	AlertTestRequest          = alert.AlertTestRequest
	CanonicalIngressAlert     = alert.CanonicalIngressAlert
	AlertInhibitionRuleListQuery   = alert.AlertInhibitionRuleListQuery
	AlertInhibitionRuleUpsertRequest = alert.AlertInhibitionRuleUpsertRequest
	AlertDatasourceListQuery       = alert.AlertDatasourceListQuery
	AlertDatasourceUpsertRequest     = alert.AlertDatasourceUpsertRequest
	AlertMonitorRuleListQuery      = alert.AlertMonitorRuleListQuery
	AlertMonitorRuleUpsertRequest  = alert.AlertMonitorRuleUpsertRequest
	AlertSilenceListQuery          = alert.AlertSilenceListQuery
	AlertSilenceUpsertRequest      = alert.AlertSilenceUpsertRequest
	AlertSilenceBatchRequest       = alert.AlertSilenceBatchRequest
	PromQueryRequest               = alert.PromQueryRequest
	PromQueryRangeRequest          = alert.PromQueryRangeRequest
	AlertRuleAssigneeUpsertRequest = alert.AlertRuleAssigneeUpsertRequest
	AlertDutyBlockListQuery        = alert.AlertDutyBlockListQuery
	AlertDutyBlockUpsertRequest    = alert.AlertDutyBlockUpsertRequest
	AlertReceiverGroupListQuery    = alert.AlertReceiverGroupListQuery
	AlertSubscriptionNodeListQuery = alert.AlertSubscriptionNodeListQuery
	AlertSubscriptionNodeUpsertRequest = alert.AlertSubscriptionNodeUpsertRequest
	SubscriptionMigrationReport    = alert.SubscriptionMigrationReport
	MigrateFromPoliciesOptions     = alert.MigrateFromPoliciesOptions
)

var (
	NewAlertService              = alert.NewAlertService
	NewAlertSilenceService       = alert.NewAlertSilenceService
	NewAlertDutyService          = alert.NewAlertDutyService
	NewAlertRuleAssigneeService  = alert.NewAlertRuleAssigneeService
	NewReceiverGroupCache        = alert.NewReceiverGroupCache
	NewAlertDatasourceService    = alert.NewAlertDatasourceService
	NewAlertMonitorRuleService   = alert.NewAlertMonitorRuleService
	NewAlertReceiverGroupService = alert.NewAlertReceiverGroupService
	NewAlertInhibitionService    = alert.NewAlertInhibitionServiceWithRepo
	NewAlertSubscriptionService  = alert.NewAlertSubscriptionService
)


// --- auto-synced DTO aliases ---
type (
	NamespaceListItem = k8s.NamespaceListItem
	NamespaceListQuery = k8s.NamespaceListQuery
	PodEventQuery = k8s.PodEventQuery
	PodExecRequest = k8s.PodExecRequest
	PodFileQuery = k8s.PodFileQuery
	PodListQuery = k8s.PodListQuery
	PodLogsQuery = k8s.PodLogsQuery
	MysqlBackupJobListQuery = mysqlbackup.MysqlBackupJobListQuery
	SendLoginCodeByUsernameRequest = system.SendLoginCodeByUsernameRequest
)
