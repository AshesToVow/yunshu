package inspect

import (
	"context"
	"log/slog"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/cronutil"
)

const inspectSchedulerLeaderKey = "inspect:scheduler:leader"

// RunScheduler 定时轮询启用中的巡检计划（阻塞至 ctx 取消）。
func (s *Service) RunScheduler(ctx context.Context) {
	if s == nil {
		return
	}
	log := slog.Default().With("component", "inspect.scheduler")
	log.Info("inspect scheduler started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("inspect scheduler stopped")
			return
		case <-ticker.C:
			s.tickSchedules(ctx)
			if _, err := s.CleanupExpiredReports(ctx); err != nil {
				log.Warn("inspect report cleanup failed", "error", err.Error())
			}
		}
	}
}

func (s *Service) tickSchedules(ctx context.Context) {
	ok, release := s.acquireLeader(ctx)
	if !ok {
		return
	}
	defer release()

	var plans []model.InspectPlan
	if err := s.db.WithContext(ctx).
		Where("enabled = ? AND datasource_id > 0 AND cron_spec <> ''", true).
		Find(&plans).Error; err != nil {
		return
	}
	now := time.Now()
	for i := range plans {
		plan := &plans[i]
		if !cronutil.ShouldRunWithDayAnchor(plan.CronSpec, plan.LastRunAt, now) {
			continue
		}
		runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		_, err := s.executeRun(runCtx, plan, plan.DatasourceID, "cron", 0, "")
		cancel()
		if err != nil {
			slog.Default().With("component", "inspect.scheduler").Warn("inspect cron run failed",
				"project_id", plan.ProjectID, "error", err.Error())
		}
	}
}

func (s *Service) acquireLeader(ctx context.Context) (bool, func()) {
	if s.redis == nil {
		return true, func() {}
	}
	ok, err := s.redis.SetNX(ctx, inspectSchedulerLeaderKey, "1", 2*time.Minute).Result()
	if err != nil || !ok {
		return false, func() {}
	}
	return true, func() {
		relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redis.Del(relCtx, inspectSchedulerLeaderKey).Err()
	}
}
