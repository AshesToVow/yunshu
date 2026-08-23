package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	AlertRuleChangePending  = "pending"
	AlertRuleChangeApproved = "approved"
	AlertRuleChangeRejected = "rejected"
)

// AlertMonitorRuleChangeRequest 监控规则变更审批。
type AlertMonitorRuleChangeRequest struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	RuleID      uint   `json:"rule_id" gorm:"not null;index"`
	ProposerID  uint   `json:"proposer_id" gorm:"not null;index"`
	ReviewerID  uint   `json:"reviewer_id" gorm:"index"`
	Status      string `json:"status" gorm:"size:32;not null;default:pending;index"`
	PayloadJSON string `json:"payload_json" gorm:"type:longtext;not null;comment:拟变更字段 JSON"`
	Comment     string `json:"comment" gorm:"type:text"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertMonitorRuleChangeRequest) TableName() string { return "alert_monitor_rule_change_requests" }
