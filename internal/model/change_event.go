package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ChangeSourceCicd   = "cicd"
	ChangeSourceK8s    = "k8s"
	ChangeSourceDbmgmt = "dbmgmt"
	ChangeSourceCmdb   = "cmdb"
	ChangeSourceAlert  = "alert"

	ChangeRiskLow    = "low"
	ChangeRiskMedium = "medium"
	ChangeRiskHigh   = "high"

	ChangeStatusStarted   = "started"
	ChangeStatusSucceeded = "succeeded"
	ChangeStatusFailed    = "failed"
	ChangeStatusAborted   = "aborted"
)

// ChangeEvent 统一变更流水（画像/变更中心/故障关联的时间线底座）。
type ChangeEvent struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	ProjectID    uint           `json:"project_id" gorm:"not null;index:idx_change_proj_time,priority:1"`
	ServiceID    *uint          `json:"service_id" gorm:"index;comment:service_catalog.id"`
	Source       string         `json:"source" gorm:"size:32;not null;index;comment:cicd|k8s|dbmgmt|cmdb|alert"`
	Action       string         `json:"action" gorm:"size:64;not null"`
	RiskLevel    string         `json:"risk_level" gorm:"size:16;not null;default:'medium'"`
	Status       string         `json:"status" gorm:"size:32;not null;default:'started';index"`
	ActorUserID  *uint          `json:"actor_user_id"`
	Summary      string         `json:"summary" gorm:"size:512;not null"`
	PayloadJSON  string         `json:"payload_json" gorm:"type:longtext"`
	StartedAt    time.Time      `json:"started_at" gorm:"index:idx_change_proj_time,priority:2"`
	FinishedAt   *time.Time     `json:"finished_at"`
	RollbackRef  string         `json:"rollback_ref" gorm:"size:256"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ChangeEvent) TableName() string { return "change_events" }
