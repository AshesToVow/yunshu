package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type PromqlSavedQueryRepository struct {
	db *gorm.DB
}

func NewPromqlSavedQueryRepository(db *gorm.DB) PromqlSavedQueryRepo {
	return &PromqlSavedQueryRepository{db: db}
}

func (r *PromqlSavedQueryRepository) ListByUser(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error) {
	var list []model.PlatformSavedQuery
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *PromqlSavedQueryRepository) Create(ctx context.Context, row *model.PlatformSavedQuery) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *PromqlSavedQueryRepository) DeleteByUser(ctx context.Context, userID, id uint) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.PlatformSavedQuery{})
	return res.RowsAffected, res.Error
}

var _ PromqlSavedQueryRepo = (*PromqlSavedQueryRepository)(nil)
