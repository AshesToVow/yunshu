package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertConsulEndpointListFilter filters Consul endpoint list queries.
type AlertConsulEndpointListFilter struct {
	ProjectID uint
	Keyword   string
}

// AlertMonitorObjectListFilter filters monitor object list queries.
type AlertMonitorObjectListFilter struct {
	ProjectID    uint
	EndpointID   uint
	ExporterRole string
	Keyword      string
}

// AlertConsulRepo is implemented by *AlertConsulRepository.
type AlertConsulRepo interface {
	ListEndpoints(ctx context.Context, f AlertConsulEndpointListFilter, offset, limit int) ([]model.AlertConsulEndpoint, int64, error)
	CreateEndpoint(ctx context.Context, row *model.AlertConsulEndpoint) error
	GetEndpointByID(ctx context.Context, id uint) (*model.AlertConsulEndpoint, error)
	SaveEndpoint(ctx context.Context, row *model.AlertConsulEndpoint) error
	DeleteEndpointWithObjects(ctx context.Context, id uint) error

	ListObjects(ctx context.Context, f AlertMonitorObjectListFilter, offset, limit int) ([]model.AlertMonitorObject, int64, error)

	UpdateEndpointMeta(ctx context.Context, id uint, fields map[string]any) error
	// UpsertMonitorObject creates or updates (incl. soft-deleted revive) by endpoint_id+service_name+service_id.
	UpsertMonitorObject(ctx context.Context, row *model.AlertMonitorObject) error
	ListObjectsByEndpoint(ctx context.Context, endpointID uint) ([]model.AlertMonitorObject, error)
	DeleteObject(ctx context.Context, row *model.AlertMonitorObject) error

	Transaction(ctx context.Context, fn func(AlertConsulRepo) error) error
}
