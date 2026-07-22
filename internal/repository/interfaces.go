package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"

	"gorm.io/gorm"
)

// LogSourceRepo is implemented by *LogSourceRepository.
type LogSourceRepo interface {
	GetByID(ctx context.Context, id uint) (*model.ServiceLogSource, error)
	GetByIDInProject(ctx context.Context, projectID uint, id uint) (*model.ServiceLogSource, error)
	Create(ctx context.Context, it *model.ServiceLogSource) (error)
	Save(ctx context.Context, it *model.ServiceLogSource) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	List(ctx context.Context, p LogSourceListParams) ([]model.ServiceLogSource, int64, error)
	BelongsToProjectServer(ctx context.Context, projectID uint, serverID uint, logSourceID uint) (bool, error)
	ListByProjectAndServer(ctx context.Context, projectID uint, serverID uint) ([]model.ServiceLogSource, error)
}

var _ LogSourceRepo = (*LogSourceRepository)(nil)

// LogRetentionRepo is implemented by *LogRetentionRepository.
type LogRetentionRepo interface {
	GetByScope(ctx context.Context, projectID, serverID uint) (*model.LogRetentionPolicy, error)
	List(ctx context.Context) ([]model.LogRetentionPolicy, error)
	Save(ctx context.Context, it *model.LogRetentionPolicy) error
	DeleteByScope(ctx context.Context, projectID, serverID uint) error
}

var _ LogRetentionRepo = (*LogRetentionRepository)(nil)

// LoggieAgentRepo is implemented by *LoggieAgentRepository.
type LoggieAgentRepo interface {
	GetByToken(ctx context.Context, token string) (*model.LoggieAgent, error)
	GetByProjectAndServer(ctx context.Context, projectID, serverID uint) (*model.LoggieAgent, error)
	ListByProject(ctx context.Context, projectID uint) ([]model.LoggieAgent, error)
	Save(ctx context.Context, it *model.LoggieAgent) error
	DeleteByProjectAndServer(ctx context.Context, projectID, serverID uint) error
	TouchSeen(ctx context.Context, id uint, at time.Time) error
}

var _ LoggieAgentRepo = (*LoggieAgentRepository)(nil)

// RegistrationRequestRepo is implemented by *RegistrationRequestRepository.
type RegistrationRequestRepo interface {
	Create(ctx context.Context, req *model.RegistrationRequest) (error)
	GetByID(ctx context.Context, id uint) (*model.RegistrationRequest, error)
	List(ctx context.Context, params RegistrationRequestListParams) ([]model.RegistrationRequest, int64, error)
	UpdateStatus(ctx context.Context, id uint, status model.RegistrationRequestStatus, reviewerID uint, comment string) (error)
	CountPending(ctx context.Context, username string, email string) (int64, error)
}

var _ RegistrationRequestRepo = (*RegistrationRequestRepository)(nil)

// ServerGroupRepo is implemented by *ServerGroupRepository.
type ServerGroupRepo interface {
	Create(ctx context.Context, item *model.ServerGroup) (error)
	Save(ctx context.Context, item *model.ServerGroup) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.ServerGroup, error)
	ListByProject(ctx context.Context, projectID uint) ([]model.ServerGroup, error)
}

var _ ServerGroupRepo = (*ServerGroupRepository)(nil)

// ServerRepo is implemented by *ServerRepository.
type ServerRepo interface {
	Create(ctx context.Context, s *model.Server) (error)
	Save(ctx context.Context, s *model.Server) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.Server, error)
	ProjectNameByID(ctx context.Context, projectID uint) (string, error)
	List(ctx context.Context, params ServerListParams) ([]model.Server, int64, error)
	GetByProjectProviderInstance(ctx context.Context, projectID uint, provider string, cloudInstanceID string) (*model.Server, error)
	ListByProjectWithoutGroup(ctx context.Context, projectID uint) ([]model.Server, error)
	ListByProjectGroupProvider(ctx context.Context, projectID uint, groupID uint, provider string) ([]model.Server, error)
	ListByProjectProviderCloud(ctx context.Context, projectID uint, provider string) ([]model.Server, error)
	UpsertCredential(ctx context.Context, cred *model.ServerCredential) (error)
	GetCredentialByServerID(ctx context.Context, serverID uint) (*model.ServerCredential, error)
}

var _ ServerRepo = (*ServerRepository)(nil)

// UserGroupRepo is implemented by *UserGroupRepository.
type UserGroupRepo interface {
	GetByID(ctx context.Context, id uint) (*model.UserGroup, error)
	GetByCode(ctx context.Context, code string) (*model.UserGroup, error)
	Create(ctx context.Context, g *model.UserGroup) (error)
	Save(ctx context.Context, g *model.UserGroup) (error)
	Delete(ctx context.Context, g *model.UserGroup) (error)
	List(ctx context.Context, params UserGroupListParams) ([]model.UserGroup, int64, error)
	ListMemberUserIDs(ctx context.Context, groupID uint) ([]uint, error)
	CountMembers(ctx context.Context, groupID uint) (int64, error)
	ReplaceMemberUserIDs(ctx context.Context, groupID uint, userIDs []uint) (error)
}

var _ UserGroupRepo = (*UserGroupRepository)(nil)

// DepartmentRepo is implemented by *DepartmentRepository.
type DepartmentRepo interface {
	DB() (*gorm.DB)
	Create(ctx context.Context, item *model.Department) (error)
	Save(ctx context.Context, item *model.Department) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	ClearLeaderByUserID(ctx context.Context, userID uint) (error)
	GetByID(ctx context.Context, id uint) (*model.Department, error)
	GetByCode(ctx context.Context, code string) (*model.Department, error)
	ListAll(ctx context.Context) ([]model.Department, error)
	ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error)
	ExistsByNameInParent(ctx context.Context, parentID *uint, name string, excludeID uint) (bool, error)
	CountChildren(ctx context.Context, id uint) (int64, error)
	CountUsers(ctx context.Context, id uint) (int64, error)
	ListDescendantIDsAndSelf(ctx context.Context, id uint) ([]uint, error)
}

var _ DepartmentRepo = (*DepartmentRepository)(nil)

// DictEntryRepo is implemented by *DictEntryRepository.
type DictEntryRepo interface {
	Create(ctx context.Context, item *model.DictEntry) (error)
	Update(ctx context.Context, item *model.DictEntry) (error)
	Delete(ctx context.Context, id uint) (error)
	DeleteByTypeAndLabel(ctx context.Context, dictType string, label string) (error)
	DeleteByTypes(ctx context.Context, dictTypes []string) (error)
	GetByID(ctx context.Context, id uint) (*model.DictEntry, error)
	ExistsByTypeValue(ctx context.Context, dictType string, value string, excludeID uint) (bool, error)
	ExistsByType(ctx context.Context, dictType string, excludeID uint) (bool, error)
	ExistsByTypeLabel(ctx context.Context, dictType string, label string, excludeID uint) (bool, error)
	DeleteByTypeAndValue(ctx context.Context, dictType string, value string) (error)
	CleanupDuplicateTypeValue(ctx context.Context) (error)
	CleanupDuplicateTypeLabel(ctx context.Context) (error)
	List(ctx context.Context, dictType string, keyword string, category string, status *int, page int, pageSize int) ([]model.DictEntry, int64, error)
	ListByTypeEnabled(ctx context.Context, dictType string) ([]model.DictEntry, error)
	ListByType(ctx context.Context, dictType string) ([]model.DictEntry, error)
	GetByDictTypeAndValue(ctx context.Context, dictType string, value string) (*model.DictEntry, error)
	GetByDictTypeAndLabel(ctx context.Context, dictType string, label string) (*model.DictEntry, error)
	NormalizeDictTypeCase(ctx context.Context) error
}

var _ DictEntryRepo = (*DictEntryRepository)(nil)

// K8sClusterRepo is implemented by *K8sClusterRepository.
type K8sClusterRepo interface {
	Create(ctx context.Context, cluster *model.K8sCluster) (error)
	Update(ctx context.Context, cluster *model.K8sCluster) (error)
	Delete(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.K8sCluster, error)
	List(ctx context.Context, params K8sClusterListParams) ([]model.K8sCluster, int64, error)
	ListAllBrief(ctx context.Context) ([]model.K8sCluster, error)
}

var _ K8sClusterRepo = (*K8sClusterRepository)(nil)

// OperationLogRepo is implemented by *OperationLogRepository.
type OperationLogRepo interface {
	Create(ctx context.Context, log *model.OperationLog) (error)
	List(ctx context.Context, p OperationLogListParams) ([]model.OperationLog, int64, error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByIDs(ctx context.Context, ids []uint) (error)
}

var _ OperationLogRepo = (*OperationLogRepository)(nil)

// RoleRepo is implemented by *RoleRepository.
type RoleRepo interface {
	Create(ctx context.Context, role *model.Role) (error)
	Save(ctx context.Context, role *model.Role) (error)
	Delete(ctx context.Context, role *model.Role) (error)
	GetByID(ctx context.Context, id uint) (*model.Role, error)
	GetByIDs(ctx context.Context, ids []uint) ([]model.Role, error)
	List(ctx context.Context, params RoleListParams) ([]model.Role, int64, error)
	ListAll(ctx context.Context) ([]model.Role, error)
}

var _ RoleRepo = (*RoleRepository)(nil)

// CloudAccountRepo is implemented by *CloudAccountRepository.
type CloudAccountRepo interface {
	Create(ctx context.Context, item *model.CloudAccount) (error)
	Save(ctx context.Context, item *model.CloudAccount) (error)
	GetByID(ctx context.Context, id uint) (*model.CloudAccount, error)
	ListByProjectAndGroup(ctx context.Context, projectID uint, groupID *uint) ([]model.CloudAccount, error)
	ListEnabledByProject(ctx context.Context, projectID uint, provider string) ([]model.CloudAccount, error)
	DeleteByID(ctx context.Context, id uint) (error)
}

var _ CloudAccountRepo = (*CloudAccountRepository)(nil)

// K8sClusterAccessRepo is implemented by *K8sClusterAccessRepository.
type K8sClusterAccessRepo interface {
	Upsert(ctx context.Context, it *model.K8sClusterAccessGrant) (error)
	ListGrantsApplyingToCluster(ctx context.Context, clusterID uint) ([]model.K8sClusterAccessGrant, error)
	ListByPrincipal(ctx context.Context, kind string, ref string) ([]model.K8sClusterAccessGrant, error)
	ListByRoleCode(ctx context.Context, roleCode string) ([]model.K8sClusterAccessGrant, error)
	GetByID(ctx context.Context, id uint) (*model.K8sClusterAccessGrant, error)
	DeleteByID(ctx context.Context, id uint) (error)
	EffectiveTier(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) (int)
	BuildEffectiveTierIndex(ctx context.Context, pack k8sauth.PrincipalPack) (EffectiveTierIndex, error)
	HasAnyK8sGrant(ctx context.Context, pack k8sauth.PrincipalPack) (bool)
}

var _ K8sClusterAccessRepo = (*K8sClusterAccessRepository)(nil)

// MysqlBackupRepo is implemented by *MysqlBackupRepository.
type MysqlBackupRepo interface {
	CreateInstance(ctx context.Context, inst *model.MysqlBackupInstance) (error)
	UpdateInstance(ctx context.Context, inst *model.MysqlBackupInstance) (error)
	DeleteInstance(ctx context.Context, id uint) (error)
	GetInstance(ctx context.Context, id uint) (*model.MysqlBackupInstance, error)
	GetInstanceInProject(ctx context.Context, projectID uint, id uint) (*model.MysqlBackupInstance, error)
	ListInstances(ctx context.Context, p MysqlBackupInstanceListParams) ([]model.MysqlBackupInstance, int64, error)
	CreateJob(ctx context.Context, job *model.MysqlBackupJob) (error)
	UpdateJob(ctx context.Context, job *model.MysqlBackupJob) (error)
	PatchJob(ctx context.Context, jobID uint, fields map[string]any) (error)
	GetJob(ctx context.Context, id uint) (*model.MysqlBackupJob, error)
	GetJobInProject(ctx context.Context, projectID, jobID uint) (*model.MysqlBackupJob, error)
	DeleteJob(ctx context.Context, projectID, jobID uint) error
	ListScheduleEnabledInstances(ctx context.Context) ([]model.MysqlBackupInstance, error)
	TouchLastScheduledAt(ctx context.Context, id uint, at time.Time) (error)
	HasRunningJob(ctx context.Context, instanceID uint) (bool, error)
	FailStaleRunningJobs(ctx context.Context, maxAge time.Duration) (int64, error)
	ListJobs(ctx context.Context, p MysqlBackupJobListParams) ([]model.MysqlBackupJob, int64, error)
}

var _ MysqlBackupRepo = (*MysqlBackupRepository)(nil)

// PermissionRepo is implemented by *PermissionRepository.
type PermissionRepo interface {
	Create(ctx context.Context, permission *model.Permission) (error)
	GetByResourceAction(ctx context.Context, resource, action string) (*model.Permission, error)
	Save(ctx context.Context, permission *model.Permission) (error)
	Delete(ctx context.Context, permission *model.Permission) (error)
	GetByID(ctx context.Context, id uint) (*model.Permission, error)
	List(ctx context.Context, params PermissionListParams) ([]model.Permission, int64, error)
	ListFiltered(ctx context.Context, params PermissionListParams) ([]model.Permission, error)
	ListAll(ctx context.Context) ([]model.Permission, error)
	BatchSetK8sScopeEnabled(ctx context.Context, params PermissionListParams, enabled bool) (int64, error)
}

var _ PermissionRepo = (*PermissionRepository)(nil)

// ProjectMemberRepo is implemented by *ProjectMemberRepository.
type ProjectMemberRepo interface {
	ListByProject(ctx context.Context, projectID uint) ([]model.ProjectMember, error)
	ListUserIDsByProject(ctx context.Context, projectID uint) ([]uint, error)
	DeleteByUserID(ctx context.Context, userID uint) (error)
	ListRolesByUserAndProjectIDs(ctx context.Context, userID uint, projectIDs []uint) (map[uint]string, error)
	ListProjectIDsByUser(ctx context.Context, userID uint) ([]uint, error)
	GetByID(ctx context.Context, id uint) (*model.ProjectMember, error)
	GetByProjectAndUser(ctx context.Context, projectID uint, userID uint) (*model.ProjectMember, error)
	Create(ctx context.Context, row *model.ProjectMember) (error)
	Save(ctx context.Context, row *model.ProjectMember) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByProject(ctx context.Context, projectID uint) (error)
	ListDisplayByProject(ctx context.Context, projectID uint) ([]ProjectMemberListRow, error)
}

var _ ProjectMemberRepo = (*ProjectMemberRepository)(nil)

// ProjectRepo is implemented by *ProjectRepository.
type ProjectRepo interface {
	Create(ctx context.Context, p *model.Project) (error)
	Save(ctx context.Context, p *model.Project) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	GetByID(ctx context.Context, id uint) (*model.Project, error)
	List(ctx context.Context, params ProjectListParams) ([]model.Project, int64, error)
	ListVisibleToUser(ctx context.Context, userID uint, params ProjectListParams) ([]model.Project, int64, error)
}

var _ ProjectRepo = (*ProjectRepository)(nil)

// UserRepo is implemented by *UserRepository.
type UserRepo interface {
	Create(ctx context.Context, user *model.User) (error)
	Save(ctx context.Context, user *model.User) (error)
	Delete(ctx context.Context, user *model.User) (error)
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByUsernameForAuth(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByEmailForAuth(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, params UserListParams) ([]model.User, int64, error)
	ReplaceRoles(ctx context.Context, user *model.User, roles []model.Role) (error)
	ExistsByUsernameOrEmail(ctx context.Context, username string, email string) (bool, error)
	ListAll(ctx context.Context) ([]model.User, error)
	ListByIDs(ctx context.Context, ids []uint) ([]model.User, error)
	ListUserIDsByRoleCode(ctx context.Context, roleCode string) ([]uint, error)
	ListActiveIDsByDepartmentIDs(ctx context.Context, deptIDs []uint) ([]uint, error)
	ListActiveIDsByDepartmentSubtree(ctx context.Context, rootDeptIDs []uint) ([]uint, error)
}

var _ UserRepo = (*UserRepository)(nil)

// LoginLogRepo is implemented by *LoginLogRepository.
type LoginLogRepo interface {
	Create(ctx context.Context, log *model.LoginLog) (error)
	List(ctx context.Context, p LoginLogListParams) ([]model.LoginLog, int64, error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByIDs(ctx context.Context, ids []uint) (error)
}

var _ LoginLogRepo = (*LoginLogRepository)(nil)

// MenuRepo is implemented by *MenuRepository.
type MenuRepo interface {
	Create(ctx context.Context, menu *model.Menu) (error)
	GetByID(ctx context.Context, id uint) (*model.Menu, error)
	Update(ctx context.Context, menu *model.Menu) (error)
	Delete(ctx context.Context, id uint) (error)
	ListAll(ctx context.Context) ([]model.Menu, error)
	Tree(ctx context.Context) ([]model.Menu, error)
	CountChildren(ctx context.Context, parentID uint) (int64, error)
	BatchUpdateStatus(ctx context.Context, ids []uint, status int) (error)
	ListPermissionBindings(ctx context.Context) ([]model.MenuPermissionBinding, error)
	ListPermissionBindingsByMenuID(ctx context.Context, menuID uint) ([]model.MenuPermissionBinding, error)
	ReplacePermissionBindings(ctx context.Context, menuID uint, bindings []model.MenuPermissionBinding) (error)
}

var _ MenuRepo = (*MenuRepository)(nil)

// ServiceRepo is implemented by *ServiceRepository.
type ServiceRepo interface {
	GetByID(ctx context.Context, id uint) (*model.Service, error)
	Create(ctx context.Context, s *model.Service) (error)
	Save(ctx context.Context, s *model.Service) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	List(ctx context.Context, p ServiceListParams) ([]model.Service, int64, error)
}

var _ ServiceRepo = (*ServiceRepository)(nil)

// K8sNamespaceAllowRepo is implemented by *K8sNamespaceAllowRepository.
type K8sNamespaceAllowRepo interface {
	WhitelistActiveForCluster(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) (bool, error)
	NamespaceAllowed(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error)
	WhitelistUnionNamespaces(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) ([]string, error)
	DistinctNamespacesForPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]string, error)
	List(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]model.K8sNamespaceAllowRule, error)
	Create(ctx context.Context, it *model.K8sNamespaceAllowRule) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) (error)
}

var _ K8sNamespaceAllowRepo = (*K8sNamespaceAllowRepository)(nil)

// K8sNamespaceDenyRepo is implemented by *K8sNamespaceDenyRepository.
type K8sNamespaceDenyRepo interface {
	IsDenied(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint, namespace string) (bool, error)
	DeniedNamespaceNames(ctx context.Context, pack k8sauth.PrincipalPack, clusterID uint) ([]string, error)
	List(ctx context.Context, principalKind string, principalRef string, clusterID uint) ([]model.K8sNamespaceDenyRule, error)
	Create(ctx context.Context, it *model.K8sNamespaceDenyRule) (error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByPrincipalCluster(ctx context.Context, principalKind string, principalRef string, clusterID uint) (error)
}

var _ K8sNamespaceDenyRepo = (*K8sNamespaceDenyRepository)(nil)

// DbmgmtRepo is implemented by *DbmgmtRepository.
type DbmgmtRepo interface {
	CreateInstance(ctx context.Context, inst *model.DbInstance) error
	UpdateInstance(ctx context.Context, inst *model.DbInstance) error
	DeleteInstance(ctx context.Context, id uint) error
	GetInstance(ctx context.Context, id uint) (*model.DbInstance, error)
	GetInstanceInProject(ctx context.Context, projectID, id uint) (*model.DbInstance, error)
	ListInstances(ctx context.Context, p DbInstanceListParams) ([]model.DbInstance, int64, error)
	ListAllInstances(ctx context.Context) ([]model.DbInstance, error)
	CountReplicasByPrimary(ctx context.Context, projectID, primaryID uint) (int64, error)

	CreateGrant(ctx context.Context, g *model.DbAccessGrant) error
	UpdateGrant(ctx context.Context, g *model.DbAccessGrant) error
	DeleteGrant(ctx context.Context, id uint) error
	GetGrantInProject(ctx context.Context, projectID, id uint) (*model.DbAccessGrant, error)
	ListGrants(ctx context.Context, projectID, instanceID uint) ([]model.DbAccessGrant, error)
	ListGrantsForPrincipal(ctx context.Context, projectID, instanceID uint, kind, ref string) ([]model.DbAccessGrant, error)

	CreateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error
	UpdateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error
	GetAccessRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAccessRequest, error)
	ListAccessRequests(ctx context.Context, p DbAccessRequestListParams) ([]model.DbAccessRequest, int64, error)
	CreateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error
	ListAccessRequestSteps(ctx context.Context, requestID uint) ([]model.DbAccessRequestStep, error)
	UpdateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error

	CreateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error
	UpdateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error
	GetSqlTicketInProject(ctx context.Context, projectID, id uint) (*model.DbSqlTicket, error)
	ListSqlTickets(ctx context.Context, p DbSqlTicketListParams) ([]model.DbSqlTicket, int64, error)
	CreateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error
	ListSqlTicketSteps(ctx context.Context, ticketID uint) ([]model.DbSqlTicketStep, error)
	UpdateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error

	CreateSqlExecution(ctx context.Context, ex *model.DbSqlExecution) error
	ListSqlExecutions(ctx context.Context, p DbSqlExecutionListParams) ([]model.DbSqlExecution, int64, error)
	CreateAuditLog(ctx context.Context, log *model.DbAuditLog) error
	ListAuditLogs(ctx context.Context, p DbAuditLogListParams) ([]model.DbAuditLog, int64, error)

	ListApprovalFlowStages(ctx context.Context, projectID uint) ([]model.DbApprovalFlowStage, error)
	UpsertApprovalFlowStage(ctx context.Context, stage *model.DbApprovalFlowStage) error
	DeleteApprovalFlowStagesNotIn(ctx context.Context, projectID uint, keys []string) error
	ListPendingAccessStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbAccessRequestStep, error)
	ListPendingTicketStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbSqlTicketStep, error)

	CreateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error
	UpdateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error
	GetAppUserRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAppUserRequest, error)
	ListAppUserRequests(ctx context.Context, p DbAppUserRequestListParams) ([]model.DbAppUserRequest, int64, error)
	CreateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error
	ListAppUserRequestSteps(ctx context.Context, requestID uint) ([]model.DbAppUserRequestStep, error)
	UpdateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error

	CreateInstanceAccount(ctx context.Context, acc *model.DbInstanceAccount) error
	ListInstanceAccounts(ctx context.Context, projectID, instanceID uint) ([]model.DbInstanceAccount, error)
	GetInstanceAccount(ctx context.Context, projectID, id uint) (*model.DbInstanceAccount, error)
}

var _ DbmgmtRepo = (*DbmgmtRepository)(nil)

