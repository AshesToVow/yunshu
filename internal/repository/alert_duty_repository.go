package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertDutyRepository struct {
	db *gorm.DB
}

func NewAlertDutyRepository(db *gorm.DB) AlertDutyRepo {
	return &AlertDutyRepository{db: db}
}

func (r *AlertDutyRepository) List(ctx context.Context, f AlertDutyListFilter, offset, limit int) ([]model.AlertDutyBlock, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertDutyBlock{})
	if f.MonitorRuleID != nil && *f.MonitorRuleID > 0 {
		tx = tx.Where("monitor_rule_id = ?", *f.MonitorRuleID)
	}
	if f.ProjectID != nil {
		tx = tx.
			Joins("JOIN alert_monitor_rules amr ON amr.id = alert_duty_blocks.monitor_rule_id AND amr.deleted_at IS NULL").
			Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
			Where("ad.project_id = ?", *f.ProjectID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertDutyBlock
	err := tx.Order("starts_at ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertDutyRepository) GetByID(ctx context.Context, id uint) (*model.AlertDutyBlock, error) {
	var row model.AlertDutyBlock
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertDutyRepository) Create(ctx context.Context, row *model.AlertDutyBlock) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertDutyRepository) Save(ctx context.Context, row *model.AlertDutyBlock) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertDutyRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertDutyBlock{}, id)
	return res.RowsAffected, res.Error
}

func (r *AlertDutyRepository) HasActiveAtRule(ctx context.Context, monitorRuleID uint, t time.Time) (bool, error) {
	if monitorRuleID == 0 {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertDutyBlock{}).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Limit(1).Count(&n).Error
	return n > 0, err
}

func (r *AlertDutyRepository) ListActiveAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]model.AlertDutyBlock, error) {
	var list []model.AlertDutyBlock
	err := r.db.WithContext(ctx).
		Where("monitor_rule_id = ? AND starts_at <= ? AND ends_at >= ?", monitorRuleID, t, t).
		Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AlertDutyRepository) ListBetween(ctx context.Context, f AlertDutyListFilter, from, to time.Time) ([]model.AlertDutyBlock, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertDutyBlock{}).
		Where("starts_at < ? AND ends_at > ?", to, from)
	if f.MonitorRuleID != nil && *f.MonitorRuleID > 0 {
		tx = tx.Where("monitor_rule_id = ?", *f.MonitorRuleID)
	}
	if f.ProjectID != nil {
		tx = tx.
			Joins("JOIN alert_monitor_rules amr ON amr.id = alert_duty_blocks.monitor_rule_id AND amr.deleted_at IS NULL").
			Joins("JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
			Where("ad.project_id = ?", *f.ProjectID)
	}
	var list []model.AlertDutyBlock
	err := tx.Order("starts_at ASC").Find(&list).Error
	return list, err
}

var _ AlertDutyRepo = (*AlertDutyRepository)(nil)
