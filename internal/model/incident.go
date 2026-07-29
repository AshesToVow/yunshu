package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	IncidentStatusOpen        = "open"
	IncidentStatusMitigating  = "mitigating"
	IncidentStatusResolved    = "resolved"
	IncidentStatusPostmortem  = "postmortem"

	IncidentSeverityP1 = "p1"
	IncidentSeverityP2 = "p2"
)

// Incident 故障单（工作台实体，关联告警与变更时间线）。
type Incident struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	ProjectID       uint           `json:"project_id" gorm:"not null;index:idx_incident_proj_status,priority:1"`
	ServiceID       *uint          `json:"service_id" gorm:"index;comment:service_catalog.id"`
	Title           string         `json:"title" gorm:"size:256;not null"`
	Severity        string         `json:"severity" gorm:"size:16;not null;default:'p1';index"`
	Status          string         `json:"status" gorm:"size:32;not null;default:'open';index:idx_incident_proj_status,priority:2"`
	Summary         string         `json:"summary" gorm:"size:1024"`
	AlertFingerprint string        `json:"alert_fingerprint" gorm:"size:256;index"`
	AssigneeUserID  *uint          `json:"assignee_user_id"`
	OpenedBy        *uint          `json:"opened_by"`
	AcknowledgedAt  *time.Time     `json:"acknowledged_at"`
	ResolvedAt      *time.Time     `json:"resolved_at"`
	MTTASeconds     *int64         `json:"mtta_seconds"`
	MTTRSeconds     *int64         `json:"mttr_seconds"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Incident) TableName() string { return "incidents" }

// IncidentNote 故障处置备注。
type IncidentNote struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	IncidentID uint      `json:"incident_id" gorm:"not null;index"`
	AuthorID   uint      `json:"author_id"`
	Body       string    `json:"body" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
}

func (IncidentNote) TableName() string { return "incident_notes" }
