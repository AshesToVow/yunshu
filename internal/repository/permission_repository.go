package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

type PermissionListParams struct {
	Keyword  string
	Page     int
	PageSize int
	K8sScope   string // 空=全部；on/enabled/true/1；off/disabled/false/0
	K8sRelated string // 空=全部；on=仅集群资源接口路径
}

func NewPermissionRepository(db *gorm.DB) PermissionRepo {
	return &PermissionRepository{db: db}
}

func (r *PermissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	permission.Resource = strings.TrimSpace(permission.Resource)
	permission.Action = strings.ToUpper(strings.TrimSpace(permission.Action))
	return r.db.WithContext(ctx).Create(permission).Error
}

func (r *PermissionRepository) GetByResourceAction(ctx context.Context, resource, action string) (*model.Permission, error) {
	var permission model.Permission
	resource = strings.TrimSpace(resource)
	action = strings.ToUpper(strings.TrimSpace(action))
	err := r.db.WithContext(ctx).
		Where("resource = ? AND UPPER(action) = ?", resource, action).
		First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// GetSoftDeletedByResourceAction 查同键的软删除权限行。
// 唯一键 idx_perm_resource_action 不含 deleted_at，软删除行仍会阻塞重新 Create。
func (r *PermissionRepository) GetSoftDeletedByResourceAction(ctx context.Context, resource, action string) (*model.Permission, error) {
	var permission model.Permission
	resource = strings.TrimSpace(resource)
	action = strings.ToUpper(strings.TrimSpace(action))
	err := r.db.WithContext(ctx).Unscoped().
		Where("resource = ? AND UPPER(action) = ? AND deleted_at IS NOT NULL", resource, action).
		Order("id ASC").
		First(&permission).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

// Restore 复活软删除的权限行（清空 deleted_at），并同步最近一次同步写入的名称/描述。
func (r *PermissionRepository) Restore(ctx context.Context, id uint, name, description, action string) error {
	return r.db.WithContext(ctx).Unscoped().Model(&model.Permission{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"name":        name,
			"description": description,
			"action":      strings.ToUpper(strings.TrimSpace(action)),
			"deleted_at":  nil,
		}).Error
}

func (r *PermissionRepository) Save(ctx context.Context, permission *model.Permission) error {
	permission.Resource = strings.TrimSpace(permission.Resource)
	permission.Action = strings.ToUpper(strings.TrimSpace(permission.Action))
	return r.db.WithContext(ctx).Save(permission).Error
}

func (r *PermissionRepository) Delete(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Delete(permission).Error
}

func (r *PermissionRepository) GetByID(ctx context.Context, id uint) (*model.Permission, error) {
	var permission model.Permission
	err := r.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}

func applyPermissionListFilters(query *gorm.DB, params PermissionListParams) *gorm.DB {
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR resource LIKE ? OR action LIKE ?", keyword, keyword, keyword)
	}
	switch strings.ToLower(strings.TrimSpace(params.K8sScope)) {
	case "on", "enabled", "true", "1", "yes":
		query = query.Where("k8s_scope_enabled = ?", true)
	case "off", "disabled", "false", "0", "no":
		query = query.Where("k8s_scope_enabled = ?", false)
	}
	switch strings.ToLower(strings.TrimSpace(params.K8sRelated)) {
	case "on", "enabled", "true", "1", "yes":
		prefixes := constants.K8sClusterPermissionPathPrefixes
		if len(prefixes) > 0 {
			var parts []string
			var args []any
			for _, p := range prefixes {
				parts = append(parts, "resource LIKE ?")
				args = append(args, p+"%")
			}
			query = query.Where(strings.Join(parts, " OR "), args...)
		}
	}
	return query
}

func (r *PermissionRepository) List(ctx context.Context, params PermissionListParams) ([]model.Permission, int64, error) {
	query := applyPermissionListFilters(r.db.WithContext(ctx).Model(&model.Permission{}), params)
	var permissions []model.Permission
	page, pageSize := pagination.Normalize(params.Page, params.PageSize)
	total, err := listWithPagination(query, page, pageSize, "id DESC", &permissions)
	if err != nil {
		return nil, 0, err
	}
	return permissions, total, nil
}

func (r *PermissionRepository) ListFiltered(ctx context.Context, params PermissionListParams) ([]model.Permission, error) {
	query := applyPermissionListFilters(r.db.WithContext(ctx).Model(&model.Permission{}), params)
	var permissions []model.Permission
	if err := query.Order("id DESC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

// BatchSetK8sScopeEnabled 按列表筛选条件批量更新 k8s_scope_enabled。
func (r *PermissionRepository) BatchSetK8sScopeEnabled(ctx context.Context, params PermissionListParams, enabled bool) (int64, error) {
	query := applyPermissionListFilters(r.db.WithContext(ctx).Model(&model.Permission{}), params)
	result := query.Update("k8s_scope_enabled", enabled)
	return result.RowsAffected, result.Error
}

func (r *PermissionRepository) ListAll(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
