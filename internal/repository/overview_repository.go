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

func dayExpr(dialect string) string {
	switch dialect {
	case "postgres":
		return "to_char(created_at, 'YYYY-MM-DD')"
	case "sqlite":
		return "strftime('%Y-%m-%d', created_at)"
	default:
		return "DATE_FORMAT(created_at, '%Y-%m-%d')"
	}
}

func (r *OverviewRepository) countByDay(ctx context.Context, table, daySQL, where string, start, end time.Time, args ...any) (map[string]int64, error) {
	query := fmt.Sprintf(
		"SELECT %s AS day, COUNT(*) AS cnt FROM %s WHERE created_at >= ? AND created_at < ? %s GROUP BY %s",
		daySQL, table, where, daySQL,
	)
	allArgs := append([]any{start, end}, args...)
	var rows []OverviewDayCount
	if err := r.db.WithContext(ctx).Raw(query, allArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(rows))
	for _, row := range rows {
		m[row.Day] = row.Cnt
	}
	return m, nil
}

func (r *OverviewRepository) CountLoginLogsByDay(ctx context.Context, start, end time.Time, status int) (map[string]int64, error) {
	return r.countByDay(ctx, "login_logs", dayExpr(r.DialectName()), "AND status = ?", start, end, status)
}

func (r *OverviewRepository) CountOperationLogsByDay(ctx context.Context, start, end time.Time) (map[string]int64, error) {
	return r.countByDay(ctx, "operation_logs", dayExpr(r.DialectName()), "", start, end)
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
