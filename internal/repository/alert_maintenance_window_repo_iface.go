package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

type AlertMaintenanceWindowRepo interface {
	ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertMaintenanceWindow, error)
	ListPaged(ctx context.Context, projectID uint, keyword string, offset, limit int) ([]model.AlertMaintenanceWindow, int64, error)
	Create(ctx context.Context, row *model.AlertMaintenanceWindow) error
	GetByID(ctx context.Context, id uint) (*model.AlertMaintenanceWindow, error)
	Save(ctx context.Context, row *model.AlertMaintenanceWindow) error
	Delete(ctx context.Context, id uint) error
}
