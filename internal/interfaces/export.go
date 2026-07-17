// Package interfaces defines repository contracts and re-exports concrete bindings.
package interfaces

import "yunshu/internal/repository"

type (
	UserRepository                = repository.UserRepo
	DepartmentRepository          = repository.DepartmentRepo
	RoleRepository                = repository.RoleRepo
	PermissionRepository          = repository.PermissionRepo
	ProjectRepository             = repository.ProjectRepo
	ProjectMemberRepository       = repository.ProjectMemberRepo
	ServerRepository              = repository.ServerRepo
	ServerGroupRepository         = repository.ServerGroupRepo
	CloudAccountRepository        = repository.CloudAccountRepo
	ServiceRepository             = repository.ServiceRepo
	LogSourceRepository           = repository.LogSourceRepo
	LogRetentionRepository        = repository.LogRetentionRepo
	LoggieAgentRepository         = repository.LoggieAgentRepo
	MysqlBackupRepository         = repository.MysqlBackupRepo
	DbmgmtRepository              = repository.DbmgmtRepo
	LoginLogRepository            = repository.LoginLogRepo
	OperationLogRepository        = repository.OperationLogRepo
	MenuRepository                = repository.MenuRepo
	DictEntryRepository           = repository.DictEntryRepo
	RegistrationRequestRepository = repository.RegistrationRequestRepo
	UserGroupRepository           = repository.UserGroupRepo
	K8sClusterRepository          = repository.K8sClusterRepo
	K8sClusterAccessRepository    = repository.K8sClusterAccessRepo
	K8sNamespaceDenyRepository    = repository.K8sNamespaceDenyRepo
	K8sNamespaceAllowRepository   = repository.K8sNamespaceAllowRepo
	OverviewRepository            = repository.OverviewRepo
	K8sEventForwardRepository     = repository.K8sEventForwardRepo
)
