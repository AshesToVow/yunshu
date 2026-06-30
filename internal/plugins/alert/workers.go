package alert

import (
	"context"

	"yunshu/internal/plugin"
)

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil || !rt.IsEnabled("alert") {
		return nil
	}
	if svc := rt.AlertSvc(); svc != nil {
		svc.RunBackgroundWorkers(bgCtx)
	}
	return nil
}
