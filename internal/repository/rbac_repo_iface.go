package repository

import (
	"context"

	"yunshu/internal/model"
)

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

// OperationLogRepo is implemented by *OperationLogRepository.
type OperationLogRepo interface {
	Create(ctx context.Context, log *model.OperationLog) (error)
	List(ctx context.Context, p OperationLogListParams) ([]model.OperationLog, int64, error)
	DeleteByID(ctx context.Context, id uint) (error)
	DeleteByIDs(ctx context.Context, ids []uint) (error)
}

var _ OperationLogRepo = (*OperationLogRepository)(nil)

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

