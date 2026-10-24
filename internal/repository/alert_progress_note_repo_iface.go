package repository

import (
	"context"

	"yunshu/internal/model"
)

// AlertProgressNoteRepo is implemented by *AlertProgressNoteRepository.
type AlertProgressNoteRepo interface {
	Create(ctx context.Context, row *model.AlertProgressNote) error
	ListByFingerprint(ctx context.Context, fingerprint string, limit int) ([]model.AlertProgressNote, error)
	// ListLatestByFingerprints returns notes ordered by id DESC; callers dedupe per fingerprint.
	ListLatestByFingerprints(ctx context.Context, fingerprints []string) ([]model.AlertProgressNote, error)
}
