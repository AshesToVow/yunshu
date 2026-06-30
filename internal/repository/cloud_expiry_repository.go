package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type CloudExpiryRuleRepository struct {
	db *gorm.DB
}

func NewCloudExpiryRuleRepository(db *gorm.DB) CloudExpiryRuleRepo {
	return &CloudExpiryRuleRepository{db: db}
}

func (r *CloudExpiryRuleRepository) List(ctx context.Context, keyword string, offset, limit int) ([]model.CloudExpiryRule, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.CloudExpiryRule{}).
		Select("cloud_expiry_rules.*, p.name AS project_name").
		Joins("LEFT JOIN projects p ON p.id = cloud_expiry_rules.project_id AND p.deleted_at IS NULL")
	if kw := strings.TrimSpace(keyword); kw != "" {
		tx = tx.Where("cloud_expiry_rules.name LIKE ? OR cloud_expiry_rules.region_scope LIKE ?", "%"+kw+"%", "%"+kw+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.CloudExpiryRule
	err := tx.Order("cloud_expiry_rules.id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *CloudExpiryRuleRepository) Create(ctx context.Context, row *model.CloudExpiryRule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *CloudExpiryRuleRepository) GetByID(ctx context.Context, id uint) (*model.CloudExpiryRule, error) {
	var row model.CloudExpiryRule
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CloudExpiryRuleRepository) Save(ctx context.Context, row *model.CloudExpiryRule) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *CloudExpiryRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.CloudExpiryRule{}, id).Error
}

func (r *CloudExpiryRuleRepository) ListEnabled(ctx context.Context) ([]model.CloudExpiryRule, error) {
	var list []model.CloudExpiryRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("id ASC").Find(&list).Error
	return list, err
}

var _ CloudExpiryRuleRepo = (*CloudExpiryRuleRepository)(nil)
