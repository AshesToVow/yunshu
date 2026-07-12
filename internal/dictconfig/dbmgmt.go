package dictconfig

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// DbmgmtDictTypes 数据字典覆盖 dbmgmt.* 的 dict_type。
type DbmgmtDictTypes struct {
	QueryTimeoutSeconds           string
	MaxResultRows                 string
	MaxImportFileMB               string
	ProdForceApproval             string
	ApprovalSlaHours              string
	ApprovalReminderIntervalHours string
	PingIntervalSeconds           string
	MaxConcurrentPerInstance      string
	GoInceptionEnabled            string
	GoInceptionHost               string
	GoInceptionPort               string
	GoInceptionBackup             string
}

func DefaultDbmgmtDictTypes() DbmgmtDictTypes {
	return DbmgmtDictTypes{
		QueryTimeoutSeconds:           "dbmgmt_query_timeout_seconds",
		MaxResultRows:                 "dbmgmt_max_rows",
		MaxImportFileMB:               "dbmgmt_max_import_file_mb",
		ProdForceApproval:             "dbmgmt_prod_force_approval",
		ApprovalSlaHours:              "dbmgmt_approval_sla_hours",
		ApprovalReminderIntervalHours: "dbmgmt_approval_reminder_interval_hours",
		PingIntervalSeconds:           "dbmgmt_ping_interval_seconds",
		MaxConcurrentPerInstance:      "dbmgmt_max_concurrent_per_instance",
		GoInceptionEnabled:            "dbmgmt_goinception_enabled",
		GoInceptionHost:               "dbmgmt_goinception_host",
		GoInceptionPort:               "dbmgmt_goinception_port",
		GoInceptionBackup:             "dbmgmt_goinception_backup",
	}
}

// ResolveDbmgmtConfig 字典优先合并 dbmgmt 配置。
func ResolveDbmgmtConfig(ctx context.Context, db *gorm.DB, base config.DbmgmtConfig) config.DbmgmtConfig {
	if db == nil {
		return base
	}
	types := DefaultDbmgmtDictTypes()
	if v, ok := FetchEnabledDictValue(ctx, db, types.QueryTimeoutSeconds); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.QueryTimeoutSeconds = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.MaxResultRows); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.MaxResultRows = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.MaxImportFileMB); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.MaxImportFileMB = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.ProdForceApproval); ok {
		base.ProdForceApproval = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.ApprovalSlaHours); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.ApprovalSlaHours = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.ApprovalReminderIntervalHours); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.ApprovalReminderIntervalHours = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.PingIntervalSeconds); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.PingIntervalSeconds = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.MaxConcurrentPerInstance); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.MaxConcurrentPerInstance = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.GoInceptionEnabled); ok {
		base.GoInceptionEnabled = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.GoInceptionHost); ok {
		if h := strings.TrimSpace(v); h != "" {
			base.GoInceptionHost = h
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.GoInceptionPort); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			base.GoInceptionPort = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.GoInceptionBackup); ok {
		base.GoInceptionBackup = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	return base
}
