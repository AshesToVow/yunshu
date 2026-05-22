package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertReceiverGroupRepository struct {
	db *gorm.DB
}

func NewAlertReceiverGroupRepository(db *gorm.DB) AlertReceiverGroupRepo {
	return &AlertReceiverGroupRepository{db: db}
}

func (r *AlertReceiverGroupRepository) List(ctx context.Context, f AlertReceiverGroupListFilter, offset, limit int) ([]model.AlertReceiverGroup, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertReceiverGroup{})
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
	var list []model.AlertReceiverGroup
	err := tx.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertReceiverGroupRepository) ListEnabled(ctx context.Context) ([]model.AlertReceiverGroup, error) {
	var list []model.AlertReceiverGroup
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&list).Error
	return list, err
}

func (r *AlertReceiverGroupRepository) GetByID(ctx context.Context, id uint) (*model.AlertReceiverGroup, error) {
	var row model.AlertReceiverGroup
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertReceiverGroupRepository) Create(ctx context.Context, row *model.AlertReceiverGroup) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertReceiverGroupRepository) Save(ctx context.Context, row *model.AlertReceiverGroup) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertReceiverGroupRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertReceiverGroup{}, id)
	return res.RowsAffected, res.Error
}

var _ AlertReceiverGroupRepo = (*AlertReceiverGroupRepository)(nil)
