package repository

import (
	"context"

	"yunshu/internal/model"
)

// LogDropRuleRepo is implemented by *LogDropRuleRepository.
type LogDropRuleRepo interface {
	ListByProject(ctx context.Context, projectID uint) ([]model.LogDropRule, error)
	ListEnabledByProject(ctx context.Context, projectID uint) ([]model.LogDropRule, error)
	Create(ctx context.Context, row *model.LogDropRule) error
	GetByIDInProject(ctx context.Context, projectID, ruleID uint) (*model.LogDropRule, error)
	Save(ctx context.Context, row *model.LogDropRule) error
	DeleteByIDInProject(ctx context.Context, projectID, ruleID uint) (rowsAffected int64, err error)
}

var _ LogDropRuleRepo = (*LogDropRuleRepository)(nil)
