package repository

import (
	"context"

	"yunshu/internal/model"
)

// CloudExpiryRuleRepo persists cloud expiry evaluation rules.
type CloudExpiryRuleRepo interface {
	List(ctx context.Context, keyword, provider string, projectID uint, offset, limit int) ([]model.CloudExpiryRule, int64, error)
	Create(ctx context.Context, row *model.CloudExpiryRule) error
	GetByID(ctx context.Context, id uint) (*model.CloudExpiryRule, error)
	Save(ctx context.Context, row *model.CloudExpiryRule) error
	Delete(ctx context.Context, id uint) error
	ListEnabled(ctx context.Context) ([]model.CloudExpiryRule, error)
}
