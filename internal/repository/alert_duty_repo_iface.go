package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// AlertDutyListFilter filters duty block list queries.
type AlertDutyListFilter struct {
	MonitorRuleID *uint
	ProjectID     *uint
}

// AlertDutyRepo is implemented by *AlertDutyRepository.
type AlertDutyRepo interface {
	List(ctx context.Context, f AlertDutyListFilter, offset, limit int) ([]model.AlertDutyBlock, int64, error)
	GetByID(ctx context.Context, id uint) (*model.AlertDutyBlock, error)
	Create(ctx context.Context, row *model.AlertDutyBlock) error
	Save(ctx context.Context, row *model.AlertDutyBlock) error
	Delete(ctx context.Context, id uint) (int64, error)
	HasActiveAtRule(ctx context.Context, monitorRuleID uint, t time.Time) (bool, error)
	ListActiveAtRule(ctx context.Context, monitorRuleID uint, t time.Time) ([]model.AlertDutyBlock, error)
}
