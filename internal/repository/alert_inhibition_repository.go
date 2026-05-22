package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertInhibitionRuleRepository struct {
	db *gorm.DB
}

func NewAlertInhibitionRuleRepository(db *gorm.DB) AlertInhibitionRuleRepo {
	return &AlertInhibitionRuleRepository{db: db}
}

func (r *AlertInhibitionRuleRepository) ListEnabled(ctx context.Context) ([]model.AlertInhibitionRule, error) {
	var list []model.AlertInhibitionRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *AlertInhibitionRuleRepository) ListFiltered(ctx context.Context, f AlertInhibitionListFilter, offset, limit int) ([]model.AlertInhibitionRule, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertInhibitionRule{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if f.Enabled != nil {
		tx = tx.Where("enabled = ?", *f.Enabled)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertInhibitionRule
	err := tx.Order("priority ASC, id ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertInhibitionRuleRepository) GetByID(ctx context.Context, id uint) (*model.AlertInhibitionRule, error) {
	var rule model.AlertInhibitionRule
	if err := r.db.WithContext(ctx).First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *AlertInhibitionRuleRepository) Create(ctx context.Context, rule *model.AlertInhibitionRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *AlertInhibitionRuleRepository) Save(ctx context.Context, rule *model.AlertInhibitionRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *AlertInhibitionRuleRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertInhibitionRule{}, id)
	return res.RowsAffected, res.Error
}

func (r *AlertInhibitionRuleRepository) CreateEvent(ctx context.Context, event *model.AlertInhibitionEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

var _ AlertInhibitionRuleRepo = (*AlertInhibitionRuleRepository)(nil)
