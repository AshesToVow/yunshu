package inspect

import (
	"context"
	"log/slog"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/cronutil"
)

const (
	inspectSchedulerLeaderKey = "inspect:scheduler:leader"
	// inspectLeaderTTL 崩溃兜底：持锁副本非正常退出时最多阻塞其他副本这么久。
	// 取值需大于单次 tick 的最坏耗时（清理 + 入队），远小于原先的 12m，避免故障后长时间无人调度。
	inspectLeaderTTL = 2 * time.Minute
)

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
			s.tick(ctx, log)
		}
	}
}

// tick 单次调度周期。整周期都在 leader 锁内：reclaimStaleRunning 与 CleanupExpiredReports
// 同样是全局写操作（改 run 状态、删报告文件），多副本并发执行会互相踩，
// 因此不能只把锁加在 tickSchedules 上。
func (s *Service) tick(ctx context.Context, log *slog.Logger) {
	ok, release := s.acquireLeader(ctx)
	if !ok {
		return
	}
	defer release()

	s.reclaimStaleRunning(ctx)
	s.tickSchedules(ctx)
	if _, err := s.CleanupExpiredReports(ctx); err != nil {
		log.Warn("inspect report cleanup failed", "error", err.Error())
	}
}

// tickSchedules 由 tick 在持有 leader 锁时调用。
func (s *Service) tickSchedules(ctx context.Context) {
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
	// TTL 只作为持锁进程崩溃时的兜底释放：正常路径在 tick 结束时 Del。
	// 巡检本体在 worker 中异步执行，不在锁内，因此 TTL 只需覆盖单次 tick 的最坏耗时。
	ok, err := s.redis.SetNX(ctx, inspectSchedulerLeaderKey, "1", inspectLeaderTTL).Result()
	if err != nil || !ok {
		return false, func() {}
	}
	return true, func() {
		relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redis.Del(relCtx, inspectSchedulerLeaderKey).Err()
	}
}
