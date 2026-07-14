package logplatform

import (
	"context"

	"log/slog"
	"yunshu/internal/config"
	"yunshu/internal/pkg/cronutil"
)

// RunLogRetentionScheduler 按配置 Cron 清理 ES 过期日志索引/文档。
func RunLogRetentionScheduler(ctx context.Context, svc *LogRetentionService, _ config.ElasticsearchConfig) {
	if svc == nil || svc.es == nil {
		slog.Default().With("component", "logretention").Info("Log retention scheduler skipped: elasticsearch disabled")
		return
	}
	cfg, err := svc.es.Resolve(ctx)
	if err != nil || !cfg.Enabled {
		slog.Default().With("component", "logretention").Info("Log retention scheduler skipped: elasticsearch disabled")
		return
	}
	spec := cfg.CleanupCronSpec
	slog.Default().With("component", "logretention").Info("Started log retention scheduler", "cron", spec)
	cronutil.RunWorker(ctx, spec, func() {
		res, err := svc.RunCleanup(ctx)
		if err != nil {
			slog.Default().With("component", "logretention").Warn("Log retention cleanup failed", "error", err)
			return
		}
		if res != nil && (len(res.DeletedIndices) > 0 || res.DeletedDocuments > 0) {
			slog.Default().With("component", "logretention").Info("Log retention cleanup done",
				"deleted_indices", len(res.DeletedIndices),
				"deleted_documents", res.DeletedDocuments,
			)
		}
	}, "0 3 * * *")
}
