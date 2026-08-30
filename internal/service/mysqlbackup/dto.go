// MySQL 备份对外 DTO 与模型转换：请求/响应结构、列表查询、mysqldump 选项序列化。
// 仅数据结构与纯转换，不含 IO；服务装配见 mysql_backup_service.go。
package mysqlbackup

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/mysqlbackup"
)

type MysqlBackupInstanceItem struct {
	ID                 uint                        `json:"id"`
	ProjectID          uint                        `json:"project_id"`
	ServerID           uint                        `json:"server_id"`
	ServerName         string                      `json:"server_name,omitempty"`
	Name               string                      `json:"name"`
	Enabled            bool                        `json:"enabled"`
	MysqlHost          string                      `json:"mysql_host"`
	MysqlPort          int                         `json:"mysql_port"`
	MysqlSocket        string                      `json:"mysql_socket"`
	MysqlUser          string                      `json:"mysql_user"`
	BackupMode         string                      `json:"backup_mode"`
	BackupScope        string                      `json:"backup_scope"`
	DatabaseName       string                      `json:"database_name"`
	TableName          string                      `json:"table_name"`
	DatabaseNames      string                      `json:"database_names"`
	RemoteDataDir      string                      `json:"remote_data_dir"`
	RemoteLogDir       string                      `json:"remote_log_dir"`
	MysqlDataDir       string                      `json:"mysql_datadir"`
	UploadToMinio      bool                        `json:"upload_to_minio"`
	MysqldumpWorkDir   string                      `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string                    `json:"mysqldump_options"`
	MysqldumpExtraArgs string                      `json:"mysqldump_extra_args"`
	MysqldumpBin       string                      `json:"mysqldump_bin"`
	XtrabackupTool     string                      `json:"xtrabackup_tool"`
	XtrabackupBin      string                      `json:"xtrabackup_bin"`
	InnobackupexBin    string                      `json:"innobackupex_bin"`
	ScheduleEnabled    bool                        `json:"schedule_enabled"`
	CronSpec           string                      `json:"cron_spec"`
	LastScheduledAt    string                      `json:"last_scheduled_at,omitempty"`
	NotifyEnabled      bool                        `json:"notify_enabled"`
	NotifyUserIDs      []uint                      `json:"notify_user_ids"`
	NotifyUsers        []MysqlBackupNotifyUserItem `json:"notify_users,omitempty"`
	// HasMysqlPassword 是否已保存加密密码（不回显明文；编辑表单据此提示「已配置」）。
	HasMysqlPassword bool   `json:"has_mysql_password"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

type MysqlBackupInstanceUpsertRequest struct {
	ProjectID          uint     `json:"project_id"`
	ServerID           uint     `json:"server_id" binding:"required"`
	Name               string   `json:"name" binding:"required,max=128"`
	Enabled            *bool    `json:"enabled"`
	MysqlHost          string   `json:"mysql_host"`
	MysqlPort          int      `json:"mysql_port"`
	MysqlSocket        string   `json:"mysql_socket"`
	MysqlUser          string   `json:"mysql_user" binding:"required"`
	MysqlPassword      string   `json:"mysql_password"`
	BackupMode         string   `json:"backup_mode"`
	BackupScope        string   `json:"backup_scope"`
	DatabaseName       string   `json:"database_name"`
	TableName          string   `json:"table_name"`
	DatabaseNames      string   `json:"database_names"`
	RemoteDataDir      string   `json:"remote_data_dir"`
	RemoteLogDir       string   `json:"remote_log_dir"`
	MysqlDataDir       string   `json:"mysql_datadir"`
	UploadToMinio      *bool    `json:"upload_to_minio"`
	MysqldumpWorkDir   string   `json:"mysqldump_work_dir"`
	MysqldumpOptions   []string `json:"mysqldump_options"`
	MysqldumpExtraArgs string   `json:"mysqldump_extra_args"`
	MysqldumpBin       string   `json:"mysqldump_bin"`
	XtrabackupTool     string   `json:"xtrabackup_tool"`
	XtrabackupBin      string   `json:"xtrabackup_bin"`
	InnobackupexBin    string   `json:"innobackupex_bin"`
	ScheduleEnabled    *bool    `json:"schedule_enabled"`
	CronSpec           string   `json:"cron_spec"`
	NotifyEnabled      *bool    `json:"notify_enabled"`
	NotifyUserIDs      []uint   `json:"notify_user_ids"`
}

type MysqlBackupInstanceListQuery struct {
	ProjectID uint `form:"project_id"`
	Page      int  `form:"page"`
	PageSize  int  `form:"page_size"`
}

type MysqlBackupJobListQuery struct {
	ProjectID  uint `form:"project_id"`
	InstanceID uint `form:"instance_id"`
	Page       int  `form:"page"`
	PageSize   int  `form:"page_size"`
}

func (s *MysqlBackupService) toInstanceItem(ctx context.Context, inst model.MysqlBackupInstance) MysqlBackupInstanceItem {
	item := MysqlBackupInstanceItem{
		ID: inst.ID, ProjectID: inst.ProjectID, ServerID: inst.ServerID,
		Name: inst.Name, Enabled: inst.Enabled, MysqlHost: inst.MysqlHost, MysqlPort: inst.MysqlPort,
		MysqlSocket: inst.MysqlSocket, MysqlUser: inst.MysqlUser, BackupMode: inst.BackupMode,
		BackupScope: inst.BackupScope, DatabaseName: inst.DatabaseName, TableName: inst.BackupTable,
		DatabaseNames: inst.DatabaseNames, RemoteDataDir: inst.RemoteDataDir, RemoteLogDir: inst.RemoteLogDir,
		MysqlDataDir:  inst.MysqlDataDir,
		UploadToMinio: inst.UploadToMinio, MysqldumpWorkDir: inst.MysqldumpWorkDir,
		MysqldumpExtraArgs: inst.MysqldumpExtraArgs, MysqldumpBin: inst.MysqldumpBin,
		XtrabackupTool: inst.XtrabackupTool, XtrabackupBin: inst.XtrabackupBin, InnobackupexBin: inst.InnobackupexBin,
		ScheduleEnabled: inst.ScheduleEnabled, CronSpec: inst.CronSpec,
		NotifyEnabled:    inst.NotifyEnabled,
		HasMysqlPassword: strings.TrimSpace(inst.EncPassword) != "",
	}
	notifyIDs := parseNotifyUserIDs(inst.NotifyUserIDs)
	item.NotifyUserIDs = notifyIDs
	item.NotifyUsers = s.resolveNotifyUserBriefs(ctx, notifyIDs)
	item.MysqldumpOptions = parseMysqldumpOptionsForAPI(inst.MysqldumpOptions)
	if inst.LastScheduledAt != nil && !inst.LastScheduledAt.IsZero() {
		item.LastScheduledAt = inst.LastScheduledAt.Format(time.RFC3339)
	}
	if sv, err := s.serverRepo.GetByID(ctx, inst.ServerID); err == nil {
		item.ServerName = sv.Name
	}
	if !inst.CreatedAt.IsZero() {
		item.CreatedAt = inst.CreatedAt.Format(time.RFC3339)
	}
	if !inst.UpdatedAt.IsZero() {
		item.UpdatedAt = inst.UpdatedAt.Format(time.RFC3339)
	}
	return item
}

func (s *MysqlBackupService) ListMysqldumpOptions() []mysqlbackup.MysqldumpOption {
	return mysqlbackup.MysqldumpOptionCatalog
}

func marshalMysqldumpOptionIDs(ids []string) string {
	if len(ids) == 0 {
		bs, _ := json.Marshal(mysqlbackup.DefaultMysqldumpOptionIDs())
		return string(bs)
	}
	bs, err := json.Marshal(ids)
	if err != nil {
		return "[]"
	}
	return string(bs)
}

func parseMysqldumpOptionsForAPI(raw string) []string {
	ids, err := mysqlbackup.ParseMysqldumpOptionIDs(raw)
	if err != nil {
		return mysqlbackup.DefaultMysqldumpOptionIDs()
	}
	return ids
}
