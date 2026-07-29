package changegate

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/service/changeevent"
)

// AssertWritable 冻结窗硬拦截；冲突仅写入事件摘要（默认不阻断）。
// 被冻结时写入 aborted change_event 并返回错误。
func AssertWritable(ctx context.Context, in CheckInput) error {
	res := Check(ctx, in)
	if res.Allowed {
		return nil
	}
	if res.BlockedByFreeze && in.ProjectID > 0 {
		changeevent.Record(ctx, changeevent.Input{
			ProjectID: in.ProjectID,
			ServiceID: in.ServiceID,
			Source:    in.Source,
			Action:    "freeze_blocked",
			RiskLevel: model.ChangeRiskHigh,
			Status:    model.ChangeStatusAborted,
			Summary:   res.Message,
			Payload: map[string]any{
				"freeze_window_id": res.FreezeWindowID,
				"freeze_name":      res.FreezeName,
				"env":              in.Env,
				"namespace":        in.Namespace,
				"action":           in.Action,
			},
		})
	}
	return ErrBlocked(res)
}

// Peek 只读检查（含冲突警告），不写事件。
func Peek(ctx context.Context, in CheckInput) CheckResult {
	return Check(ctx, in)
}
