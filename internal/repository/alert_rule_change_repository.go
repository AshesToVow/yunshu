package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertRuleChangeRepository struct {
	db *gorm.DB
}

func NewAlertRuleChangeRepository(db *gorm.DB) AlertRuleChangeRepo {
	return &AlertRuleChangeRepository{db: db}
}

func (r *AlertRuleChangeRepository) Create(ctx context.Context, row *model.AlertMonitorRuleChangeRequest) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertRuleChangeRepository) ListPending(ctx context.Context) ([]model.AlertMonitorRuleChangeRequest, error) {
	var list []model.AlertMonitorRuleChangeRequest
	err := r.db.WithContext(ctx).
		Where("status = ?", model.AlertRuleChangePending).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *AlertRuleChangeRepository) GetByID(ctx context.Context, id uint) (*model.AlertMonitorRuleChangeRequest, error) {
	var row model.AlertMonitorRuleChangeRequest
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertRuleChangeRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.AlertMonitorRuleChangeRequest{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *AlertRuleChangeRepository) UpdatePending(ctx context.Context, id uint, fields map[string]any) (int64, error) {
	if len(fields) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&model.AlertMonitorRuleChangeRequest{}).
		Where("id = ? AND status = ?", id, model.AlertRuleChangePending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

var _ AlertRuleChangeRepo = (*AlertRuleChangeRepository)(nil)
