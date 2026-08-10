package dictconfig

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// EsmgmtBackupSchedulerDictTypes ES 索引定时备份 Worker 字典项。
type EsmgmtBackupSchedulerDictTypes struct {
	Enabled  string
	TickSpec string
}

func DefaultEsmgmtBackupSchedulerDictTypes() EsmgmtBackupSchedulerDictTypes {
	return EsmgmtBackupSchedulerDictTypes{
		Enabled:  "esmgmt_backup_scheduler_enabled",
		TickSpec: "esmgmt_backup_scheduler_tick_spec",
	}
}

// EsmgmtBackupSchedulerConfig 后台调度节拍。
type EsmgmtBackupSchedulerConfig struct {
	Enabled  bool
	TickSpec string
}

const defaultEsmgmtBackupSchedulerTick = "*/30 * * * * *"

// ResolveEsmgmtBackupSchedulerConfig 字典优先；未配置时默认启用、每 30 秒轮询。
func ResolveEsmgmtBackupSchedulerConfig(ctx context.Context, db *gorm.DB, types EsmgmtBackupSchedulerDictTypes) EsmgmtBackupSchedulerConfig {
	cfg := EsmgmtBackupSchedulerConfig{
		Enabled:  true,
		TickSpec: defaultEsmgmtBackupSchedulerTick,
	}
	if db == nil {
		return cfg
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.Enabled); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.Enabled = bv
		}
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.TickSpec); ok {
		cfg.TickSpec = strings.TrimSpace(v)
	}
	if cfg.TickSpec == "" {
		cfg.TickSpec = defaultEsmgmtBackupSchedulerTick
	}
	return cfg
}
