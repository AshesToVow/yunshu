package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LoggieAgentRepository struct{ db *gorm.DB }

func NewLoggieAgentRepository(db *gorm.DB) LoggieAgentRepo {
	return &LoggieAgentRepository{db: db}
}

func (r *LoggieAgentRepository) GetByToken(ctx context.Context, token string) (*model.LoggieAgent, error) {
	var it model.LoggieAgent
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&it).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LoggieAgentRepository) GetByProjectAndServer(ctx context.Context, projectID, serverID uint) (*model.LoggieAgent, error) {
	var it model.LoggieAgent
	err := r.db.WithContext(ctx).Where("project_id = ? AND server_id = ?", projectID, serverID).First(&it).Error
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *LoggieAgentRepository) ListByProject(ctx context.Context, projectID uint) ([]model.LoggieAgent, error) {
	var list []model.LoggieAgent
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("server_id ASC").Find(&list).Error
	return list, err
}

func (r *LoggieAgentRepository) Save(ctx context.Context, it *model.LoggieAgent) error {
	return r.db.WithContext(ctx).Save(it).Error
}

func (r *LoggieAgentRepository) TouchSeen(ctx context.Context, id uint, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.LoggieAgent{}).Where("id = ?", id).Updates(map[string]any{
		"last_seen_at": at,
		"updated_at":   at,
	}).Error
}
