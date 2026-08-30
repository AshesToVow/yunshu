package backup

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/plugin"
	"yunshu/internal/service"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "backup" }
func (m *module) Description() string { return "MySQL 定时备份与 MinIO 归档" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/mysql-backup"},
		APIPrefixes:      []string{"/api/v1/mysql-backup"},
		DependsOn:        []string{"project"},
		Workers:          []string{"mysql_backup_scheduler"},
	}
}

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
	if svc, ok := rt.MysqlBackup.(*service.MysqlBackupService); ok && svc != nil {
		lifecycle.Go("backup.mysql-scheduler", func() { svc.RunMysqlBackupScheduler(bgCtx) })
	}
	return nil
}
