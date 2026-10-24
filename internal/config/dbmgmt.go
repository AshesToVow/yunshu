package config

// DbmgmtConfig 数据库管理插件配置（数据字典 dbmgmt_* 可覆盖）。
type DbmgmtConfig struct {
	QueryTimeoutSeconds           int      `mapstructure:"query_timeout_seconds"`
	MaxResultRows                 int      `mapstructure:"max_result_rows"`
	MaxImportFileMB               int      `mapstructure:"max_import_file_mb"`
	ProdForceApproval             bool     `mapstructure:"prod_force_approval"`
	ForbidSelfApprove             bool     `mapstructure:"forbid_self_approve"`
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
		ForbidSelfApprove:             true,
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

// ApplyDefaults 填充 dbmgmt 零值字段。
func (c *DbmgmtConfig) ApplyDefaults() {
	def := DefaultDbmgmtConfig()
	if c.QueryTimeoutSeconds <= 0 {
		c.QueryTimeoutSeconds = def.QueryTimeoutSeconds
	}
	if c.MaxResultRows <= 0 {
		c.MaxResultRows = def.MaxResultRows
	}
	if c.MaxImportFileMB <= 0 {
		c.MaxImportFileMB = def.MaxImportFileMB
	}
	if c.ApprovalSlaHours <= 0 {
		c.ApprovalSlaHours = def.ApprovalSlaHours
	}
	if c.ApprovalReminderIntervalHours <= 0 {
		c.ApprovalReminderIntervalHours = def.ApprovalReminderIntervalHours
	}
	if len(c.AllowedDrivers) == 0 {
		c.AllowedDrivers = append([]string(nil), def.AllowedDrivers...)
	}
	if c.PingIntervalSeconds <= 0 {
		c.PingIntervalSeconds = def.PingIntervalSeconds
	}
	if c.MaxConcurrentPerInstance <= 0 {
		c.MaxConcurrentPerInstance = def.MaxConcurrentPerInstance
	}
}

