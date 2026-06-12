package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertMaintenanceWindowRepository struct {
	db *gorm.DB
}

func NewAlertMaintenanceWindowRepository(db *gorm.DB) AlertMaintenanceWindowRepo {
	return &AlertMaintenanceWindowRepository{db: db}
}

func (r *AlertMaintenanceWindowRepository) ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertMaintenanceWindow, error) {
	var list []model.AlertMaintenanceWindow
	err := r.db.WithContext(ctx).Model(&model.AlertMaintenanceWindow{}).
		Where("enabled = ? AND starts_at <= ? AND ends_at >= ?", true, at, at).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (r *AlertMaintenanceWindowRepository) ListPaged(ctx context.Context, projectID uint, keyword string, offset, limit int) ([]model.AlertMaintenanceWindow, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertMaintenanceWindow{})
	if projectID > 0 {
		tx = tx.Where("project_id = ?", projectID)
	}
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR comment LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertMaintenanceWindow
	err := tx.Order("starts_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertMaintenanceWindowRepository) Create(ctx context.Context, row *model.AlertMaintenanceWindow) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertMaintenanceWindowRepository) GetByID(ctx context.Context, id uint) (*model.AlertMaintenanceWindow, error) {
	var row model.AlertMaintenanceWindow
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertMaintenanceWindowRepository) Save(ctx context.Context, row *model.AlertMaintenanceWindow) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertMaintenanceWindowRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AlertMaintenanceWindow{}, id).Error
}

var _ AlertMaintenanceWindowRepo = (*AlertMaintenanceWindowRepository)(nil)
