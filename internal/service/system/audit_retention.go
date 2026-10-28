package system

import (
	"context"
	"log/slog"
	"time"

	"yunshu/internal/interfaces"
)

// RunAuditRetentionWorker 每天分批清理超过保留期的审计日志；days<=0 表示永久保留（不启动）。
// 审计日志默认不清（取证数据），需运维显式在配置中开启。
func RunAuditRetentionWorker(ctx context.Context, opRepo interfaces.OperationLogRepository, loginRepo interfaces.LoginLogRepository, days int) {
	if days <= 0 {
		return
	}
	logger := slog.Default().With("component", "audit-retention")
	logger.Info("audit log retention enabled", "days", days)

	sweep := func() {
		cutoff := time.Now().AddDate(0, 0, -days)
		sweepCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
		for name, repo := range map[string]repoCutoff{
			"operation_logs": opRepo,
			"login_logs":     loginRepo,
		} {
			if repo == nil {
				continue
			}
			deleted, err := repo.DeleteOlderThan(sweepCtx, cutoff)
			if err != nil {
				logger.Error("audit retention sweep failed", "table", name, "deleted", deleted, "error", err)
				continue
			}
			if deleted > 0 {
				logger.Info("audit retention sweep done", "table", name, "deleted", deleted)
			}
		}
	}

	sweep()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}

type repoCutoff interface {
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
