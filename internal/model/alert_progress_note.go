package model

import "time"

// AlertProgressNote 告警备注/进展，按指纹追加，恢复后仍可查看。
type AlertProgressNote struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Fingerprint string    `json:"fingerprint" gorm:"size:512;not null;index:idx_alert_note_fp;comment:告警指纹"`
	UserID      uint      `json:"user_id" gorm:"not null;index;comment:作者用户ID"`
	UserName    string    `json:"user_name" gorm:"size:64;comment:作者显示名"`
	Content     string    `json:"content" gorm:"type:text;not null;comment:进展内容"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertProgressNote) TableName() string { return "alert_progress_notes" }
