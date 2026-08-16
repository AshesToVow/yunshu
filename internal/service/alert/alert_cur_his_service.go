package alert

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlertCurEventListQuery struct {
	ProjectID    uint   `form:"project_id"`
	DatasourceID uint   `form:"datasource_id"`
	Severity     string `form:"severity"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type AlertHisEventListQuery struct {
	ProjectID    uint   `form:"project_id"`
	DatasourceID uint   `form:"datasource_id"`
	Severity     string `form:"severity"`
	Keyword      string `form:"keyword"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

// UpsertCurEvent 写入/更新当前告警（屏蔽后不调用）。
func (s *AlertService) UpsertCurEvent(ctx context.Context, row *model.AlertCurEvent) error {
	if s == nil || s.db == nil || row == nil {
		return nil
	}
	fp := strings.TrimSpace(row.Fingerprint)
	if fp == "" {
		return nil
	}
	row.Fingerprint = fp
	if strings.TrimSpace(row.Status) == "" {
		row.Status = "firing"
	}
	now := time.Now().UTC()
	if row.StartsAt.IsZero() {
		row.StartsAt = now
	}
	row.UpdatedAt = now
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fingerprint"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"alertname", "severity", "status", "source", "receiver", "cluster",
			"project_id", "datasource_id", "group_key", "labels_json", "annotations_json",
			"summary", "value", "updated_at",
		}),
	}).Create(row).Error
	return bizerrors.Pass(ctx, "alert.cur", "UpsertCurEvent", err)
}

// ResolveCurEvent 将当前告警迁入历史并删除当前行。
func (s *AlertService) ResolveCurEvent(ctx context.Context, fingerprint string, resolvedAt time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil
	}
	if resolvedAt.IsZero() {
		resolvedAt = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

func (s *AlertService) ListCurEvents(ctx context.Context, q AlertCurEventListQuery) ([]model.AlertCurEvent, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AlertCurEvent{})
	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if q.DatasourceID > 0 {
		db = db.Where("datasource_id = ?", q.DatasourceID)
	}
	if sev := strings.TrimSpace(q.Severity); sev != "" {
		db = db.Where("severity = ?", sev)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("alertname LIKE ? OR fingerprint LIKE ? OR summary LIKE ? OR cluster LIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.cur", "ListCurEvents", err)
	}
	var list []model.AlertCurEvent
	if err := db.Order("updated_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.cur", "ListCurEvents", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertService) ListHisEvents(ctx context.Context, q AlertHisEventListQuery) ([]model.AlertHisEvent, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	db := s.db.WithContext(ctx).Model(&model.AlertHisEvent{})
	if q.ProjectID > 0 {
		db = db.Where("project_id = ?", q.ProjectID)
	}
	if q.DatasourceID > 0 {
		db = db.Where("datasource_id = ?", q.DatasourceID)
	}
	if sev := strings.TrimSpace(q.Severity); sev != "" {
		db = db.Where("severity = ?", sev)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		db = db.Where("alertname LIKE ? OR fingerprint LIKE ? OR summary LIKE ? OR cluster LIKE ?", like, like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.his", "ListHisEvents", err)
	}
	var list []model.AlertHisEvent
	if err := db.Order("resolved_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.his", "ListHisEvents", err)
	}
	return list, total, page, pageSize, nil
}

func buildCurEventFromIngress(
	source, receiver, title, severity, status, envLabel, groupKey string,
	dsID uint, labels, annotations map[string]string,
	fp string, startsAt time.Time, value string,
) *model.AlertCurEvent {
	lb, _ := json.Marshal(labels)
	ab, _ := json.Marshal(annotations)
	projectID := uint(0)
	if v := strings.TrimSpace(labels["project_id"]); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			projectID = uint(n)
		}
	}
	summary := strings.TrimSpace(annotations["summary"])
	if summary == "" {
		summary = title
	}
	if startsAt.IsZero() {
		startsAt = time.Now().UTC()
	}
	return &model.AlertCurEvent{
		Fingerprint:     fp,
		Alertname:       title,
		Severity:        severity,
		Status:          status,
		Source:          source,
		Receiver:        receiver,
		Cluster:         envLabel,
		ProjectID:       projectID,
		DatasourceID:    dsID,
		GroupKey:        groupKey,
		LabelsJSON:      string(lb),
		AnnotationsJSON: string(ab),
		Summary:         truncateText(summary, 512),
		Value:           truncateText(value, 128),
		StartsAt:        startsAt,
	}
}
