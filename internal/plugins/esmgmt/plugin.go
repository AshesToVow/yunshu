package esmgmt

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/plugin"
	esmgmtsvc "yunshu/internal/service/esmgmt"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string { return "esmgmt" }
func (m *module) Description() string {
	return "ES 管理控制台：连接管理、集群概览、文档检索、索引模板、reindex 与备份"
}

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/esmgmt"},
		APIPrefixes:      []string{"/api/v1/esmgmt"},
		Workers:          []string{"esmgmt_backup_scheduler"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.EsmgmtConnection{},
		&model.EsmgmtBackupJob{},
		&model.EsmgmtBackupSchedule{},
		&model.EsmgmtRestoreJob{},
		&model.EsmgmtReindexJob{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc, ok := plugin.As[*esmgmtsvc.Service](rt.Esmgmt); ok && svc != nil {
		lifecycle.Go("esmgmt.backup-scheduler", func() { svc.RunBackupScheduler(bgCtx) })
	}
	return nil
}
