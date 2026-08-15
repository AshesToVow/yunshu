package model

import (
	"time"

	"gorm.io/gorm"
)

// AiToolApproval AI 高危工具审批单。
type AiToolApproval struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	ToolName    string         `json:"tool_name" gorm:"size:64;not null;index"`
	ArgsJSON    string         `json:"args_json" gorm:"type:text"`
	ClusterID   uint           `json:"cluster_id" gorm:"index"`
	Namespace   string         `json:"namespace" gorm:"size:128"`
	Resource    string         `json:"resource" gorm:"size:256"`
	Reason      string         `json:"reason" gorm:"size:512"`
	Status      string         `json:"status" gorm:"size:32;not null;default:pending;index"` // pending|approved|rejected|executing|executed|failed
	ReviewerID  *uint          `json:"reviewer_id,omitempty"`
	ReviewNote  string         `json:"review_note" gorm:"size:512"`
	ResultMsg   string         `json:"result_msg" gorm:"size:1024"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiToolApproval) TableName() string { return "ai_tool_approvals" }

// EsmgmtConnection Elasticsearch 管理连接。
type EsmgmtConnection struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	Name         string         `json:"name" gorm:"size:128;not null"`
	Addresses    string         `json:"addresses" gorm:"size:1024;not null;comment:逗号分隔"`
	Username     string         `json:"username" gorm:"size:128"`
	PasswordEnc  string         `json:"-" gorm:"column:password_enc;size:512"`
	HasPassword  bool           `json:"has_password" gorm:"-"`
	TimeoutSec   int            `json:"timeout_sec" gorm:"not null;default:30"`
	IsDefault    bool           `json:"is_default" gorm:"not null;default:false"`
	Remark       string         `json:"remark" gorm:"size:255"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (EsmgmtConnection) TableName() string { return "esmgmt_connections" }

// EsmgmtBackupJob 索引备份任务（分词/settings → mapping → 数据 → MinIO）。
type EsmgmtBackupJob struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	ConnectionID   uint           `json:"connection_id" gorm:"index"`
	IndexName      string         `json:"index_name" gorm:"size:256;not null;index"`
	Trigger        string         `json:"trigger" gorm:"size:32;not null;default:manual"` // manual|scheduled
	Status         string         `json:"status" gorm:"size:32;not null;default:pending;index"` // pending|running|success|failed
	Phase          string         `json:"phase" gorm:"size:64"`                                  // analysis|mapping|data|upload
	DocCount       int            `json:"doc_count"`
	MinioBucket    string         `json:"minio_bucket" gorm:"size:128"`
	MinioObject    string         `json:"minio_object" gorm:"size:512"` // zip 相对对象键
	AnalysisObject string         `json:"analysis_object" gorm:"size:512"`
	MappingObject  string         `json:"mapping_object" gorm:"size:512"`
	DataObject     string         `json:"data_object" gorm:"size:512"`
	ErrorMessage   string         `json:"error_message" gorm:"size:1024"`
	CreatedBy      uint           `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (EsmgmtBackupJob) TableName() string { return "esmgmt_backup_jobs" }

// EsmgmtBackupSchedule 索引定时备份规则。
type EsmgmtBackupSchedule struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ConnectionID    uint           `json:"connection_id" gorm:"not null;index"`
	IndexName       string         `json:"index_name" gorm:"size:256;not null;index"`
	Enabled         bool           `json:"enabled" gorm:"not null;default:true;index"`
	CronSpec        string         `json:"cron_spec" gorm:"size:256;not null"`
	MaxDocs         int            `json:"max_docs" gorm:"not null;default:0"`
	LastScheduledAt *time.Time     `json:"last_scheduled_at,omitempty"`
	Remark          string         `json:"remark" gorm:"size:255"`
	CreatedBy       uint           `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (EsmgmtBackupSchedule) TableName() string { return "esmgmt_backup_schedules" }

// EsmgmtRestoreJob 从 MinIO 备份恢复索引。
type EsmgmtRestoreJob struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	BackupJobID    uint           `json:"backup_job_id" gorm:"not null;index"`
	ConnectionID   uint           `json:"connection_id" gorm:"index"`
	SourceIndex    string         `json:"source_index" gorm:"size:256"`
	TargetIndex    string         `json:"target_index" gorm:"size:256;not null"`
	DeleteExisting bool           `json:"delete_existing" gorm:"not null;default:false"`
	Status         string         `json:"status" gorm:"size:32;not null;default:pending;index"`
	Phase          string         `json:"phase" gorm:"size:64"`
	DocCount       int            `json:"doc_count"`
	ErrorMessage   string         `json:"error_message" gorm:"size:1024"`
	CreatedBy      uint           `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (EsmgmtRestoreJob) TableName() string { return "esmgmt_restore_jobs" }
