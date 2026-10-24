package model

import "time"

const (
	LogAnomalyTypeNewPattern = "new_pattern"
	LogAnomalyTypeErrorSpike = "error_spike"

	LogAnomalyStatusOpen     = "open"
	LogAnomalyStatusAck      = "acknowledged"
	LogAnomalyStatusResolved = "resolved"
	LogAnomalyStatusMuted    = "muted"

	LogAnomalySeverityWarning  = "warning"
	LogAnomalySeverityCritical = "critical"
)

// LogAnomaly 日志异常 / Error Tracking 问题对象。
type LogAnomaly struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	ProjectID    uint       `json:"project_id" gorm:"not null;index:idx_log_anomaly_proj_status,priority:1"`
	PatternID    *uint      `json:"pattern_id" gorm:"index"`
	AnomalyType  string     `json:"anomaly_type" gorm:"size:32;not null;index"`
	Signature    string     `json:"signature" gorm:"size:512"`
	Title        string     `json:"title" gorm:"size:256;not null"`
	Detail       string     `json:"detail" gorm:"type:text"`
	Severity     string     `json:"severity" gorm:"size:16;not null;default:'warning';index"`
	Status       string     `json:"status" gorm:"size:32;not null;default:'open';index:idx_log_anomaly_proj_status,priority:2"`
	AssigneeID   uint       `json:"assignee_id" gorm:"not null;default:0;index;comment:负责人用户ID"`
	AssigneeName string     `json:"assignee_name" gorm:"size:64;comment:负责人显示名"`
	MutedUntil   *time.Time `json:"muted_until" gorm:"comment:静默截止时间"`
	MetadataJSON string     `json:"metadata_json" gorm:"type:text"`
	DetectedAt   time.Time  `json:"detected_at" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (LogAnomaly) TableName() string { return "log_anomalies" }
