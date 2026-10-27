package dictconfig

import (
	"context"

	"gorm.io/gorm"
)

// MysqlBackupSchedulerDictTypes 定时备份 Worker 字典项（不写 config.yaml）。
type MysqlBackupSchedulerDictTypes struct {
	Enabled  string
	TickSpec string
}

func DefaultMysqlBackupSchedulerDictTypes() MysqlBackupSchedulerDictTypes {
	return MysqlBackupSchedulerDictTypes{
		Enabled:  "mysql_backup_scheduler_enabled",
		TickSpec: "mysql_backup_scheduler_tick_spec",
	}
}

// MysqlBackupSchedulerConfig 后台调度节拍。
type MysqlBackupSchedulerConfig struct {
	Enabled  bool
	TickSpec string
}

const defaultMysqlBackupSchedulerTick = "*/30 * * * * *"

// ResolveMysqlBackupSchedulerConfig 字典优先；未配置时默认启用、每 30 秒轮询一次 Cron 到点判断。
func ResolveMysqlBackupSchedulerConfig(ctx context.Context, db *gorm.DB, types MysqlBackupSchedulerDictTypes) MysqlBackupSchedulerConfig {
	cfg := MysqlBackupSchedulerConfig{
		Enabled:  true,
		TickSpec: defaultMysqlBackupSchedulerTick,
	}
	if db == nil {
		return cfg
	}
	snap := fetchDictSnapshot(ctx, db, types.Enabled, types.TickSpec)
	if v, ok := snap.get(types.Enabled); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.Enabled = bv
		}
	}
	if v, ok := snap.getNonEmpty(types.TickSpec); ok {
		cfg.TickSpec = v
	}
	if cfg.TickSpec == "" {
		cfg.TickSpec = defaultMysqlBackupSchedulerTick
	}
	return cfg
}
