package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertReceiverGroupListFilter filters receiver group list queries.
type AlertReceiverGroupListFilter struct {
	ProjectID uint
	Keyword   string
	Enabled   *bool
}

// AlertReceiverGroupRepo is implemented by *AlertReceiverGroupRepository.
type AlertReceiverGroupRepo interface {
	List(ctx context.Context, f AlertReceiverGroupListFilter, offset, limit int) ([]model.AlertReceiverGroup, int64, error)
	ListEnabled(ctx context.Context) ([]model.AlertReceiverGroup, error)
	GetByID(ctx context.Context, id uint) (*model.AlertReceiverGroup, error)
	Create(ctx context.Context, row *model.AlertReceiverGroup) error
	Save(ctx context.Context, row *model.AlertReceiverGroup) error
	Delete(ctx context.Context, id uint) (rowsAffected int64, err error)
}
