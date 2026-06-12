// Package alert defines alert domain service interfaces and DTOs.
package alert

import (
	"context"
	"time"
)

type AlertStateService interface {
	GetOrCreateState(ctx context.Context, fingerprint string) (*AggregateState, error)
	UpdateStatus(ctx context.Context, fingerprint string, newStatus string) error
	TouchFingerprint(ctx context.Context, fingerprint, status string) (count int64, err error)
	ClearFingerprint(ctx context.Context, fingerprint string) error
	IsDuplicate(ctx context.Context, fingerprint string) (bool, error)
	CleanupExpiredStates(ctx context.Context, ttl time.Duration) (int64, error)
	MarkResolvedNotificationSent(ctx context.Context, fingerprint string) (first bool, err error)
	ClearResolvedNotificationSent(ctx context.Context, fingerprint string) error
	MarkFiringDelivered(ctx context.Context, fingerprint string) error
	WasFiringDelivered(ctx context.Context, fingerprint string) bool
	ClearFiringDelivered(ctx context.Context, fingerprint string) error
	ClearCurrentMetric(ctx context.Context, fingerprint string) error
}

type AggregateState struct {
	Fingerprint  string     `json:"fingerprint"`
	GroupKey     string     `json:"group_key"`
	Status       string     `json:"status"`
	FirstFiredAt time.Time  `json:"first_fired_at"`
	LastFiredAt  time.Time  `json:"last_fired_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	FireCount    int64      `json:"fire_count"`
	NotifyCount  int64      `json:"notify_count"`
	LabelsDigest string     `json:"labels_digest"`
}
