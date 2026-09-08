package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlertCurHisRepository struct {
	db *gorm.DB
}

func NewAlertCurHisRepository(db *gorm.DB) AlertCurHisRepo {
	return &AlertCurHisRepository{db: db}
}

func (r *AlertCurHisRepository) UpsertCurEvent(ctx context.Context, row *model.AlertCurEvent) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fingerprint"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"alertname", "severity", "status", "source", "receiver", "cluster",
			"project_id", "datasource_id", "group_key", "labels_json", "annotations_json",
			"summary", "value", "updated_at",
		}),
	}).Create(row).Error
}

func (r *AlertCurHisRepository) ResolveCurEvent(ctx context.Context, fingerprint string, resolvedAt time.Time) error {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur model.AlertCurEvent
		if err := tx.Where("fingerprint = ?", fp).First(&cur).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		his := model.AlertHisEvent{
			Fingerprint:     cur.Fingerprint,
			Alertname:       cur.Alertname,
			Severity:        cur.Severity,
			Status:          "resolved",
			Source:          cur.Source,
			Receiver:        cur.Receiver,
			Cluster:         cur.Cluster,
			ProjectID:       cur.ProjectID,
			DatasourceID:    cur.DatasourceID,
			GroupKey:        cur.GroupKey,
			LabelsJSON:      cur.LabelsJSON,
			AnnotationsJSON: cur.AnnotationsJSON,
			Summary:         cur.Summary,
			Value:           cur.Value,
			StartsAt:        cur.StartsAt,
			ResolvedAt:      resolvedAt,
		}
		if err := tx.Create(&his).Error; err != nil {
			return err
		}
		return tx.Where("fingerprint = ?", fp).Delete(&model.AlertCurEvent{}).Error
	})
}

func (r *AlertCurHisRepository) curListQuery(ctx context.Context, f AlertCurEventListFilter) *gorm.DB {
	tx := r.db.WithContext(ctx).Model(&model.AlertCurEvent{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if f.DatasourceID > 0 {
		tx = tx.Where("datasource_id = ?", f.DatasourceID)
	}
	if sev := strings.TrimSpace(f.Severity); sev != "" {
		tx = tx.Where("severity = ?", sev)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where(
			"alertname LIKE ? OR fingerprint LIKE ? OR summary LIKE ? OR cluster LIKE ?",
			like, like, like, like,
		)
	}
	return tx
}

func (r *AlertCurHisRepository) ListCurEvents(ctx context.Context, f AlertCurEventListFilter, offset, limit int) ([]model.AlertCurEvent, int64, error) {
	tx := r.curListQuery(ctx, f)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertCurEvent
	err := tx.Order("updated_at desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertCurHisRepository) hisListQuery(ctx context.Context, f AlertHisEventListFilter) *gorm.DB {
	tx := r.db.WithContext(ctx).Model(&model.AlertHisEvent{})
	if f.ProjectID > 0 {
		tx = tx.Where("project_id = ?", f.ProjectID)
	}
	if f.DatasourceID > 0 {
		tx = tx.Where("datasource_id = ?", f.DatasourceID)
	}
	if sev := strings.TrimSpace(f.Severity); sev != "" {
		tx = tx.Where("severity = ?", sev)
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where(
			"alertname LIKE ? OR fingerprint LIKE ? OR summary LIKE ? OR cluster LIKE ?",
			like, like, like, like,
		)
	}
	return tx
}

func (r *AlertCurHisRepository) ListHisEvents(ctx context.Context, f AlertHisEventListFilter, offset, limit int) ([]model.AlertHisEvent, int64, error) {
	tx := r.hisListQuery(ctx, f)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertHisEvent
	err := tx.Order("resolved_at desc").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertCurHisRepository) GetCurByFingerprint(ctx context.Context, fingerprint string) (*model.AlertCurEvent, error) {
	var row model.AlertCurEvent
	if err := r.db.WithContext(ctx).Where("fingerprint = ?", fingerprint).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertCurHisRepository) GetLatestHisByFingerprint(ctx context.Context, fingerprint string) (*model.AlertHisEvent, error) {
	var row model.AlertHisEvent
	if err := r.db.WithContext(ctx).
		Where("fingerprint = ?", fingerprint).
		Order("id DESC").
		First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *AlertCurHisRepository) CountCurByFingerprint(ctx context.Context, fingerprint string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.AlertCurEvent{}).
		Where("fingerprint = ?", fingerprint).
		Count(&n).Error
	return n, err
}

func (r *AlertCurHisRepository) Transaction(ctx context.Context, fn func(AlertCurHisRepo) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&AlertCurHisRepository{db: tx})
	})
}

var _ AlertCurHisRepo = (*AlertCurHisRepository)(nil)
