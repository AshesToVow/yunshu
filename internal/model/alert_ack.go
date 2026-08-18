package model

import "time"

// AlertAck 告警认领：有效期内同指纹 firing 通知抑制（仍保留当前告警台账）。
type AlertAck struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Fingerprint string    `json:"fingerprint" gorm:"size:512;not null;index:idx_alert_ack_fp_exp,priority:1;comment:告警指纹"`
	UserID      uint      `json:"user_id" gorm:"not null;index;comment:认领人用户ID"`
	UserName    string    `json:"user_name" gorm:"size:64;comment:认领人显示名"`
	ExpiresAt   time.Time `json:"expires_at" gorm:"not null;index:idx_alert_ack_fp_exp,priority:2;comment:认领失效时间"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertAck) TableName() string { return "alert_acks" }
