package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertAckRepository struct {
	db *gorm.DB
}

func NewAlertAckRepository(db *gorm.DB) AlertAckRepo {
	return &AlertAckRepository{db: db}
}

func (r *AlertAckRepository) Create(ctx context.Context, row *model.AlertAck) error {
	return r.db.WithContext(ctx).Create(row).Error
}

// ReplaceActive 在同一事务内结束旧认领并写入新认领，保证同指纹仅一条有效记录。
func (r *AlertAckRepository) ReplaceActive(ctx context.Context, row *model.AlertAck, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AlertAck{}).
			Where("fingerprint = ? AND expires_at > ?", row.Fingerprint, now).
			Update("expires_at", now).Error; err != nil {
			return err
		}
		return tx.Create(row).Error
	})
}

func (r *AlertAckRepository) ExpireActive(ctx context.Context, fingerprint string, now time.Time) error {
	return r.db.WithContext(ctx).Model(&model.AlertAck{}).
		Where("fingerprint = ? AND expires_at > ?", fingerprint, now).
		Update("expires_at", now).Error
}

func (r *AlertAckRepository) GetActive(ctx context.Context, fingerprint string, now time.Time) (*model.AlertAck, error) {
	var row model.AlertAck
	err := r.db.WithContext(ctx).
		Where("fingerprint = ? AND expires_at > ?", fingerprint, now).
		Order("expires_at desc").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertAckRepository) ListActiveByFingerprints(ctx context.Context, fingerprints []string, now time.Time) ([]model.AlertAck, error) {
	if len(fingerprints) == 0 {
		return nil, nil
	}
	var list []model.AlertAck
	err := r.db.WithContext(ctx).
		Where("fingerprint IN ? AND expires_at > ?", fingerprints, now).
		Order("expires_at desc").
		Find(&list).Error
	return list, err
}

var _ AlertAckRepo = (*AlertAckRepository)(nil)
