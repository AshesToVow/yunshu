package config

// DbmgmtConfig 数据库管理插件配置（数据字典 dbmgmt_* 可覆盖）。
type DbmgmtConfig struct {
	QueryTimeoutSeconds           int      `mapstructure:"query_timeout_seconds"`
	MaxResultRows                 int      `mapstructure:"max_result_rows"`
	MaxImportFileMB               int      `mapstructure:"max_import_file_mb"`
	ProdForceApproval             bool     `mapstructure:"prod_force_approval"`
	ApprovalSlaHours              int      `mapstructure:"approval_sla_hours"`
	ApprovalReminderIntervalHours int      `mapstructure:"approval_reminder_interval_hours"`
	AllowedDrivers                []string `mapstructure:"allowed_drivers"`
	PingIntervalSeconds           int      `mapstructure:"ping_interval_seconds"`
	MaxConcurrentPerInstance      int      `mapstructure:"max_concurrent_per_instance"`
	GoInceptionEnabled            bool     `mapstructure:"goinception_enabled"`
	GoInceptionHost               string   `mapstructure:"goinception_host"`
	GoInceptionPort               int      `mapstructure:"goinception_port"`
	GoInceptionBackup             bool     `mapstructure:"goinception_backup"`
}

func DefaultDbmgmtConfig() DbmgmtConfig {
	return DbmgmtConfig{
		QueryTimeoutSeconds:           30,
		MaxResultRows:                 1000,
		MaxImportFileMB:               10,
		ProdForceApproval:             true,
		ApprovalSlaHours:              24,
		ApprovalReminderIntervalHours: 4,
		AllowedDrivers:                []string{"mysql", "postgres"},
		PingIntervalSeconds:           300,
		MaxConcurrentPerInstance:      5,
		GoInceptionEnabled:            false,
		GoInceptionHost:               "127.0.0.1",
		GoInceptionPort:               4000,
		GoInceptionBackup:             true,
	}
}
