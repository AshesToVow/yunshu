package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// AlertCurEventListFilter filters current alert list queries.
type AlertCurEventListFilter struct {
	ProjectID    uint
	DatasourceID uint
	Severity     string
	Keyword      string
}

// AlertHisEventListFilter filters historical alert list queries.
type AlertHisEventListFilter struct {
	ProjectID    uint
	DatasourceID uint
	Severity     string
	Keyword      string
}

// AlertCurHisRepo is implemented by *AlertCurHisRepository.
type AlertCurHisRepo interface {
	// UpsertCurEvent inserts or updates by fingerprint OnConflict.
	UpsertCurEvent(ctx context.Context, row *model.AlertCurEvent) error
	// ResolveCurEvent copies the current row to history and deletes it (transaction).
	ResolveCurEvent(ctx context.Context, fingerprint string, resolvedAt time.Time) error
	ListCurEvents(ctx context.Context, f AlertCurEventListFilter, offset, limit int) ([]model.AlertCurEvent, int64, error)
	ListHisEvents(ctx context.Context, f AlertHisEventListFilter, offset, limit int) ([]model.AlertHisEvent, int64, error)
	GetCurByFingerprint(ctx context.Context, fingerprint string) (*model.AlertCurEvent, error)
	GetLatestHisByFingerprint(ctx context.Context, fingerprint string) (*model.AlertHisEvent, error)
	CountCurByFingerprint(ctx context.Context, fingerprint string) (int64, error)
	Transaction(ctx context.Context, fn func(AlertCurHisRepo) error) error
}
