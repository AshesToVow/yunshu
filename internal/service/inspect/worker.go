package inspect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"yunshu/internal/model"
)

const (
	inspectWorkerCount       = 2
	inspectJobQueueSize      = 128
	inspectRunTimeout        = 10 * time.Minute
	inspectStaleRunningAfter = 30 * time.Minute
)

// startWorkers 启动巡检异步队列（幂等）。HTTP 入队后由 worker 执行，避免阻塞「立即巡检」。
func (s *Service) startWorkers(ctx context.Context) {
	if s == nil {
		return
	}
	s.workerOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		s.workerCtx = ctx
		s.jobCh = make(chan uint, inspectJobQueueSize)
		log := slog.Default().With("component", "inspect.worker")
		log.Info("inspect workers started", "count", inspectWorkerCount)
		for i := 0; i < inspectWorkerCount; i++ {
			go s.inspectWorkerLoop(i)
		}
		s.reclaimOrphanRuns(context.Background())
	})
}

func (s *Service) enqueueRun(runID uint) {
	if s == nil || runID == 0 {
		return
	}
	s.startWorkers(context.Background())
	if s.jobCh == nil {
		return
	}
	select {
	case s.jobCh <- runID:
	default:
		go func() {
			select {
			case s.jobCh <- runID:
			case <-time.After(30 * time.Second):
				_, _ = s.failRun(context.Background(), &model.InspectRun{ID: runID}, fmt.Errorf("巡检队列繁忙，请稍后重试"))
			}
		}()
	}
}

func (s *Service) inspectWorkerLoop(workerID int) {
	log := slog.Default().With("component", "inspect.worker", "worker_id", workerID)
	for {
		select {
		case <-s.workerCtx.Done():
			log.Info("inspect worker stopped")
			return
		case runID, ok := <-s.jobCh:
			if !ok {
				return
			}
			s.processQueuedRun(runID)
		}
	}
}

func (s *Service) processQueuedRun(runID uint) {
	log := slog.Default().With("component", "inspect.worker", "run_id", runID)
	defer func() {
		if r := recover(); r != nil {
			log.Error("inspect run panic", "panic", fmt.Sprint(r))
			_, _ = s.failRun(context.Background(), &model.InspectRun{ID: runID}, fmt.Errorf("巡检异常: %v", r))
		}
	}()

	dbCtx := context.Background()
	var run model.InspectRun
	if err := s.db.WithContext(dbCtx).First(&run, runID).Error; err != nil {
		log.Warn("load run failed", "error", err.Error())
		return
	}
	switch run.Status {
	case "success", "failed":
		return
	case "pending", "running":
	default:
		return
	}

	var plan model.InspectPlan
	if err := s.db.WithContext(dbCtx).First(&plan, run.PlanID).Error; err != nil {
		_, _ = s.failRun(dbCtx, &run, fmt.Errorf("加载巡检计划失败: %w", err))
		return
	}

	now := time.Now()
	res := s.db.WithContext(dbCtx).Model(&model.InspectRun{}).
		Where("id = ? AND status IN ?", runID, []string{"pending", "running"}).
		Updates(map[string]any{
			"status":     "running",
			"started_at": now,
		})
	if res.Error != nil {
		log.Warn("mark running failed", "error", res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		return
	}
	run.Status = "running"
	run.StartedAt = &now

	runCtx, cancel := context.WithTimeout(context.Background(), inspectRunTimeout)
	defer cancel()
	if _, err := s.performRun(runCtx, &plan, &run); err != nil {
		log.Warn("inspect run failed", "error", err.Error())
	}
}

// reclaimOrphanRuns 进程重启后：pending 重新入队；遗留 running 标记失败，避免永远「执行中」。
func (s *Service) reclaimOrphanRuns(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	log := slog.Default().With("component", "inspect.worker")
	finished := time.Now()
	staleBefore := finished.Add(-5 * time.Minute)
	res := s.db.WithContext(ctx).Model(&model.InspectRun{}).
		Where("status = ? AND (started_at IS NULL OR started_at < ?)", "running", staleBefore).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": "巡检中断（服务重启或上次请求超时），请重新执行",
			"finished_at":   finished,
		})
	if res.Error == nil && res.RowsAffected > 0 {
		log.Warn("reclaimed orphan running inspect runs", "count", res.RowsAffected)
	}

	var pending []model.InspectRun
	if err := s.db.WithContext(ctx).Where("status = ?", "pending").Order("id ASC").Limit(200).Find(&pending).Error; err != nil {
		return
	}
	for _, r := range pending {
		s.enqueueRun(r.ID)
	}
	if len(pending) > 0 {
		log.Info("re-enqueued pending inspect runs", "count", len(pending))
	}
}

// reclaimStaleRunning 定时回收长时间卡在 running 的任务。
func (s *Service) reclaimStaleRunning(ctx context.Context) {
	if s == nil || s.db == nil {
		return
	}
	cutoff := time.Now().Add(-inspectStaleRunningAfter)
	finished := time.Now()
	_ = s.db.WithContext(ctx).Model(&model.InspectRun{}).
		Where("status = ? AND started_at IS NOT NULL AND started_at < ?", "running", cutoff).
		Updates(map[string]any{
			"status":        "failed",
			"error_message": "巡检超时未完成，已自动标记失败，请重新执行",
			"finished_at":   finished,
		}).Error
}
