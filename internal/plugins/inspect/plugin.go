package inspect

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "inspect" }
func (m *module) Description() string { return "项目级 Prometheus 巡检报告与定时调度" }

func (m *module) Models() []any {
	return []any{
		&model.InspectPlan{},
		&model.InspectItem{},
		&model.InspectRun{},
		&model.InspectReportTemplate{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc := rt.InspectSvc(); svc != nil {
		_ = svc.SeedGlobalTemplates(bgCtx)
		_ = svc.SeedReportTemplates(bgCtx)
		go svc.RunScheduler(bgCtx)
	}
	return nil
}
