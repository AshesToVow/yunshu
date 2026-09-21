package repository

import (
	"context"

	"yunshu/internal/model"
)

// LogSavedQueryRepo is implemented by *LogSavedQueryRepository (platform_saved_queries kind=log).
type LogSavedQueryRepo interface {
	List(ctx context.Context, userID, projectID uint) ([]model.PlatformSavedQuery, error)
	Create(ctx context.Context, row *model.PlatformSavedQuery) error
	DeleteByUser(ctx context.Context, userID, id uint) (rowsAffected int64, err error)
}

var _ LogSavedQueryRepo = (*LogSavedQueryRepository)(nil)
