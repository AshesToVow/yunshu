package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LogRetentionRepository struct{ db *gorm.DB }

func NewLogRetentionRepository(db *gorm.DB) LogRetentionRepo {
	return &LogRetentionRepository{db: db}
}

func (r *LogRetentionRepository) GetByScope(ctx context.Context, projectID, serverID uint) (*model.LogRetentionPolicy, error) {
	var it model.LogRetentionPolicy
	err := r.db.WithContext(ctx).Where("project_id = ? AND server_id = ?", projectID, serverID).First(&it).Error
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LogRetentionRepository) List(ctx context.Context) ([]model.LogRetentionPolicy, error) {
	var list []model.LogRetentionPolicy
	err := r.db.WithContext(ctx).Order("project_id ASC, server_id ASC").Find(&list).Error
	return list, err
}

func (r *LogRetentionRepository) Save(ctx context.Context, it *model.LogRetentionPolicy) error {
	return r.db.WithContext(ctx).Save(it).Error
}

func (r *LogRetentionRepository) DeleteByScope(ctx context.Context, projectID, serverID uint) error {
	return r.db.WithContext(ctx).Where("project_id = ? AND server_id = ?", projectID, serverID).Delete(&model.LogRetentionPolicy{}).Error
}
