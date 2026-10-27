package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertSilenceRepository struct {
	db *gorm.DB
}

func NewAlertSilenceRepository(db *gorm.DB) AlertSilenceRepo {
	return &AlertSilenceRepository{db: db}
}

func (r *AlertSilenceRepository) DisableExpired(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Model(&model.AlertSilence{}).
		Where("enabled = ? AND ends_at < ?", true, now).
		Update("enabled", false).Error
}

func (r *AlertSilenceRepository) ListActiveAt(ctx context.Context, at time.Time) ([]model.AlertSilence, error) {
	_ = r.DisableExpired(ctx, at)
	var list []model.AlertSilence
	err := r.db.WithContext(ctx).Model(&model.AlertSilence{}).
		Where("enabled = ? AND starts_at <= ? AND ends_at >= ?", true, at, at).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

func (r *AlertSilenceRepository) ListEnabledUnexpiredByProject(ctx context.Context, projectID uint, now time.Time) ([]model.AlertSilence, error) {
	var list []model.AlertSilence
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND ends_at > ? AND project_id = ?", true, now, projectID).
		Find(&list).Error
	return list, err
}

func (r *AlertSilenceRepository) Create(ctx context.Context, row *model.AlertSilence) error {
	return r.db.WithContext(ctx).Create(row).Error
}

// CreateBatch 批量创建：同一事务内原子写入，避免部分成功导致调度状态只落一半。
func (r *AlertSilenceRepository) CreateBatch(ctx context.Context, rows []*model.AlertSilence) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Create(rows).Error
	})
}

func (r *AlertSilenceRepository) GetByID(ctx context.Context, id uint) (*model.AlertSilence, error) {
	var row model.AlertSilence
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertSilenceRepository) Save(ctx context.Context, row *model.AlertSilence) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *AlertSilenceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.AlertSilence{}, id).Error
}

func (r *AlertSilenceRepository) ListPaged(ctx context.Context, projectID uint, projectIDs []uint, keyword string, offset, limit int) ([]model.AlertSilence, int64, error) {
	tx := r.db.WithContext(ctx).Model(&model.AlertSilence{})
	if projectID > 0 {
		tx = tx.Where("project_id = ?", projectID)
	} else if len(projectIDs) > 0 {
		tx = tx.Where("project_id IN ?", projectIDs)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := likeContains(kw)
		tx = tx.Where(`(name LIKE ? ESCAPE '!' OR comment LIKE ? ESCAPE '!')`, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertSilence
	err := tx.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

var _ AlertSilenceRepo = (*AlertSilenceRepository)(nil)
