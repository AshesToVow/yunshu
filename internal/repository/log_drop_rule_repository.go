package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogDropRuleRepository struct {
	db *gorm.DB
}

func NewLogDropRuleRepository(db *gorm.DB) LogDropRuleRepo {
	return &LogDropRuleRepository{db: db}
}

func (r *LogDropRuleRepository) ListByProject(ctx context.Context, projectID uint) ([]model.LogDropRule, error) {
	var list []model.LogDropRule
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).
		Order("enabled DESC, id DESC").Find(&list).Error
	return list, err
}

func (r *LogDropRuleRepository) ListEnabledByProject(ctx context.Context, projectID uint) ([]model.LogDropRule, error) {
	var list []model.LogDropRule
	err := r.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).Find(&list).Error
	return list, err
}

func (r *LogDropRuleRepository) Create(ctx context.Context, row *model.LogDropRule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *LogDropRuleRepository) GetByIDInProject(ctx context.Context, projectID, ruleID uint) (*model.LogDropRule, error) {
	var row model.LogDropRule
	if err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *LogDropRuleRepository) Save(ctx context.Context, row *model.LogDropRule) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *LogDropRuleRepository) DeleteByIDInProject(ctx context.Context, projectID, ruleID uint) (int64, error) {
	res := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).Delete(&model.LogDropRule{})
	return res.RowsAffected, res.Error
}
