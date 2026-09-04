package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertMonitorRuleRepository struct {
	db *gorm.DB
}

func NewAlertMonitorRuleRepository(db *gorm.DB) AlertMonitorRuleRepo {
	return &AlertMonitorRuleRepository{db: db}
}

func (r *AlertMonitorRuleRepository) List(ctx context.Context, f AlertMonitorRuleListFilter, offset, limit int) ([]model.AlertMonitorRule, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertMonitorRule{})
	if f.DatasourceID != nil && *f.DatasourceID > 0 {
		tx = tx.Where("datasource_id = ?", *f.DatasourceID)
	}
	if f.ProjectID != nil && *f.ProjectID > 0 {
		tx = tx.Where("datasource_id IN (?)",
			r.db.WithContext(ctx).Model(&model.AlertDatasource{}).Select("id").Where("project_id = ?", *f.ProjectID),
		)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR expr LIKE ?", like, like)
	}
	if f.Enabled != nil {
		tx = tx.Where("enabled = ?", *f.Enabled)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertMonitorRule
	err := tx.Order("id ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertMonitorRuleRepository) ListEnabled(ctx context.Context) ([]model.AlertMonitorRule, error) {
	var list []model.AlertMonitorRule
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *AlertMonitorRuleRepository) ListEnabledWithProject(ctx context.Context) ([]EvalMonitorRule, error) {
	var rules []EvalMonitorRule
	// LEFT JOIN：日志规则无 datasource；project_id 优先取规则列，否则取数据源。
	err := r.db.WithContext(ctx).
		Table("alert_monitor_rules amr").
		Select("amr.*, COALESCE(NULLIF(amr.project_id, 0), ad.project_id, 0) AS project_id").
		Joins("LEFT JOIN alert_datasources ad ON ad.id = amr.datasource_id AND ad.deleted_at IS NULL").
		Where("amr.enabled = ? AND amr.deleted_at IS NULL", true).
		Find(&rules).Error
	return rules, err
}

func (r *AlertMonitorRuleRepository) GetByID(ctx context.Context, id uint) (*model.AlertMonitorRule, error) {
	var row model.AlertMonitorRule
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertMonitorRuleRepository) Create(ctx context.Context, row *model.AlertMonitorRule) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertMonitorRuleRepository) Save(ctx context.Context, row *model.AlertMonitorRule) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertMonitorRuleRepository) DeleteCascade(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("monitor_rule_id = ?", id).Delete(&model.AlertRuleAssignee{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&model.AlertMonitorRule{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

var _ AlertMonitorRuleRepo = (*AlertMonitorRuleRepository)(nil)
