package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertRuleChangeRepo is implemented by *AlertRuleChangeRepository.
type AlertRuleChangeRepo interface {
	Create(ctx context.Context, row *model.AlertMonitorRuleChangeRequest) error
	ListPending(ctx context.Context) ([]model.AlertMonitorRuleChangeRequest, error)
	GetByID(ctx context.Context, id uint) (*model.AlertMonitorRuleChangeRequest, error)
	UpdateFields(ctx context.Context, id uint, fields map[string]any) error
	UpdatePending(ctx context.Context, id uint, fields map[string]any) (rowsAffected int64, err error)
}
