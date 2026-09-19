package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

const logSavedQueryKind = "log"

type LogSavedQueryRepository struct {
	db *gorm.DB
}

func NewLogSavedQueryRepository(db *gorm.DB) LogSavedQueryRepo {
	return &LogSavedQueryRepository{db: db}
}

func (r *LogSavedQueryRepository) List(ctx context.Context, userID, projectID uint) ([]model.PlatformSavedQuery, error) {
	q := r.db.WithContext(ctx).Where("user_id = ? AND kind = ?", userID, logSavedQueryKind)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	var list []model.PlatformSavedQuery
	if err := q.Order("updated_at DESC").Limit(100).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *LogSavedQueryRepository) Create(ctx context.Context, row *model.PlatformSavedQuery) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *LogSavedQueryRepository) DeleteByUser(ctx context.Context, userID, id uint) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND kind = ?", id, userID, logSavedQueryKind).
		Delete(&model.PlatformSavedQuery{})
	return res.RowsAffected, res.Error
}
