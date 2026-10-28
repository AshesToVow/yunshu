package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertDutyRotationRepository struct {
	db *gorm.DB
}

func NewAlertDutyRotationRepository(db *gorm.DB) AlertDutyRotationRepo {
	return &AlertDutyRotationRepository{db: db}
}

func (r *AlertDutyRotationRepository) List(ctx context.Context, f AlertDutyRotationListFilter, offset, limit int) ([]model.AlertDutyRotation, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertDutyRotation{})
	if f.MonitorRuleID != nil && *f.MonitorRuleID > 0 {
		tx = tx.Where("monitor_rule_id = ?", *f.MonitorRuleID)
	}
	if f.Enabled != nil {
		tx = tx.Where("enabled = ?", *f.Enabled)
	}
	if f.ProjectID != nil {
		// 日志规则 datasource_id=0，项目归属落在 alert_monitor_rules.project_id；
		// PromQL 规则可能沿用数据源项目，两者取并集。
		tx = tx.
			Joins("JOIN alert_monitor_rules amr ON amr.id = alert_duty_rotations.monitor_rule_id AND amr.deleted_at IS NULL").
			Joins("LEFT JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
			Where("amr.project_id = ? OR ad.project_id = ?", *f.ProjectID, *f.ProjectID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertDutyRotation
	err := tx.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertDutyRotationRepository) GetByID(ctx context.Context, id uint) (*model.AlertDutyRotation, error) {
	var row model.AlertDutyRotation
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertDutyRotationRepository) Create(ctx context.Context, row *model.AlertDutyRotation) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertDutyRotationRepository) Save(ctx context.Context, row *model.AlertDutyRotation) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertDutyRotationRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertDutyRotation{}, id)
	return res.RowsAffected, res.Error
}

func (r *AlertDutyRotationRepository) ListEnabled(ctx context.Context) ([]model.AlertDutyRotation, error) {
	var list []model.AlertDutyRotation
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AlertDutyRotationRepository) ListEnabledByRule(ctx context.Context, monitorRuleID uint) ([]model.AlertDutyRotation, error) {
	var list []model.AlertDutyRotation
	err := r.db.WithContext(ctx).
		Where("monitor_rule_id = ? AND enabled = ?", monitorRuleID, true).
		Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AlertDutyRotationRepository) CountEnabledByRuleExcept(ctx context.Context, monitorRuleID, exceptID uint) (int64, error) {
	var n int64
	tx := r.db.WithContext(ctx).Model(&model.AlertDutyRotation{}).
		Where("monitor_rule_id = ? AND enabled = ?", monitorRuleID, true)
	if exceptID > 0 {
		tx = tx.Where("id <> ?", exceptID)
	}
	err := tx.Count(&n).Error
	return n, err
}

var _ AlertDutyRotationRepo = (*AlertDutyRotationRepository)(nil)
