package inspect

import (
	"context"
	"log/slog"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/cronutil"
	"yunshu/internal/pkg/lifecycle"
)

const inspectSchedulerLeaderKey = "inspect:scheduler:leader"

// RunScheduler 定时轮询启用中的巡检计划（阻塞至 ctx 取消）。
func (s *Service) RunScheduler(ctx context.Context) {
	if s == nil {
		return
	}
	s.startWorkers(ctx)
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
			s.reclaimStaleRunning(ctx)
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
		// 执行前先占位 LastRunAt，避免长任务窗口内重复触发
		claimed := now
		if err := s.db.WithContext(ctx).Model(plan).Update("last_run_at", claimed).Error; err != nil {
			continue
		}
		plan.LastRunAt = &claimed

		runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		_, err := s.executeRun(runCtx, plan, plan.DatasourceID, "cron", 0, "")
		cancel()
		if err != nil {
			slog.Default().With("component", "inspect.scheduler").Warn("inspect cron enqueue failed",
				"project_id", plan.ProjectID, "error", err.Error())
		}
	}
}

func (s *Service) acquireLeader(ctx context.Context) (bool, func()) {
	if s.redis == nil {
		return true, func() {}
	}
	// TTL 覆盖最长巡检窗口，配合 renewLeader 心跳续期
	ok, err := s.redis.SetNX(ctx, inspectSchedulerLeaderKey, "1", 12*time.Minute).Result()
	if err != nil || !ok {
		return false, func() {}
	}
	return true, func() {
		relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redis.Del(relCtx, inspectSchedulerLeaderKey).Err()
	}
}

// renewLeader 在长任务期间周期性续期 leader 锁。
func (s *Service) renewLeader(ctx context.Context) func() {
	if s.redis == nil {
		return func() {}
	}
	done := make(chan struct{})
	lifecycle.GoDetached("inspect.leader-renew", func() {
		t := time.NewTicker(2 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case <-t.C:
				_ = s.redis.Expire(ctx, inspectSchedulerLeaderKey, 12*time.Minute).Err()
			}
		}
	})
	return func() { close(done) }
}
