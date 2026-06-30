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
	q := r.db.WithContext(ctx).Model(&model.K8sCluster{}).Where("status = ?", 1)
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
	_ = r.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("status = ?", "firing").Count(&out.AlertFiringCount).Error
	_ = r.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&out.AlertEventsTodayCount).Error
	var online int64
	_ = r.db.WithContext(ctx).Model(&model.LogAgent{}).
		Where("deleted_at IS NULL AND status = ?", 1).
		Where("last_seen_at IS NOT NULL AND last_seen_at >= ?", agentCutoff).
		Count(&online).Error
	out.LogAgentsOnlineCount = online
	var totalAgents int64
	_ = r.db.WithContext(ctx).Model(&model.LogAgent{}).
		Where("deleted_at IS NULL AND status = ?", 1).
		Count(&totalAgents).Error
	if totalAgents >= online {
		out.LogAgentsOfflineCount = totalAgents - online
	}
	return out, nil
}
