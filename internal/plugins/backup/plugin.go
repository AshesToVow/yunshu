package backup

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

func (m *module) Name() string        { return "backup" }
func (m *module) Description() string { return "MySQL 定时备份与 MinIO 归档" }

func (m *module) Models() []any {
	return []any{
		&model.MysqlBackupInstance{},
		&model.MysqlBackupJob{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc := rt.MysqlBackupSvc(); svc != nil {
		go svc.RunMysqlBackupScheduler(bgCtx)
	}
	return nil
}
