package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertRuleAssigneeRepo is implemented by *AlertRuleAssigneeRepository.
type AlertRuleAssigneeRepo interface {
	ListByRule(ctx context.Context, ruleID uint) ([]model.AlertRuleAssignee, error)
	GetPrimaryByRule(ctx context.Context, ruleID uint) (*model.AlertRuleAssignee, error)
	Create(ctx context.Context, row *model.AlertRuleAssignee) error
	Save(ctx context.Context, row *model.AlertRuleAssignee) error
	Delete(ctx context.Context, id uint) (int64, error)
	ListAll(ctx context.Context) ([]model.AlertRuleAssignee, error)
	UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error
	ListProjectMemberUserIDsByDepartments(ctx context.Context, projectID uint, deptIDs []uint) ([]uint, error)
}
