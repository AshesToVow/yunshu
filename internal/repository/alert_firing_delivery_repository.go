package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlertFiringDeliveryRepository struct {
	db *gorm.DB
}

func NewAlertFiringDeliveryRepository(db *gorm.DB) AlertFiringDeliveryRepo {
	return &AlertFiringDeliveryRepository{db: db}
}

func (r *AlertFiringDeliveryRepository) Mark(ctx context.Context, fingerprint string) error {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	row := model.AlertFiringDelivery{Fingerprint: fp, UpdatedAt: time.Now().UTC()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fingerprint"}},
		DoUpdates: clause.AssignmentColumns([]string{"updated_at"}),
	}).Create(&row).Error
}

func (r *AlertFiringDeliveryRepository) Exists(ctx context.Context, fingerprint string) (bool, error) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertFiringDelivery{}).Where("fingerprint = ?", fp).Count(&n).Error
	return n > 0, err
}

func (r *AlertFiringDeliveryRepository) Delete(ctx context.Context, fingerprint string) error {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	return r.db.WithContext(ctx).Where("fingerprint = ?", fp).Delete(&model.AlertFiringDelivery{}).Error
}

var _ AlertFiringDeliveryRepo = (*AlertFiringDeliveryRepository)(nil)
