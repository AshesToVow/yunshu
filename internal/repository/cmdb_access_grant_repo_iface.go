package repository

import (
	"context"

	"yunshu/internal/model"
)

// ServerAccessGrantListParams filters server access grant listing.
type ServerAccessGrantListParams struct {
	ProjectID     uint
	PrincipalKind string
	PrincipalRef  string
	ServerID      uint
}

// ServerAccessGrantRepo is implemented by *ServerAccessGrantRepository.
type ServerAccessGrantRepo interface {
	Get(ctx context.Context, projectID, serverID uint, principalKind, principalRef string) (*model.ServerAccessGrant, error)
	ListVisibleServerIDs(ctx context.Context, projectID uint, principalKind, principalRef string) ([]uint, error)
	List(ctx context.Context, p ServerAccessGrantListParams) ([]model.ServerAccessGrant, error)
	Upsert(ctx context.Context, row *model.ServerAccessGrant) error
	Delete(ctx context.Context, projectID, grantID uint) (int64, error)
}

var _ ServerAccessGrantRepo = (*ServerAccessGrantRepository)(nil)
