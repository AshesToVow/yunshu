package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertSubscriptionListFilter filters subscription nodes for admin list APIs.
type AlertSubscriptionListFilter struct {
	ProjectID uint
	ParentID  *uint
	Keyword   string
	Enabled   *bool
}

// AlertSubscriptionRepo is implemented by *AlertSubscriptionRepository.
type AlertSubscriptionRepo interface {
	ListFiltered(ctx context.Context, f AlertSubscriptionListFilter, offset, limit int) ([]model.AlertSubscriptionNode, int64, error)
	ListByProject(ctx context.Context, projectID uint) ([]model.AlertSubscriptionNode, error)
	ListEnabled(ctx context.Context) ([]model.AlertSubscriptionNode, error)
	GetByID(ctx context.Context, id uint) (*model.AlertSubscriptionNode, error)
	Create(ctx context.Context, node *model.AlertSubscriptionNode) error
	Save(ctx context.Context, node *model.AlertSubscriptionNode) error
	Delete(ctx context.Context, id uint) (rowsAffected int64, err error)
	CountChildren(ctx context.Context, parentID uint) (int64, error)
	ListChildren(ctx context.Context, parentID uint) ([]model.AlertSubscriptionNode, error)
	UpdateFields(ctx context.Context, id uint, updates map[string]interface{}) error
}
