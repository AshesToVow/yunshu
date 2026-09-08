package repository

import (
	"context"

	"yunshu/internal/model"
)

// PromqlSavedQueryRepo is implemented by *PromqlSavedQueryRepository.
type PromqlSavedQueryRepo interface {
	ListByUser(ctx context.Context, userID uint) ([]model.PlatformSavedQuery, error)
	Create(ctx context.Context, row *model.PlatformSavedQuery) error
	DeleteByUser(ctx context.Context, userID, id uint) (rowsAffected int64, err error)
}
