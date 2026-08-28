package inspect

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/plugin"
	inspectsvc "yunshu/internal/service/inspect"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "inspect" }
func (m *module) Description() string { return "项目级 Prometheus 巡检报告与定时调度" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/project-inspect"},
		APIPrefixes:      []string{},
		DependsOn:        []string{"project", "alert"},
		Workers:          []string{"inspect_scheduler"},
	}
}

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
	if svc, ok := rt.Inspect.(*inspectsvc.Service); ok && svc != nil {
		_ = svc.SeedGlobalTemplates(bgCtx)
		_ = svc.SeedReportTemplates(bgCtx)
		lifecycle.Go("inspect.scheduler", func() { svc.RunScheduler(bgCtx) })
	}
	return nil
}
