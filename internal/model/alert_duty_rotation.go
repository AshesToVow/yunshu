package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertDutyRotation 值班轮换配置：按 period_minutes 在 participants 之间轮转，
// 由后台任务物化为 alert_duty_blocks（rotation_id 标记），复用既有通知解析与日历。
type AlertDutyRotation struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	MonitorRuleID     uint      `json:"monitor_rule_id" gorm:"not null;index;comment:监控规则ID"`
	Name              string    `json:"name" gorm:"size:128;not null;comment:轮换名称"`
	ParticipantsJSON  string    `json:"participants_json" gorm:"type:text;comment:参与轮换的用户 ID JSON 数组（按轮转顺序）"`
	DepartmentIDsJSON string    `json:"department_ids_json" gorm:"type:text;comment:部门根 ID JSON 数组（子树全员附加）"`
	ExtraEmailsJSON   string    `json:"extra_emails_json" gorm:"type:text;comment:额外邮箱 JSON"`
	AnchorAt          time.Time `json:"anchor_at" gorm:"not null;comment:轮换起算时间（第一班开始）"`
	PeriodMinutes     int       `json:"period_minutes" gorm:"not null;default:1440;comment:每班时长（分钟）"`
	Enabled           bool      `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`

	Remark string `json:"remark" gorm:"size:512;comment:备注"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertDutyRotation) TableName() string {
	return "alert_duty_rotations"
}
