package repository

import (
	"context"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertProgressNoteRepository struct {
	db *gorm.DB
}

func NewAlertProgressNoteRepository(db *gorm.DB) AlertProgressNoteRepo {
	return &AlertProgressNoteRepository{db: db}
}

func (r *AlertProgressNoteRepository) Create(ctx context.Context, row *model.AlertProgressNote) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *AlertProgressNoteRepository) ListByFingerprint(ctx context.Context, fingerprint string, limit int) ([]model.AlertProgressNote, error) {
	var list []model.AlertProgressNote
	tx := r.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).Order("id ASC")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err := tx.Find(&list).Error
	return list, err
}

func (r *AlertProgressNoteRepository) ListLatestByFingerprints(ctx context.Context, fingerprints []string) ([]model.AlertProgressNote, error) {
	if len(fingerprints) == 0 {
		return nil, nil
	}
	var list []model.AlertProgressNote
	err := r.db.WithContext(ctx).
		Where("fingerprint IN ?", fingerprints).
		Order("id DESC").
		Find(&list).Error
	return list, err
}

var _ AlertProgressNoteRepo = (*AlertProgressNoteRepository)(nil)
