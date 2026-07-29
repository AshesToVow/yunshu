package alert

import (
	"context"

	"yunshu/internal/plugin"
	"yunshu/internal/service"
)

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil || !rt.IsEnabled("alert") {
		return nil
	}
	if svc, ok := rt.Alert.(*service.AlertService); ok && svc != nil {
		svc.RunBackgroundWorkers(bgCtx)
	}
	return nil
}
