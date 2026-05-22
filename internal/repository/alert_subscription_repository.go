package repository

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertSubscriptionRepository struct {
	db *gorm.DB
}

func NewAlertSubscriptionRepository(db *gorm.DB) AlertSubscriptionRepo {
	return &AlertSubscriptionRepository{db: db}
}

func (r *AlertSubscriptionRepository) ListFiltered(ctx context.Context, f AlertSubscriptionListFilter, offset, limit int) ([]model.AlertSubscriptionNode, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if f.ParentID != nil {
		if *f.ParentID == 0 {
			tx = tx.Where("parent_id IS NULL")
		} else {
			tx = tx.Where("parent_id = ?", *f.ParentID)
		}
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if f.Enabled != nil {
		tx = tx.Where("enabled = ?", *f.Enabled)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertSubscriptionNode
	err := tx.Order("level ASC, id ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertSubscriptionRepository) ListByProject(ctx context.Context, projectID uint) ([]model.AlertSubscriptionNode, error) {
	var list []model.AlertSubscriptionNode
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("path ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *AlertSubscriptionRepository) ListEnabled(ctx context.Context) ([]model.AlertSubscriptionNode, error) {
	var list []model.AlertSubscriptionNode
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("path ASC, level ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *AlertSubscriptionRepository) GetByID(ctx context.Context, id uint) (*model.AlertSubscriptionNode, error) {
	var node model.AlertSubscriptionNode
	if err := r.db.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *AlertSubscriptionRepository) Create(ctx context.Context, node *model.AlertSubscriptionNode) error {
	return r.db.WithContext(ctx).Create(node).Error
}

func (r *AlertSubscriptionRepository) Save(ctx context.Context, node *model.AlertSubscriptionNode) error {
	return r.db.WithContext(ctx).Save(node).Error
}

func (r *AlertSubscriptionRepository) Delete(ctx context.Context, id uint) (int64, error) {
	res := r.db.WithContext(ctx).Delete(&model.AlertSubscriptionNode{}, id)
	return res.RowsAffected, res.Error
}

func (r *AlertSubscriptionRepository) CountChildren(ctx context.Context, parentID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).Where("parent_id = ?", parentID).Count(&n).Error
	return n, err
}

func (r *AlertSubscriptionRepository) ListChildren(ctx context.Context, parentID uint) ([]model.AlertSubscriptionNode, error) {
	var list []model.AlertSubscriptionNode
	err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&list).Error
	return list, err
}

func (r *AlertSubscriptionRepository) UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AlertSubscriptionRepository) Transaction(ctx context.Context, fn func(AlertSubscriptionRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&AlertSubscriptionRepository{db: tx})
	})
}

func (r *AlertSubscriptionRepository) WithTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

var _ AlertSubscriptionRepo = (*AlertSubscriptionRepository)(nil)
