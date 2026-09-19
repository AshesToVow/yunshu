package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// AlertAckRepo is implemented by *AlertAckRepository.
type AlertAckRepo interface {
	Create(ctx context.Context, row *model.AlertAck) error
	ExpireActive(ctx context.Context, fingerprint string, now time.Time) error
	// GetActive returns gorm.ErrRecordNotFound if none is active.
	GetActive(ctx context.Context, fingerprint string, now time.Time) (*model.AlertAck, error)
	ListActiveByFingerprints(ctx context.Context, fingerprints []string, now time.Time) ([]model.AlertAck, error)
}
