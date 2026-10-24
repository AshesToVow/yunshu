package repository

import (
	"context"

	"yunshu/internal/model"
)

// KafkamgmtRepo is implemented by *KafkamgmtRepository.
type KafkamgmtRepo interface {
	ListConnections(ctx context.Context) ([]model.KafkamgmtConnection, error)
	GetConnection(ctx context.Context, id uint) (*model.KafkamgmtConnection, error)
	GetConnectionOwner(ctx context.Context, id uint) (*model.KafkamgmtConnection, error)
	FindDictImportConnection(ctx context.Context, remarkMarker, name string) (*model.KafkamgmtConnection, error)
	CountDefaultConnections(ctx context.Context) (int64, error)
	GetDefaultConnection(ctx context.Context) (*model.KafkamgmtConnection, error)
	CreateConnectionClearDefaults(ctx context.Context, row *model.KafkamgmtConnection) error
	UpdateConnectionClearDefaults(ctx context.Context, id uint, row *model.KafkamgmtConnection) error
	DeleteConnection(ctx context.Context, id uint) (int64, error)
}

var _ KafkamgmtRepo = (*KafkamgmtRepository)(nil)
