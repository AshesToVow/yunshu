package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertInhibitionListFilter filters inhibition rules for list APIs.
type AlertInhibitionListFilter struct {
	ProjectID uint
	Keyword   string
	Enabled   *bool
}

// AlertInhibitionRuleRepo is implemented by *AlertInhibitionRuleRepository.
type AlertInhibitionRuleRepo interface {
	ListEnabled(ctx context.Context) ([]model.AlertInhibitionRule, error)
	ListFiltered(ctx context.Context, f AlertInhibitionListFilter, offset, limit int) ([]model.AlertInhibitionRule, int64, error)
	GetByID(ctx context.Context, id uint) (*model.AlertInhibitionRule, error)
	Create(ctx context.Context, rule *model.AlertInhibitionRule) error
	Save(ctx context.Context, rule *model.AlertInhibitionRule) error
	Delete(ctx context.Context, id uint) (rowsAffected int64, err error)
	CreateEvent(ctx context.Context, event *model.AlertInhibitionEvent) error
}
