package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertDutyRotationListFilter filters duty rotation queries.
type AlertDutyRotationListFilter struct {
	MonitorRuleID *uint
	ProjectID     *uint
	Enabled       *bool
}

// AlertDutyRotationRepo is implemented by *AlertDutyRotationRepository.
type AlertDutyRotationRepo interface {
	List(ctx context.Context, f AlertDutyRotationListFilter, offset, limit int) ([]model.AlertDutyRotation, int64, error)
	GetByID(ctx context.Context, id uint) (*model.AlertDutyRotation, error)
	Create(ctx context.Context, row *model.AlertDutyRotation) error
	Save(ctx context.Context, row *model.AlertDutyRotation) error
	Delete(ctx context.Context, id uint) (int64, error)
	ListEnabled(ctx context.Context) ([]model.AlertDutyRotation, error)
	ListEnabledByRule(ctx context.Context, monitorRuleID uint) ([]model.AlertDutyRotation, error)
	CountEnabledByRuleExcept(ctx context.Context, monitorRuleID, exceptID uint) (int64, error)
}
