package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

func (r *OverviewRepository) CountServersByStatus(ctx context.Context) ([]OverviewLabelCount, error) {
	var rows []OverviewLabelCount
	err := r.db.WithContext(ctx).Model(&model.Server{}).
		Select("CAST(status AS CHAR) AS label, COUNT(*) AS cnt").
		Where("deleted_at IS NULL").
		Group("status").
		Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) CountAlertsBySeverity(ctx context.Context, projectIDs []uint, unrestricted bool) ([]OverviewLabelCount, error) {
	q := r.db.WithContext(ctx).Model(&model.AlertCurEvent{}).
		Select("COALESCE(NULLIF(TRIM(severity), ''), 'unknown') AS label, COUNT(*) AS cnt")
	if !unrestricted {
		if len(projectIDs) == 0 {
			return nil, nil
		}
		q = q.Where("project_id IN ?", projectIDs)
	}
	var rows []OverviewLabelCount
	err := q.Group("label").Order("cnt DESC").Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) CountFiringAlerts(ctx context.Context, projectIDs []uint, unrestricted bool) (int64, error) {
	q := r.db.WithContext(ctx).Model(&model.AlertCurEvent{})
	if !unrestricted {
		if len(projectIDs) == 0 {
			return 0, nil
		}
		q = q.Where("project_id IN ?", projectIDs)
	}
	var n int64
	err := q.Count(&n).Error
	return n, err
}

func (r *OverviewRepository) ListFiringAlertsTop(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewAlertBrief, error) {
	if limit <= 0 {
		limit = 10
	}
	q := r.db.WithContext(ctx).Model(&model.AlertCurEvent{}).
		Select("id, fingerprint, alertname, severity, cluster, project_id, summary, starts_at")
	if !unrestricted {
		if len(projectIDs) == 0 {
			return nil, nil
		}
		q = q.Where("project_id IN ?", projectIDs)
	}
	var rows []OverviewAlertBrief
	err := q.Order("starts_at DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) ListRecentReleases(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewReleaseBrief, error) {
	if limit <= 0 {
		limit = 10
	}
	if !unrestricted && len(projectIDs) == 0 {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Table("cicd_release_runs AS r").
		Select(`r.id, r.project_id, COALESCE(p.name, '') AS project_name, r.title, r.status, r.tenv,
			COALESCE(NULLIF(TRIM(r.submitter_name), ''), '未知') AS submitter_name, r.finished_at, r.created_at`).
		Joins("LEFT JOIN projects p ON p.id = r.project_id AND p.deleted_at IS NULL").
		Where("r.deleted_at IS NULL")
	q = r.applyProjectScope(q, projectIDs, unrestricted, "r.project_id")
	var rows []OverviewReleaseBrief
	err := q.Order("r.id DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) ListRecentChanges(ctx context.Context, projectIDs []uint, unrestricted bool, limit int) ([]OverviewChangeBrief, error) {
	if limit <= 0 {
		limit = 10
	}
	q := r.db.WithContext(ctx).Model(&model.ChangeEvent{}).
		Select("id, project_id, source, action, risk_level, status, summary, started_at")
	if !unrestricted {
		if len(projectIDs) == 0 {
			return nil, nil
		}
		q = q.Where("project_id IN ?", projectIDs)
	}
	var rows []OverviewChangeBrief
	err := q.Order("started_at DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) CountLoggieByHealth(ctx context.Context) ([]OverviewLabelCount, error) {
	var rows []OverviewLabelCount
	err := r.db.WithContext(ctx).Model(&model.LoggieAgent{}).
		Select("COALESCE(NULLIF(TRIM(health_status), ''), 'unknown') AS label, COUNT(*) AS cnt").
		Group("label").
		Order("cnt DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) ListLoggieOfflineSample(ctx context.Context, agentCutoff time.Time, limit int) ([]OverviewLoggieBrief, error) {
	if limit <= 0 {
		limit = 8
	}
	var rows []OverviewLoggieBrief
	err := r.db.WithContext(ctx).Model(&model.LoggieAgent{}).
		Select("id, project_id, server_id, health_status, COALESCE(last_error, '') AS last_error, last_seen_at").
		Where("last_seen_at IS NULL OR last_seen_at < ?", agentCutoff).
		Order("last_seen_at ASC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *OverviewRepository) CountAIInvestigations(ctx context.Context, dayStart, dayEnd time.Time) (today int64, open int64, err error) {
	type row struct {
		Today int64
		Open  int64
	}
	var scanned row
	err = r.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= ? AND created_at < ? THEN 1 ELSE 0 END), 0) AS today,
			COALESCE(SUM(CASE WHEN status IN ('collecting', 'analyzing') THEN 1 ELSE 0 END), 0) AS open
		FROM ai_investigations
	`, dayStart, dayEnd).Scan(&scanned).Error
	if err != nil {
		// 表可能尚未迁移：大屏降级为 0
		return 0, 0, nil
	}
	return scanned.Today, scanned.Open, nil
}
