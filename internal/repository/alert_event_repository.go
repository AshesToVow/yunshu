package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type AlertEventRepository struct {
	db *gorm.DB
}

func NewAlertEventRepository(db *gorm.DB) AlertEventRepo {
	return &AlertEventRepository{db: db}
}

func (r *AlertEventRepository) Create(ctx context.Context, event *model.AlertEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *AlertEventRepository) GetByFingerprint(ctx context.Context, fingerprint string) (*model.AlertEvent, error) {
	var event model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("group_key = ? OR labels_digest = ?", fingerprint, fingerprint).
		Order("id DESC").
		First(&event).Error
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *AlertEventRepository) UpdateStatus(ctx context.Context, fingerprint, status string) error {
	res := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("group_key = ? OR labels_digest = ?", fingerprint, fingerprint).
		Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *AlertEventRepository) listQuery(ctx context.Context, f AlertEventListFilter) *gorm.DB {
	tx := r.db.WithContext(ctx).Model(&model.AlertEvent{})
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("title LIKE ? OR error_message LIKE ? OR channel_name LIKE ?", like, like, like)
	}
	if v := strings.TrimSpace(f.Cluster); v != "" {
		tx = tx.Where("cluster = ?", v)
	}
	if v := strings.TrimSpace(f.AlertIP); v != "" {
		like := "%" + v + "%"
		tx = tx.Where(
			"cluster = ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ?",
			v,
			"%\"instance\":\""+v+"\"%",
			"%\"pod_ip\":\""+v+"\"%",
			"%\"node\":\""+v+"\"%",
			"%\"ip\":\""+v+"\"%",
			like,
		)
	}
	if v := strings.ToLower(strings.TrimSpace(f.Status)); v != "" {
		tx = tx.Where("status = ?", v)
	}
	if v := strings.TrimSpace(f.MonitorPipeline); v != "" {
		tx = tx.Where("monitor_pipeline = ?", v)
	}
	if f.DatasourceID > 0 {
		tx = tx.Where("datasource_id = ?", f.DatasourceID)
	}
	if f.ProjectID > 0 {
		tx = applyAlertEventProjectFilter(tx, r.db, f.ProjectID)
	}
	if v := strings.TrimSpace(f.GroupKey); v != "" {
		tx = tx.Where("group_key = ?", v)
	}
	if v := strings.TrimSpace(f.Fingerprint); v != "" {
		like := "%\"fingerprint\":\"" + v + "\"%"
		tx = tx.Where(
			"fingerprint = ? OR group_key = ? OR request_payload LIKE ?",
			v, v, like,
		)
	}
	if v := strings.TrimSpace(f.Category); v != "" {
		tx = applyAlertEventCategoryFilter(tx, v)
	}
	return tx
}

func (r *AlertEventRepository) ListGroupedByGroupKey(ctx context.Context, f AlertEventListFilter, offset, limit int) ([]AlertEventGroupRow, int64, error) {
	sub := r.listQuery(ctx, f).
		Select(`group_key,
			MAX(title) AS title,
			COUNT(*) AS cnt,
			MAX(created_at) AS last_at,
			MAX(status) AS status,
			MAX(severity) AS severity,
			MAX(cluster) AS cluster`).
		Where("group_key <> ''").
		Group("group_key")
	var total int64
	if err := r.db.WithContext(ctx).Table("(?) AS g", sub).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AlertEventGroupRow
	err := sub.Order("last_at DESC").Offset(offset).Limit(limit).Scan(&list).Error
	return list, total, err
}

func (r *AlertEventRepository) List(ctx context.Context, f AlertEventListFilter, offset, limit int) ([]model.AlertEvent, int64, error) {
	tx := r.listQuery(ctx, f)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.AlertEvent
	err := tx.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *AlertEventRepository) ListFiringByGroupKeys(ctx context.Context, groupKeys []string) ([]model.AlertEvent, error) {
	if len(groupKeys) == 0 {
		return nil, nil
	}
	var list []model.AlertEvent
	err := r.db.WithContext(ctx).
		Where("group_key IN ? AND status = ?", groupKeys, "firing").
		Order("id DESC").
		Find(&list).Error
	return list, err
}

func (r *AlertEventRepository) HistoryStats(ctx context.Context, dayStart, dayEnd time.Time) (*AlertHistoryStatsRow, error) {
	stats := &AlertHistoryStatsRow{}
	var agg struct {
		Total        int64
		Firing       int64
		Resolved     int64
		Success      int64
		Failed       int64
		TodayCreated int64
	}
	if err := r.db.WithContext(ctx).Raw(`
SELECT
  COUNT(*) AS total,
  COALESCE(SUM(CASE WHEN status = 'firing' THEN 1 ELSE 0 END), 0) AS firing,
  COALESCE(SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END), 0) AS resolved,
  COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) AS success,
  COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) AS failed,
  COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS today_created
FROM alert_events
WHERE deleted_at IS NULL`, dayStart, dayEnd).Scan(&agg).Error; err != nil {
		return nil, err
	}
	stats.Total = agg.Total
	stats.Firing = agg.Firing
	stats.Resolved = agg.Resolved
	stats.Success = agg.Success
	stats.Failed = agg.Failed
	stats.TodayCreated = agg.TodayCreated
	if err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(cluster, '')) != ''").
		Group("cluster").Order("cluster ASC").Limit(500).
		Pluck("cluster", &stats.ClusterValues).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("TRIM(COALESCE(monitor_pipeline, '')) != ''").
		Group("monitor_pipeline").Order("monitor_pipeline ASC").Limit(32).
		Pluck("monitor_pipeline", &stats.MonitorPipelineValues).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Select("datasource_id AS id, MAX(datasource_name) AS name").
		Where("datasource_id > ?", 0).
		Group("datasource_id").Order("id DESC").Limit(200).
		Scan(&stats.DatasourceFilterOptions).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

var _ AlertEventRepo = (*AlertEventRepository)(nil)