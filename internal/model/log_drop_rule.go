package model

import "time"

const (
	LogDropOpEq       = "eq"
	LogDropOpContains = "contains"
)

// LogDropRule 日志查询侧黑名单（检索 must_not；不改写入链路）。
type LogDropRule struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ProjectID uint      `json:"project_id" gorm:"not null;index:idx_log_drop_proj_en,priority:1;comment:项目ID"`
	Name      string    `json:"name" gorm:"size:128;not null;comment:规则名称"`
	Enabled   bool      `json:"enabled" gorm:"not null;default:1;index:idx_log_drop_proj_en,priority:2;comment:是否启用"`
	Field     string    `json:"field" gorm:"size:64;not null;comment:匹配字段 level/service_name/host/pod/message/..."`
	Operator  string    `json:"operator" gorm:"size:16;not null;default:eq;comment:eq|contains"`
	Value     string    `json:"value" gorm:"size:512;not null;comment:匹配值"`
	Remark    string    `json:"remark" gorm:"size:256"`
	CreatedBy uint      `json:"created_by" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LogDropRule) TableName() string { return "log_drop_rules" }
