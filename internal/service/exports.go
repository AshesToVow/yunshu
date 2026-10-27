// Package service re-exports domain services from subpackages for handlers and router wiring.
package service

import (
	"yunshu/internal/service/ai"
	"yunshu/internal/service/alert"
	"yunshu/internal/service/cicd"
	"yunshu/internal/service/cmdb"
	"yunshu/internal/service/dbmgmt"
	"yunshu/internal/service/esmgmt"
	"yunshu/internal/service/inspect"
	"yunshu/internal/service/k8s"
	"yunshu/internal/service/k8s/eventforward"
	"yunshu/internal/service/kafkamgmt"
	"yunshu/internal/service/logplatform"
	"yunshu/internal/service/mysqlbackup"
	"yunshu/internal/service/overview"
	"yunshu/internal/service/platformtpl"
	"yunshu/internal/service/project"
	"yunshu/internal/service/system"
	"yunshu/internal/service/workflow"
)

// --- system ---
type (
	LoginLogService               = system.LoginLogService
	OperationLogService           = system.OperationLogService
	AuthService                   = system.AuthService
	UserService                   = system.UserService
	DepartmentService             = system.DepartmentService
	RoleService                   = system.RoleService
	PermissionService             = system.PermissionService
	PolicyService                 = system.PolicyService
	PolicyGovernanceService       = system.PolicyGovernanceService
	UserGroupService              = system.UserGroupService
	RegistrationService           = system.RegistrationService
	PasswordPolicyResolver        = system.PasswordPolicyResolver
	MenuService                   = system.MenuService
	DictEntryService              = system.DictEntryService
	LoginRequest                  = system.LoginRequest
	SendEmailCodeRequest          = system.SendEmailCodeRequest
	SendEmailCodeResponse         = system.SendEmailCodeResponse
	SendEmailCodeWithIPRequest    = system.SendEmailCodeWithIPRequest
	SendPasswordLoginCodeRequest  = system.SendPasswordLoginCodeRequest
	SendPasswordLoginCodeResponse = system.SendPasswordLoginCodeResponse
	EmailLoginRequest             = system.EmailLoginRequest
	RegisterRequest               = system.RegisterRequest
	RegisterResponse              = system.RegisterResponse
	LoginResponse                 = system.LoginResponse
	RefreshResponse               = system.RefreshResponse
	CreateWSTicketRequest         = system.CreateWSTicketRequest
	WSTicketResponse              = system.WSTicketResponse
	UpdateProfileRequest          = system.UpdateProfileRequest
	ChangePasswordRequest         = system.ChangePasswordRequest
)

var (
	NewLoginLogService         = system.NewLoginLogService
	NewOperationLogService     = system.NewOperationLogService
	NewAuthService             = system.NewAuthService
	NewUserService             = system.NewUserService
	NewDepartmentService       = system.NewDepartmentService
	NewRoleService             = system.NewRoleService
	NewPermissionService       = system.NewPermissionService
	NewPolicyService           = system.NewPolicyService
	NewPolicyGovernanceService = system.NewPolicyGovernanceService
	NewUserGroupService        = system.NewUserGroupService
	NewRegistrationService     = system.NewRegistrationService
	NewMenuService             = system.NewMenuService
	NewDictEntryService        = system.NewDictEntryService
	UserSubject                = system.UserSubject
	SyncUserRoles              = system.SyncUserRoles
	ReplaceRoleCode            = system.ReplaceRoleCode
	RemoveRolePolicies         = system.RemoveRolePolicies
	ReplacePermissionResource  = system.ReplacePermissionResource
	RemovePermissionPolicies   = system.RemovePermissionPolicies
	AddRolePolicies            = system.AddRolePolicies
	NewRoleItem                = system.NewRoleItem
	NewPermissionItem          = system.NewPermissionItem
	NewUserDetailResponse      = system.NewUserDetailResponse
	NewUserGroupItem           = system.NewUserGroupItem
)

type (
	UserDetailResponse                   = system.UserDetailResponse
	RoleItem                             = system.RoleItem
	PermissionItem                       = system.PermissionItem
	UserCreateRequest                    = system.UserCreateRequest
	UserUpdateRequest                    = system.UserUpdateRequest
	UserAssignRolesRequest               = system.UserAssignRolesRequest
	UserListQuery                        = system.UserListQuery
	DepartmentCreateRequest              = system.DepartmentCreateRequest
	DepartmentUpdateRequest              = system.DepartmentUpdateRequest
	DepartmentDetailResponse             = system.DepartmentDetailResponse
	ApplyRegisterRequest                 = system.ApplyRegisterRequest
	ReviewRequest                        = system.ReviewRequest
	PermissionCreateRequest              = system.PermissionCreateRequest
	PermissionUpdateRequest              = system.PermissionUpdateRequest
	PermissionBatchK8sScopeRequest       = system.PermissionBatchK8sScopeRequest
	PermissionBatchK8sScopeResponse      = system.PermissionBatchK8sScopeResponse
	PermissionListQuery                  = system.PermissionListQuery
	RoleCreateRequest                    = system.RoleCreateRequest
	RoleUpdateRequest                    = system.RoleUpdateRequest
	RoleListQuery                        = system.RoleListQuery
	UserGroupCreateRequest               = system.UserGroupCreateRequest
	UserGroupUpdateRequest               = system.UserGroupUpdateRequest
	UserGroupListQuery                   = system.UserGroupListQuery
	UserGroupAssignUsersRequest          = system.UserGroupAssignUsersRequest
	UserGroupItem                        = system.UserGroupItem
	UserGroupDetailResponse              = system.UserGroupDetailResponse
	PolicyGrantRequest                   = system.PolicyGrantRequest
	PolicyItemResponse                   = system.PolicyItemResponse
	PolicySimulateRequest                = system.PolicySimulateRequest
	PolicySimulateResponse               = system.PolicySimulateResponse
	PolicyConflictsResponse              = system.PolicyConflictsResponse
	PolicyConflictItem                   = system.PolicyConflictItem
	PermissionTreeResponse               = system.PermissionTreeResponse
	PermissionTreeNode                   = system.PermissionTreeNode
	PermissionMenuLinksResponse          = system.PermissionMenuLinksResponse
	MenuLink                             = system.MenuLink
	MenuPermissionBindingsResponse       = system.MenuPermissionBindingsResponse
	MenuPermissionBindingsReplaceRequest = system.MenuPermissionBindingsReplaceRequest
	MenuPermissionBindingItem            = system.MenuPermissionBindingItem
	OperationLogListQuery                = system.OperationLogListQuery
	LoginLogListQuery                    = system.LoginLogListQuery
	MenuCreatePayload                    = system.MenuCreatePayload
	MenuUpdatePayload                    = system.MenuUpdatePayload
	MenuBatchStatusPayload               = system.MenuBatchStatusPayload
	DictEntryListQuery                   = system.DictEntryListQuery
	DictEntryCreateRequest               = system.DictEntryCreateRequest
	DictEntryUpdateRequest               = system.DictEntryUpdateRequest
	DictEntryOption                      = system.DictEntryOption
	DictEntryListItem                    = system.DictEntryListItem
)

// --- overview ---
type OverviewService = overview.OverviewService

var NewOverviewService = overview.NewOverviewService

// --- project ---
type ProjectMgmtService = project.ProjectMgmtService

var NewProjectMgmtService = project.NewProjectMgmtService

type (
	ServiceCatalogService       = project.ServiceCatalogService
	ServiceCatalogListQuery     = project.ServiceCatalogListQuery
	ServiceCatalogUpsertRequest = project.ServiceCatalogUpsertRequest
	ServiceLinkRequest          = project.ServiceLinkRequest
	ServiceCatalogItem          = project.ServiceCatalogItem
	ChangeEventService          = project.ChangeEventService
	ChangeEventListQuery        = project.ChangeEventListQuery
	ServicePortrait             = project.ServicePortrait
	PortraitHealth              = project.PortraitHealth
)

var (
	NewServiceCatalogService = project.NewServiceCatalogService
	NewChangeEventService    = project.NewChangeEventService
)

// --- alert (cloud expiry CRUD) ---
type (
	CloudExpiryRuleService       = alert.CloudExpiryRuleService
	CloudExpiryRuleListQuery     = alert.CloudExpiryRuleListQuery
	CloudExpiryRuleUpsertRequest = alert.CloudExpiryRuleUpsertRequest
)

var (
	NewCloudExpiryRuleService   = alert.NewCloudExpiryRuleService
	ValidateCloudExpiryCronSpec = alert.ValidateCloudExpiryCronSpec
)

// --- cmdb ---
type CMDBService = cmdb.Service

var NewCMDBService = cmdb.NewService

type (
	ProjectItem                  = project.ProjectItem
	ProjectListQuery             = project.ProjectListQuery
	ProjectCreateRequest         = project.ProjectCreateRequest
	ProjectUpdateRequest         = project.ProjectUpdateRequest
	ServerItem                   = cmdb.ServerItem
	ServerListQuery              = cmdb.ServerListQuery
	ServerUpsertRequest          = cmdb.ServerUpsertRequest
	ServerDetailItem             = cmdb.ServerDetailItem
	ServerExecRequest            = cmdb.ServerExecRequest
	ServerExecResult             = cmdb.ServerExecResult
	HostProbeRequest             = cmdb.HostProbeRequest
	HostProbeResult              = cmdb.HostProbeResult
	HostProbeKind                = cmdb.HostProbeKind
	HostProbeCommandResult       = cmdb.HostProbeCommandResult
	ServerGroupItem              = cmdb.ServerGroupItem
	ServerGroupUpsertRequest     = cmdb.ServerGroupUpsertRequest
	ServerGroupTreeQuery         = cmdb.ServerGroupTreeQuery
	CloudAccountItem             = cmdb.CloudAccountItem
	CloudAccountListQuery        = cmdb.CloudAccountListQuery
	CloudAccountUpsertRequest    = cmdb.CloudAccountUpsertRequest
	CloudSyncRequest             = cmdb.CloudSyncRequest
	CloudSyncResult              = cmdb.CloudSyncResult
	CloudServerActionRequest     = cmdb.CloudServerActionRequest
	CloudServerActionResult      = cmdb.CloudServerActionResult
	BatchServerTestRequest       = cmdb.BatchServerTestRequest
	BatchServerTestResult        = cmdb.BatchServerTestResult
	ServerSyncRequest            = cmdb.ServerSyncRequest
	ServerSyncResult             = cmdb.ServerSyncResult
	ServerTestRequest            = cmdb.ServerTestRequest
	ServerTestResult             = cmdb.ServerTestResult
	ServerGrantItem              = cmdb.ServerGrantItem
	ServerGrantUpsertRequest     = cmdb.ServerGrantUpsertRequest
	ServerGrantBulkRequest       = cmdb.ServerGrantBulkRequest
	BootstrapServerGrantsRequest = cmdb.BootstrapServerGrantsRequest
	ServerAccessPerm             = cmdb.ServerAccessPerm
	ServerFileListQuery          = cmdb.ServerFileListQuery
	ServerFilePathQuery          = cmdb.ServerFilePathQuery
	LogSourceItem                = project.LogSourceItem
	LogSourceListQuery           = project.LogSourceListQuery
	LogSourceUpsertRequest       = project.LogSourceUpsertRequest
	ProjectMemberItem            = project.ProjectMemberItem
	ProjectMemberAddRequest      = project.ProjectMemberAddRequest
	ProjectMemberUpdateRequest   = project.ProjectMemberUpdateRequest
	ServiceItem                  = project.ServiceItem
	ServiceListQuery             = project.ServiceListQuery
	ServiceUpsertRequest         = project.ServiceUpsertRequest
)

// --- k8s ---
type (
	K8sRuntimeService           = k8s.K8sRuntimeService
	K8sClusterService           = k8s.K8sClusterService
	K8sPodService               = k8s.K8sPodService
	K8sNamespaceService         = k8s.K8sNamespaceService
	K8sNodeService              = k8s.K8sNodeService
	K8sWorkloadService          = k8s.K8sWorkloadService
	K8sConfigService            = k8s.K8sConfigService
	K8sStorageService           = k8s.K8sStorageService
	K8sServiceResourceService   = k8s.K8sServiceResourceService
	K8sIngressService           = k8s.K8sIngressService
	K8sNetworkPolicyService     = k8s.K8sNetworkPolicyService
	K8sDiscoveryService         = k8s.K8sDiscoveryService
	K8sHPAService               = k8s.K8sHPAService
	K8sHelmService              = k8s.K8sHelmService
	K8sEventService             = k8s.K8sEventService
	K8sCRDService               = k8s.K8sCRDService
	K8sCRService                = k8s.K8sCRService
	K8sRBACService              = k8s.K8sRBACService
	K8sServiceAccountService    = k8s.K8sServiceAccountService
	K8sScopedPolicyService      = k8s.K8sScopedPolicyService
	K8sNamespaceDenyService     = k8s.K8sNamespaceDenyService
	K8sNamespaceAllowService    = k8s.K8sNamespaceAllowService
	K8sResourceWatchQuery       = k8s.K8sResourceWatchQuery
	K8sEventForwardAdminService = eventforward.K8sEventForwardAdminService
	K8sSearchService            = k8s.K8sSearchService
	K8sSearchQuery              = k8s.K8sSearchQuery
	K8sSearchItem               = k8s.K8sSearchItem
	TopologyQuery               = k8s.TopologyQuery
	TopologyGraph               = k8s.TopologyGraph
	IngressDiagnoseQuery        = k8s.IngressDiagnoseQuery
	IngressDiagnoseResult       = k8s.IngressDiagnoseResult
)

var (
	NewK8sRuntimeService            = k8s.NewK8sRuntimeService
	NewK8sClusterService            = k8s.NewK8sClusterService
	NewK8sPodService                = k8s.NewK8sPodService
	NewK8sNamespaceService          = k8s.NewK8sNamespaceService
	NewK8sNodeService               = k8s.NewK8sNodeService
	NewK8sWorkloadService           = k8s.NewK8sWorkloadService
	NewK8sConfigService             = k8s.NewK8sConfigService
	NewK8sStorageService            = k8s.NewK8sStorageService
	NewK8sServiceResourceService    = k8s.NewK8sServiceResourceService
	NewK8sIngressService            = k8s.NewK8sIngressService
	NewK8sNetworkPolicyService      = k8s.NewK8sNetworkPolicyService
	NewK8sDiscoveryService          = k8s.NewK8sDiscoveryService
	NewK8sHPAService                = k8s.NewK8sHPAService
	NewK8sHelmService               = k8s.NewK8sHelmService
	NewK8sEventService              = k8s.NewK8sEventService
	NewK8sCRDService                = k8s.NewK8sCRDService
	NewK8sCRService                 = k8s.NewK8sCRService
	NewK8sRBACService               = k8s.NewK8sRBACService
	NewK8sServiceAccountService     = k8s.NewK8sServiceAccountService
	NewK8sScopedPolicyService       = k8s.NewK8sScopedPolicyService
	NewK8sNamespaceDenyService      = k8s.NewK8sNamespaceDenyService
	NewK8sNamespaceAllowService     = k8s.NewK8sNamespaceAllowService
	NewK8sEventForwardAdminService  = eventforward.NewK8sEventForwardAdminService
	NewK8sSearchService             = k8s.NewK8sSearchService
	ResolveWatchTarget              = k8s.ResolveWatchTarget
	WatchResourceLabel              = k8s.WatchResourceLabel
	IsK8sReadAPIPath                = k8s.IsK8sReadAPIPath
	IsK8sClusterGrantReadBypassPath = k8s.IsK8sClusterGrantReadBypassPath
	IsK8sNginxRestartRoute          = k8s.IsK8sNginxRestartRoute
	RequiredK8sAccessRank           = k8s.RequiredK8sAccessRank
	RequiredK8sCapability           = k8s.RequiredK8sCapability
	K8sScopeRouteKey                = k8s.K8sScopeRouteKey
	BuildK8sScopeMappings           = k8s.BuildK8sScopeMappings
)

const (
	K8sAccessRankReadonly     = k8s.K8sAccessRankReadonly
	K8sAccessRankReadonlyExec = k8s.K8sAccessRankReadonlyExec
	K8sAccessRankAdmin        = k8s.K8sAccessRankAdmin
)

type (
	RbacListQuery                    = k8s.RbacListQuery
	CronJobTriggerRequest            = k8s.CronJobTriggerRequest
	JobRerunRequest                  = k8s.JobRerunRequest
	ClusterKeywordQuery              = k8s.ClusterKeywordQuery
	ClusterNameQuery                 = k8s.ClusterNameQuery
	ClusterNamespaceKeywordQuery     = k8s.ClusterNamespaceKeywordQuery
	ClusterManifestApplyRequest      = k8s.ClusterManifestApplyRequest
	ClusterNamespaceNameQuery        = k8s.ClusterNamespaceNameQuery
	K8sClusterItem                   = k8s.K8sClusterItem
	K8sClusterSetStatusRequest       = k8s.K8sClusterSetStatusRequest
	K8sClusterAccessItem             = k8s.K8sClusterAccessItem
	K8sAuthMatrixRow                 = k8s.K8sAuthMatrixRow
	K8sUserClusterAuthRow            = k8s.K8sUserClusterAuthRow
	K8sEventForwardRuleUpsertRequest = eventforward.K8sEventForwardRuleUpsertRequest
	K8sForwardedEventListQuery       = eventforward.K8sForwardedEventListQuery
	RolloutUndoRequest               = k8s.RolloutUndoRequest
	RolloutUndoResult                = k8s.RolloutUndoResult
	DeploymentRevisionQuery          = k8s.DeploymentRevisionQuery
	DeploymentRevisionItem           = k8s.DeploymentRevisionItem
	ProgressiveScaleRequest          = k8s.ProgressiveScaleRequest
	ProgressivePatchImageRequest     = k8s.ProgressivePatchImageRequest
	ProgressiveSwitchServiceRequest  = k8s.ProgressiveSwitchServiceRequest
	ProgressiveEnsureCanaryRequest   = k8s.ProgressiveEnsureCanaryRequest
	PasswordPolicyResponse           = system.PasswordPolicyResponse
)

// --- logplatform ---
type (
	LogSearchService            = logplatform.LogSearchService
	LogSearchQuery              = logplatform.LogSearchQuery
	LogSearchItem               = logplatform.LogSearchItem
	LogIntelligenceService      = logplatform.LogIntelligenceService
	LogPatternListQuery         = logplatform.LogPatternListQuery
	LogPatternItem              = logplatform.LogPatternItem
	LogAnomalyListQuery         = logplatform.LogAnomalyListQuery
	LogAnomalyItem              = logplatform.LogAnomalyItem
	LogAnomalyUpdateRequest     = logplatform.LogAnomalyUpdateRequest
	LogContextQuery             = logplatform.LogContextQuery
	LogContextResult            = logplatform.LogContextResult
	LogOverviewResult           = logplatform.LogOverviewResult
	LogHistogramBucket          = logplatform.LogHistogramBucket
	NginxAccessStatsQuery       = logplatform.NginxAccessStatsQuery
	NginxAccessStatsResult      = logplatform.NginxAccessStatsResult
	NginxAccessURIStat          = logplatform.NginxAccessURIStat
	LogSignatureItem            = logplatform.LogSignatureItem
	LogSummaryResult            = logplatform.LogSummaryResult
	LogTopNQuery                = logplatform.LogTopNQuery
	LogTopNResult               = logplatform.LogTopNResult
	LogTopNItem                 = logplatform.LogTopNItem
	LogDropRuleUpsert           = logplatform.LogDropRuleUpsert
	LogDropRuleService          = logplatform.LogDropRuleService
	LogRetentionService         = logplatform.LogRetentionService
	ElasticsearchProvider       = logplatform.ElasticsearchProvider
	ElasticsearchConfigResolver = logplatform.ElasticsearchConfigResolver
	DictValueReader             = logplatform.DictValueReader
	DictValueWriter             = logplatform.DictValueWriter
	KafkaProvider               = logplatform.KafkaProvider
	KafkaConfigResolver         = logplatform.KafkaConfigResolver
	KafkaToESService            = logplatform.KafkaToESService
	LoggieAgentService          = logplatform.LoggieAgentService
	ClusterLogService           = logplatform.ClusterLogService
	ClusterLogRuleUpsert        = logplatform.ClusterLogRuleUpsert
	ClusterLogRuleItem          = logplatform.ClusterLogRuleItem
	ClusterPipelinesPreview     = logplatform.ClusterPipelinesPreview
	ClusterPipelinesUpsert      = logplatform.ClusterPipelinesUpsert
	LogPipelineUpsert           = logplatform.LogPipelineUpsert
	LogPipelineApplyRequest     = logplatform.LogPipelineApplyRequest
	LogFieldsResult             = logplatform.LogFieldsResult
	LogFieldStat                = logplatform.LogFieldStat
	LogSavedQueryUpsert         = logplatform.LogSavedQueryUpsert
	LogSavedQueryService        = logplatform.LogSavedQueryService
	LogRetentionItem            = logplatform.LogRetentionItem
	LogRetentionUpsertRequest   = logplatform.LogRetentionUpsertRequest
	LogRetentionCleanupResult   = logplatform.LogRetentionCleanupResult
	ESStorageStats              = logplatform.ESStorageStats
	KafkaQueueStats             = logplatform.KafkaQueueStats
	KafkaConfigPreviewItem      = logplatform.KafkaConfigPreviewItem
)

var (
	NewLogSearchService       = logplatform.NewLogSearchService
	NewLogIntelligenceService = logplatform.NewLogIntelligenceService
	RunLogIntelligenceWorker  = logplatform.RunIntelligenceWorker
	NewLogRetentionService    = logplatform.NewLogRetentionService
	NewElasticsearchProvider  = logplatform.NewElasticsearchProvider
	NewKafkaProvider          = logplatform.NewKafkaProvider
	NewKafkaToESService       = logplatform.NewKafkaToESService
	NewLoggieAgentService     = logplatform.NewLoggieAgentService
	NewClusterLogService      = logplatform.NewClusterLogService
	RunLogRetentionScheduler  = logplatform.RunLogRetentionScheduler
)

type (
	LoggieHeartbeatRequest       = logplatform.LoggieHeartbeatRequest
	LoggieBootstrapRequest       = logplatform.LoggieBootstrapRequest
	LoggieBootstrapResult        = logplatform.LoggieBootstrapResult
	LoggieDeployRequest          = logplatform.LoggieDeployRequest
	LoggieInstallRequest         = logplatform.LoggieInstallRequest
	LoggieUninstallRequest       = logplatform.LoggieUninstallRequest
	LoggieDeployResult           = logplatform.LoggieDeployResult
	LoggieBootstrapSourcePreview = logplatform.LoggieBootstrapSourcePreview
	LoggieStatusItem             = logplatform.LoggieStatusItem
	ESConfigPreviewItem          = logplatform.ESConfigPreviewItem
	SetESConnectionRequest       = logplatform.SetESConnectionRequest
)

// --- mysqlbackup ---
type (
	MysqlBackupService               = mysqlbackup.MysqlBackupService
	MysqlBackupInstanceItem          = mysqlbackup.MysqlBackupInstanceItem
	MysqlBackupInstanceListQuery     = mysqlbackup.MysqlBackupInstanceListQuery
	MysqlBackupInstanceUpsertRequest = mysqlbackup.MysqlBackupInstanceUpsertRequest
)

var NewMysqlBackupService = mysqlbackup.NewMysqlBackupService

// --- alert ---
type (
	AlertService                       = alert.AlertService
	AlertServiceOptions                = alert.AlertServiceOptions
	AlertQualityReport                 = alert.AlertQualityReport
	AlertEvidenceResult                = alert.AlertEvidenceResult
	AlertNoiseItem                     = alert.AlertNoiseItem
	AlertRepeatItem                    = alert.AlertRepeatItem
	AlertSilenceService                = alert.AlertSilenceService
	AlertMaintenanceService            = alert.AlertMaintenanceService
	AlertDutyService                   = alert.AlertDutyService
	AlertRuleAssigneeService           = alert.AlertRuleAssigneeService
	ReceiverGroupCache                 = alert.ReceiverGroupCache
	AlertDatasourceService             = alert.AlertDatasourceService
	AlertConsulService                 = alert.AlertConsulService
	AlertMonitorRuleService            = alert.AlertMonitorRuleService
	AlertRuleTemplate                  = alert.AlertRuleTemplate
	CreateFromTemplateRequest          = alert.CreateFromTemplateRequest
	AlertReceiverGroupService          = alert.AlertReceiverGroupService
	AlertInhibitionService             = alert.AlertInhibitionService
	AlertSubscriptionService           = alert.AlertSubscriptionService
	AlertManagerPayload                = alert.AlertManagerPayload
	PromqlSavedQueryUpsertRequest      = alert.PromqlSavedQueryUpsertRequest
	AlertManagerAlert                  = alert.AlertManagerAlert
	AlertChannelListQuery              = alert.AlertChannelListQuery
	AlertEventListQuery                = alert.AlertEventListQuery
	AlertChannelUpsertRequest          = alert.AlertChannelUpsertRequest
	AlertTestRequest                   = alert.AlertTestRequest
	AlertChannelTestResult             = alert.AlertChannelTestResult
	AlertRoutingDebugRequest           = alert.AlertRoutingDebugRequest
	AlertRoutingDebugResult            = alert.AlertRoutingDebugResult
	AlertEventGroupItem                = alert.AlertEventGroupItem
	AlertMaintenanceListQuery          = alert.AlertMaintenanceListQuery
	AlertMaintenanceUpsertRequest      = alert.AlertMaintenanceUpsertRequest
	AlertDutyCalendarQuery             = alert.AlertDutyCalendarQuery
	AlertDutyCalendarItem              = alert.AlertDutyCalendarItem
	AlertDutyValidateRequest           = alert.AlertDutyValidateRequest
	AlertDutyValidateResult            = alert.AlertDutyValidateResult
	AlertDutyHandoffRequest            = alert.AlertDutyHandoffRequest
	CanonicalIngressAlert              = alert.CanonicalIngressAlert
	AlertInhibitionRuleListQuery       = alert.AlertInhibitionRuleListQuery
	AlertInhibitionRuleUpsertRequest   = alert.AlertInhibitionRuleUpsertRequest
	AlertDatasourceListQuery           = alert.AlertDatasourceListQuery
	AlertDatasourceUpsertRequest       = alert.AlertDatasourceUpsertRequest
	DatasourceHealthResult             = alert.DatasourceHealthResult
	DatasourcePingResult               = alert.DatasourcePingResult
	AlertConsulEndpointListQuery       = alert.AlertConsulEndpointListQuery
	AlertConsulEndpointUpsertRequest   = alert.AlertConsulEndpointUpsertRequest
	AlertMonitorObjectListQuery        = alert.AlertMonitorObjectListQuery
	AlertConsulSyncResult              = alert.AlertConsulSyncResult
	AlertCurEventListQuery             = alert.AlertCurEventListQuery
	AlertCurEventView                  = alert.AlertCurEventView
	AlertHisEventListQuery             = alert.AlertHisEventListQuery
	AlertAckRequest                    = alert.AlertAckRequest
	AlertAckActiveInfo                 = alert.AlertAckActiveInfo
	AlertNoteCreateRequest             = alert.AlertNoteCreateRequest
	ImportPrometheusRulesRequest       = alert.ImportPrometheusRulesRequest
	ImportPrometheusRulesResult        = alert.ImportPrometheusRulesResult
	AlertMonitorRuleListQuery          = alert.AlertMonitorRuleListQuery
	AlertMonitorRuleUpsertRequest      = alert.AlertMonitorRuleUpsertRequest
	AlertSilenceListQuery              = alert.AlertSilenceListQuery
	AlertSilenceUpsertRequest          = alert.AlertSilenceUpsertRequest
	AlertSilenceBatchRequest           = alert.AlertSilenceBatchRequest
	PromQueryRequest                   = alert.PromQueryRequest
	PromQueryRangeRequest              = alert.PromQueryRangeRequest
	AlertRuleAssigneeUpsertRequest     = alert.AlertRuleAssigneeUpsertRequest
	AlertDutyBlockListQuery            = alert.AlertDutyBlockListQuery
	AlertDutyBlockUpsertRequest        = alert.AlertDutyBlockUpsertRequest
	AlertReceiverGroupListQuery        = alert.AlertReceiverGroupListQuery
	AlertSubscriptionNodeListQuery     = alert.AlertSubscriptionNodeListQuery
	AlertSubscriptionNodeUpsertRequest = alert.AlertSubscriptionNodeUpsertRequest
	AlertRoutingWizardRequest          = alert.AlertRoutingWizardRequest
	AlertRoutingWizardResult           = alert.AlertRoutingWizardResult
	SubscriptionMigrationReport        = alert.SubscriptionMigrationReport
	MigrateFromPoliciesOptions         = alert.MigrateFromPoliciesOptions
)

var (
	NewAlertService              = alert.NewAlertService
	NewAlertSilenceService       = alert.NewAlertSilenceService
	NewAlertMaintenanceService   = alert.NewAlertMaintenanceService
	NewAlertDutyService          = alert.NewAlertDutyService
	NewAlertRuleAssigneeService  = alert.NewAlertRuleAssigneeService
	NewReceiverGroupCache        = alert.NewReceiverGroupCache
	NewAlertDatasourceService    = alert.NewAlertDatasourceService
	NewAlertConsulService        = alert.NewAlertConsulService
	NewAlertMonitorRuleService   = alert.NewAlertMonitorRuleService
	NewAlertReceiverGroupService = alert.NewAlertReceiverGroupService
	NewAlertInhibitionService    = alert.NewAlertInhibitionServiceWithRepo
	NewAlertSubscriptionService  = alert.NewAlertSubscriptionService
	ListAlertRuleTemplates       = alert.ListAlertRuleTemplates
)

// --- auto-synced DTO aliases ---
type (
	NamespaceListItem              = k8s.NamespaceListItem
	NamespaceListQuery             = k8s.NamespaceListQuery
	PodEventQuery                  = k8s.PodEventQuery
	PodDebugRequest                = k8s.PodDebugRequest
	PodDebugResult                 = k8s.PodDebugResult
	PodExecRequest                 = k8s.PodExecRequest
	PodFileQuery                   = k8s.PodFileQuery
	PodListQuery                   = k8s.PodListQuery
	PodLogsQuery                   = k8s.PodLogsQuery
	MysqlBackupJobListQuery        = mysqlbackup.MysqlBackupJobListQuery
	SendLoginCodeByUsernameRequest = system.SendLoginCodeByUsernameRequest
)

// --- workflow ---
type (
	WorkflowService                   = workflow.Service
	CreateIncidentFromAlertRefRequest = workflow.CreateIncidentFromAlertRefRequest
	CreateTicketRequest               = workflow.CreateTicketRequest
	DefinitionKey                     = workflow.DefinitionKey
	DefinitionResponse                = workflow.DefinitionResponse
	DefinitionUpsertRequest           = workflow.DefinitionUpsertRequest
	PendingListQuery                  = workflow.PendingListQuery
	ReviewStepRequest                 = workflow.ReviewStepRequest
	StageItem                         = workflow.StageItem
	TicketDetail                      = workflow.TicketDetail
	WorkflowTicketListQuery           = workflow.TicketListQuery
)

var (
	NewWorkflowService       = workflow.NewService
	DefaultCicdStages        = workflow.DefaultCicdStages
	DefaultDbmgmtStages      = workflow.DefaultDbmgmtStages
	DefaultIncidentStages    = workflow.DefaultIncidentStages
	MigrateLegacyDefinitions = workflow.MigrateLegacyDefinitions
	MigrateLegacyTickets     = workflow.MigrateLegacyTickets
)

// --- ai ---
type (
	AIService                 = ai.Service
	PingRequest               = ai.PingRequest
	ChatEvent                 = ai.ChatEvent
	ChatRequest               = ai.ChatRequest
	SessionCreateRequest      = ai.SessionCreateRequest
	SessionListQuery          = ai.SessionListQuery
	SessionUpdateRequest      = ai.SessionUpdateRequest
	StartInvestigationRequest = ai.StartInvestigationRequest
	InvestigationListQuery    = ai.InvestigationListQuery
	LLMModelUpsertRequest     = ai.LLMModelUpsertRequest
	PromptUpsertRequest       = ai.PromptUpsertRequest
	PromptPublishRequest      = ai.PromptPublishRequest
	ToolUpsertRequest         = ai.ToolUpsertRequest
	EvalCaseUpsertRequest     = ai.EvalCaseUpsertRequest
	KBUpsertRequest           = ai.KBUpsertRequest
	KBDocUpsertRequest        = ai.KBDocUpsertRequest
	SOPUpsertRequest          = ai.SOPUpsertRequest
	IncidentCaseUpsertRequest = ai.IncidentCaseUpsertRequest
	AlertExplainAIRequest     = ai.AlertExplainAIRequest
	ApprovalListQuery         = ai.ApprovalListQuery
	CicdBuildFailAIRequest    = ai.CicdBuildFailAIRequest
	GenerateK8sYAMLRequest    = ai.GenerateK8sYAMLRequest
	LogAnalyzeAIRequest       = ai.LogAnalyzeAIRequest
	PipelineAdjustRequest     = ai.PipelineAdjustRequest
	PodDiagnoseAIRequest      = ai.PodDiagnoseAIRequest
	ReviewApprovalRequest     = ai.ReviewApprovalRequest
)

var NewAIService = ai.NewService

// --- cicd ---
type (
	CicdService                   = cicd.Service
	BuildRunItem                  = cicd.BuildRunItem
	BuildRunListQuery             = cicd.BuildRunListQuery
	TriggerBuildRequest           = cicd.TriggerBuildRequest
	BootstrapCicdGrantsRequest    = cicd.BootstrapCicdGrantsRequest
	CicdGrantBulkRequest          = cicd.CicdGrantBulkRequest
	CicdGrantUpsertRequest        = cicd.CicdGrantUpsertRequest
	CleanupPolicyUpsertRequest    = cicd.CleanupPolicyUpsertRequest
	PipelineTemplateUpsertRequest = cicd.PipelineTemplateUpsertRequest
	ProjectRegistryBindingRequest = cicd.ProjectRegistryBindingRequest
	RegistryItem                  = cicd.RegistryItem
	RegistryListQuery             = cicd.RegistryListQuery
	RegistryUpsertRequest         = cicd.RegistryUpsertRequest
	CicdApprovalFlowResponse      = cicd.ApprovalFlowResponse
	CicdApprovalFlowUpsertRequest = cicd.ApprovalFlowUpsertRequest
	BatchReleaseIDsRequest        = cicd.BatchReleaseIDsRequest
	PlatformRollbackRequest       = cicd.PlatformRollbackRequest
	ProgressiveAbortRequest       = cicd.ProgressiveAbortRequest
	ProgressivePromoteRequest     = cicd.ProgressivePromoteRequest
	ReleaseRunItem                = cicd.ReleaseRunItem
	ReleaseRunListQuery           = cicd.ReleaseRunListQuery
	ReviewReleaseRequest          = cicd.ReviewReleaseRequest
	TriggerReleaseRequest         = cicd.TriggerReleaseRequest
	CiConfigUpsertRequest         = cicd.CiConfigUpsertRequest
	CiConfigUpsertResult          = cicd.CiConfigUpsertResult
	DeployConfigUpsertRequest     = cicd.DeployConfigUpsertRequest
	DeployConfigUpsertResult      = cicd.DeployConfigUpsertResult
	HelmScaffoldQuery             = cicd.HelmScaffoldQuery
	CicdServiceItem               = cicd.ServiceItem
	CicdServiceListQuery          = cicd.ServiceListQuery
	CicdServiceUpsertRequest      = cicd.ServiceUpsertRequest
	K8sProgressiveFns             = cicd.K8sProgressiveFns
)

var (
	NewCicdService   = cicd.NewService
	ReadCallbackBody = cicd.ReadCallbackBody
)

// --- dbmgmt ---
type (
	DbmgmtService                   = dbmgmt.Service
	AccessApproveRequest            = dbmgmt.AccessApproveRequest
	AccessRequestCreateRequest      = dbmgmt.AccessRequestCreateRequest
	AccessRequestItem               = dbmgmt.AccessRequestItem
	AccessRequestListQuery          = dbmgmt.AccessRequestListQuery
	DbmgmtApprovalFlowResponse      = dbmgmt.ApprovalFlowResponse
	DbmgmtApprovalFlowUpsertRequest = dbmgmt.ApprovalFlowUpsertRequest
	GrantItem                       = dbmgmt.GrantItem
	GrantUpdateRequest              = dbmgmt.GrantUpdateRequest
	GrantUpsertRequest              = dbmgmt.GrantUpsertRequest
	DbmgmtReviewRequest             = dbmgmt.ReviewRequest
	InstanceItem                    = dbmgmt.InstanceItem
	InstanceListQuery               = dbmgmt.InstanceListQuery
	InstanceUpsertRequest           = dbmgmt.InstanceUpsertRequest
	ColumnMaskRuleUpsertRequest     = dbmgmt.ColumnMaskRuleUpsertRequest
	ExecuteRequest                  = dbmgmt.ExecuteRequest
	ExecuteResponse                 = dbmgmt.ExecuteResponse
	ImportRequest                   = dbmgmt.ImportRequest
	QueryRequest                    = dbmgmt.QueryRequest
	QueryResult                     = dbmgmt.QueryResult
	SQLCheckRequest                 = dbmgmt.SQLCheckRequest
	SQLCheckResponse                = dbmgmt.SQLCheckResponse
	OSCControlRequest               = dbmgmt.OSCControlRequest
	SubmitRollbackTicketRequest     = dbmgmt.SubmitRollbackTicketRequest
	TicketItem                      = dbmgmt.TicketItem
	DbmgmtTicketListQuery           = dbmgmt.TicketListQuery
	AppUserRequestCreateRequest     = dbmgmt.AppUserRequestCreateRequest
	AppUserRequestItem              = dbmgmt.AppUserRequestItem
	AppUserRequestListQuery         = dbmgmt.AppUserRequestListQuery
)

var NewDbmgmtService = dbmgmt.NewService

// --- esmgmt ---
type (
	EsmgmtService                 = esmgmt.Service
	BackupIndexRequest            = esmgmt.BackupIndexRequest
	EsmgmtConnectionUpsertRequest = esmgmt.ConnectionUpsertRequest
	CreateIndexRequest            = esmgmt.CreateIndexRequest
	ProxyRequest                  = esmgmt.ProxyRequest
	RestoreIndexRequest           = esmgmt.RestoreIndexRequest
	ScheduleUpsertRequest         = esmgmt.ScheduleUpsertRequest
	EsmgmtTestConnectionRequest   = esmgmt.TestConnectionRequest
	PutTemplateRequest            = esmgmt.PutTemplateRequest
	ReindexRequest                = esmgmt.ReindexRequest
	SearchDocsRequest             = esmgmt.SearchDocsRequest
	UpsertDocRequest              = esmgmt.UpsertDocRequest
)

var NewEsmgmtService = esmgmt.NewService

// --- inspect ---
type (
	InspectService              = inspect.Service
	ItemUpsertRequest           = inspect.ItemUpsertRequest
	PlanUpsertRequest           = inspect.PlanUpsertRequest
	PromoteAlertRequest         = inspect.PromoteAlertRequest
	ReportPreviewRequest        = inspect.ReportPreviewRequest
	ReportTemplateUpsertRequest = inspect.ReportTemplateUpsertRequest
	RunCreateRequest            = inspect.RunCreateRequest
	RunListQuery                = inspect.RunListQuery
	ReportStorageInfo           = inspect.ReportStorageInfo
	ReportStore                 = inspect.ReportStore
)

var (
	NewInspectService        = inspect.NewService
	EnhanceReportHTML        = inspect.EnhanceReportHTML
	ReadPDFLib               = inspect.ReadPDFLib
	ResolveReportStorageInfo = inspect.ResolveReportStorageInfo
	ResolveReportStore       = inspect.ResolveReportStore
)

// --- kafkamgmt ---
type (
	KafkamgmtService                 = kafkamgmt.Service
	KafkamgmtConnectionUpsertRequest = kafkamgmt.ConnectionUpsertRequest
	ConsumeRequest                   = kafkamgmt.ConsumeRequest
	CreatePartitionsRequest          = kafkamgmt.CreatePartitionsRequest
	CreateTopicRequest               = kafkamgmt.CreateTopicRequest
	DeleteGroupRequest               = kafkamgmt.DeleteGroupRequest
	DeleteTopicRequest               = kafkamgmt.DeleteTopicRequest
	ProduceRequest                   = kafkamgmt.ProduceRequest
	KafkamgmtTestConnectionRequest   = kafkamgmt.TestConnectionRequest
)

var NewKafkamgmtService = kafkamgmt.NewService

// --- platformtpl ---
type PlatformTplService = platformtpl.Service

var (
	NewPlatformTplService = platformtpl.NewService
	EnsureSeeded          = platformtpl.EnsureSeeded
)

// --- alert state (wire providers) ---
type AlertStateService = alert.AlertStateService

var NewRedisAlertStateService = alert.NewRedisAlertStateService

// --- alert (rule change / promql) ---
type (
	AlertRuleChangeService   = alert.AlertRuleChangeService
	ProposeRuleChangeRequest = alert.ProposeRuleChangeRequest
)

var (
	NewAlertRuleChangeService  = alert.NewAlertRuleChangeService
	NewPromqlSavedQueryService = alert.NewPromqlSavedQueryService
)

// --- k8s cr template ---
type (
	K8sCrTemplateService       = k8s.K8sCrTemplateService
	K8sCrTemplateUpsertRequest = k8s.K8sCrTemplateUpsertRequest
)

var NewK8sCrTemplateService = k8s.NewK8sCrTemplateService

// --- platformtpl (extra) ---
type (
	ListQuery        = platformtpl.ListQuery
	SaveDraftRequest = platformtpl.SaveDraftRequest
	TemplateItem     = platformtpl.TemplateItem
	UpsertRequest    = platformtpl.UpsertRequest
	VersionItem      = platformtpl.VersionItem
)

// --- system (permission sync) ---
var NewPermissionSyncService = system.NewPermissionSyncService

// --- eventforward (manager) ---
type EventForwardManager = eventforward.Manager

var (
	NewEventForwardManager = eventforward.NewManager
	EventForwardActive     = eventforward.Active
	SetEventForwardActive  = eventforward.SetActive
)
