package model

import "time"

// LogRetentionPolicy 日志保留策略（project_id=0 为全局默认，>0 为项目覆盖）。
type LogRetentionPolicy struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ProjectID       uint      `json:"project_id" gorm:"not null;index:uk_log_retention_scope,priority:1;comment:0=全局默认"`
	ServerID        uint      `json:"server_id" gorm:"not null;default:0;index:uk_log_retention_scope,priority:2;comment:0=项目内全部服务器"`
	RetentionDays   int       `json:"retention_days" gorm:"not null;default:30;comment:保留天数"`
	MaxStoreBytes   int64     `json:"max_store_bytes" gorm:"default:0;comment:ES 存储上限字节，0=不限制"`
	MaxIndexCount   int       `json:"max_index_count" gorm:"default:0;comment:最大索引个数，0=不限制"`
	Enabled         bool      `json:"enabled" gorm:"not null;default:true"`
	IndexPattern    string    `json:"index_pattern" gorm:"size:128;comment:可选索引模式覆盖"`
	Remark          *string   `json:"remark,omitempty" gorm:"size:255"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (LogRetentionPolicy) TableName() string { return "log_retention_policies" }
