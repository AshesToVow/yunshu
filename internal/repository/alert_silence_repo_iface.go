package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// AlertSilenceRepo is implemented by *AlertSilenceRepository.
type AlertSilenceRepo interface {
	DisableExpired(ctx context.Context, now time.Time) error
	ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertSilence, error)
	ListEnabledUnexpiredByProject(ctx context.Context, projectID uint, now time.Time) ([]model.AlertSilence, error)
	Create(ctx context.Context, row *model.AlertSilence) error
	GetByID(ctx context.Context, id uint) (*model.AlertSilence, error)
	Save(ctx context.Context, row *model.AlertSilence) error
	Delete(ctx context.Context, id uint) error
	ListPaged(ctx context.Context, projectID uint, keyword string, offset, limit int) ([]model.AlertSilence, int64, error)
}
