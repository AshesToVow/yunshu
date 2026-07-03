package alert

import (
	"context"

	"yunshu/internal/pkg/cronutil"
)

// cloudExpirySchedulerSpec 云到期内置轮询节拍（六段式，含秒），仅用于判断是否到达各规则在控制台配置的 Cron；
// 与 alert.monitor_eval_cron_spec（内置 PromQL 监控规则）解耦。每分钟检查一次即可满足小时级 Cron 精度。
const cloudExpirySchedulerSpec = "0 * * * * *"

func (s *AlertService) runCloudExpiryEvaluator(ctx context.Context) {
	alertLog().Info("Started cloud expiry scheduler", "poll_cron", cloudExpirySchedulerSpec)
	cronutil.RunWorker(ctx, cloudExpirySchedulerSpec, func() {
		_ = s.tickCloudExpiryRules(ctx)
	}, "")
}
