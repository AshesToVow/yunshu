package repository

import (
	"context"
	"fmt"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type OverviewRepository struct {
	db *gorm.DB
}

func NewOverviewRepository(db *gorm.DB) OverviewRepo {
	return &OverviewRepository{db: db}
}

func (r *OverviewRepository) DialectName() string {
	if r.db == nil {
		return ""
	}
	return r.db.Dialector.Name()
}

func dayExprForColumn(dialect, column string) string {
	switch dialect {
	case "postgres":
		return fmt.Sprintf("to_char(%s, 'YYYY-MM-DD')", column)
	case "sqlite":
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s)", column)
	default:
		return fmt.Sprintf("DATE_FORMAT(%s, '%%Y-%%m-%%d')", column)
	}
}

func (r *OverviewRepository) applyProjectScope(q *gorm.DB, projectIDs []uint, unrestricted bool, column string) *gorm.DB {
	if unrestricted {
		return q
	}
	if len(projectIDs) == 0 {
		return q.Where("1 = 0")
	}
	return q.Where(column+" IN ?", projectIDs)
}

func (r *OverviewRepository) CountReleaseLaunchesByProjectDay(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewProjectDayCount, error) {
	daySQL := dayExprForColumn(r.DialectName(), "r.finished_at")
	query := fmt.Sprintf(`
		SELECT r.project_id, p.name AS project_name, %s AS day, COUNT(*) AS cnt
		FROM cicd_release_runs r
		INNER JOIN projects p ON p.id = r.project_id AND p.deleted_at IS NULL
		WHERE r.deleted_at IS NULL
		  AND r.status = ?
		  AND r.finished_at IS NOT NULL
		  AND r.finished_at >= ?
		  AND r.finished_at < ?
	`, daySQL)
	args := []any{model.CicdRunStatusSuccess, start, end}
	if !unrestricted {
		if len(projectIDs) == 0 {
			return nil, nil
		}
		query += " AND r.project_id IN ?"
		args = append(args, projectIDs)
	}
	query += fmt.Sprintf(" GROUP BY r.project_id, p.name, %s ORDER BY day ASC", daySQL)
	var rows []OverviewProjectDayCount
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *OverviewRepository) CountReleaseRunsByPerson(ctx context.Context, start, end time.Time, projectIDs []uint, unrestricted bool) ([]OverviewPersonCount, error) {
	if !unrestricted && len(projectIDs) == 0 {
		return nil, nil
	}
	q := r.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
		Select("COALESCE(NULLIF(TRIM(submitter_name), ''), '未知') AS person, COUNT(*) AS cnt").
		Where("created_at >= ? AND created_at < ?", start, end)
	q = r.applyProjectScope(q, projectIDs, unrestricted, "project_id")
	var rows []OverviewPersonCount
	if err := q.Group("person").Order("cnt DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *OverviewRepository) LoadMetrics(ctx context.Context, regPendingStatus int) (*OverviewMetricsRow, error) {
	out := &OverviewMetricsRow{}
	err := r.db.WithContext(ctx).Raw(
		`SELECT
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS users_count,
			(SELECT COUNT(*) FROM k8s_clusters WHERE deleted_at IS NULL) AS clusters_count,
			(SELECT COUNT(*) FROM registration_requests WHERE status = ?) AS pending_registrations_count,
			(SELECT COUNT(*) FROM servers WHERE deleted_at IS NULL) AS servers_count`,
		regPendingStatus,
	).Scan(out).Error
	return out, err
}

func (r *OverviewRepository) ListEnabledClusters(ctx context.Context, projectIDs []uint, unrestricted bool) ([]model.K8sCluster, error) {
	var clusters []model.K8sCluster
	// 总览只需集群 ID；避免一次拉出全部 kubeconfig/direct longtext。
	q := r.db.WithContext(ctx).Model(&model.K8sCluster{}).Select("id").Where("status = ?", 1)
	if !unrestricted {
		if len(projectIDs) == 0 {
			q = q.Where("owning_project_id IS NULL")
		} else {
			q = q.Where("(owning_project_id IS NULL OR owning_project_id IN ?)", projectIDs)
		}
	}
	err := q.Find(&clusters).Error
	return clusters, err
}

func (r *OverviewRepository) FillAlertAndAgentStats(ctx context.Context, dayStart, dayEnd, agentCutoff time.Time) (*OverviewStats, error) {
	out := &OverviewStats{}
	type row struct {
		AlertFiringCount         int64
		AlertEventsTodayCount    int64
		LoggieAgentsOnlineCount  int64
		LoggieAgentsTotal        int64
	}
	var scanned row
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM alert_events WHERE deleted_at IS NULL AND status = ?) AS alert_firing_count,
			(SELECT COUNT(*) FROM alert_events WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?) AS alert_events_today_count,
			(SELECT COUNT(*) FROM loggie_agents WHERE last_seen_at IS NOT NULL AND last_seen_at >= ?) AS loggie_agents_online_count,
			(SELECT COUNT(*) FROM loggie_agents) AS loggie_agents_total
	`, "firing", dayStart, dayEnd, agentCutoff).Scan(&scanned).Error
	if err != nil {
		return out, nil
	}
	out.AlertFiringCount = scanned.AlertFiringCount
	out.AlertEventsTodayCount = scanned.AlertEventsTodayCount
	out.LoggieAgentsOnlineCount = scanned.LoggieAgentsOnlineCount
	if scanned.LoggieAgentsTotal >= scanned.LoggieAgentsOnlineCount {
		out.LoggieAgentsOfflineCount = scanned.LoggieAgentsTotal - scanned.LoggieAgentsOnlineCount
	}
	return out, nil
}
