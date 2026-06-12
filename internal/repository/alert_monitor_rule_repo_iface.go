package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertMonitorRuleListFilter filters monitor rule list queries.
type AlertMonitorRuleListFilter struct {
	DatasourceID *uint
	ProjectID    *uint
	Keyword      string
	Enabled      *bool
}

// EvalMonitorRule is a monitor rule with its project id for evaluator ticks.
type EvalMonitorRule struct {
	model.AlertMonitorRule
	ProjectID uint `gorm:"column:project_id"`
}

// AlertMonitorRuleRepo is implemented by *AlertMonitorRuleRepository.
type AlertMonitorRuleRepo interface {
	List(ctx context.Context, f AlertMonitorRuleListFilter, offset, limit int) ([]model.AlertMonitorRule, int64, error)
	ListEnabled(ctx context.Context) ([]model.AlertMonitorRule, error)
	ListEnabledWithProject(ctx context.Context) ([]EvalMonitorRule, error)
	GetByID(ctx context.Context, id uint) (*model.AlertMonitorRule, error)
	Create(ctx context.Context, row *model.AlertMonitorRule) error
	Save(ctx context.Context, row *model.AlertMonitorRule) error
	DeleteCascade(ctx context.Context, id uint) error
}
