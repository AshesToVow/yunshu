package model

import "time"

// LogPattern 日志模板聚类（签名 + 命中统计）。
type LogPattern struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ProjectID   uint      `json:"project_id" gorm:"not null;index:idx_log_pattern_proj_sig,priority:1"`
	Signature   string    `json:"signature" gorm:"size:512;not null;index:idx_log_pattern_proj_sig,priority:2"`
	Sample      string    `json:"sample" gorm:"type:text"`
	Level       string    `json:"level" gorm:"size:16;index"`
	ServiceName string    `json:"service_name" gorm:"size:128;index"`
	HitCount    int64     `json:"hit_count" gorm:"not null;default:0"`
	FirstSeenAt time.Time `json:"first_seen_at" gorm:"index"`
	LastSeenAt  time.Time `json:"last_seen_at" gorm:"index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (LogPattern) TableName() string { return "log_patterns" }
