package repository

import "context"

// AlertFiringDeliveryRepo tracks successful firing delivery per fingerprint.
type AlertFiringDeliveryRepo interface {
	Mark(ctx context.Context, fingerprint string) error
	Exists(ctx context.Context, fingerprint string) (bool, error)
	Delete(ctx context.Context, fingerprint string) error
}
