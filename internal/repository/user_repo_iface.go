package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

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

// RegistrationRequestRepo is implemented by *RegistrationRequestRepository.
type RegistrationRequestRepo interface {
	Create(ctx context.Context, req *model.RegistrationRequest) (error)
	GetByID(ctx context.Context, id uint) (*model.RegistrationRequest, error)
	List(ctx context.Context, params RegistrationRequestListParams) ([]model.RegistrationRequest, int64, error)
	UpdateStatus(ctx context.Context, id uint, status model.RegistrationRequestStatus, reviewerID uint, comment string) (error)
	CountPending(ctx context.Context, username string, email string) (int64, error)
}

var _ RegistrationRequestRepo = (*RegistrationRequestRepository)(nil)

// LoginLogRepo is implemented by *LoginLogRepository.
type LoginLogRepo interface {
	Create(ctx context.Context, log *model.LoginLog) (error)
	List(ctx context.Context, p LoginLogListParams) ([]model.LoginLog, int64, error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByIDs(ctx context.Context, ids []uint) (error)
}

var _ LoginLogRepo = (*LoginLogRepository)(nil)

