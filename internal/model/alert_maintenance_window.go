package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertMaintenanceWindow 维护窗口：命中 matchers 的告警在窗口期内抑制投递（与静默独立配置）。
type AlertMaintenanceWindow struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name         string    `json:"name" gorm:"size:128;not null;comment:维护窗口名称"`
	MatchersJSON string    `json:"matchers_json" gorm:"type:text;not null;comment:匹配器 JSON"`
	StartsAt     time.Time `json:"starts_at" gorm:"index;comment:开始时间"`
	EndsAt       time.Time `json:"ends_at" gorm:"index;comment:结束时间"`
	Comment      string    `json:"comment" gorm:"size:512;comment:说明"`
	CreatedBy    uint      `json:"created_by" gorm:"index;comment:创建人"`
	ProjectID    uint      `json:"project_id" gorm:"not null;default:0;index;comment:项目ID，0 表示全局"`
	Enabled      bool      `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertMaintenanceWindow) TableName() string {
	return "alert_maintenance_windows"
}
