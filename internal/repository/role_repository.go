package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type RoleRepository struct {
	db *gorm.DB
}

type RoleListParams struct {
	Keyword  string
	Page     int
	PageSize int
}

func NewRoleRepository(db *gorm.DB) RoleRepo {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Create(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *RoleRepository) Save(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *RoleRepository) Delete(ctx context.Context, role *model.Role) error {
	if role == nil {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 软删除角色不会触发外键级联，需显式清理 user_roles 绑定，避免残留孤儿行。
		if err := tx.Exec("DELETE FROM user_roles WHERE role_id = ?", role.ID).Error; err != nil {
			return err
		}
		return tx.Delete(role).Error
	})
}

func (r *RoleRepository) GetByID(ctx context.Context, id uint) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *RoleRepository) GetByIDs(ctx context.Context, ids []uint) ([]model.Role, error) {
	if len(ids) == 0 {
		return []model.Role{}, nil
	}

	var roles []model.Role
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ExistsByCode 软删除行的 deleted_at 为 NULL，复合唯一索引无法约束活跃行，需在服务层显式查重。
func (r *RoleRepository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.Role{}).Where("code = ?", code)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RoleRepository) List(ctx context.Context, params RoleListParams) ([]model.Role, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.Role{})
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", keyword, keyword)
	}

	var roles []model.Role
	total, err := listWithPagination(query, params.Page, params.PageSize, "id DESC", &roles)
	if err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func (r *RoleRepository) ListAll(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
