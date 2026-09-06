package model

import (
	"time"

	"gorm.io/gorm"
)

// LogPipelineVersion Pipeline 仓库历史版本快照。
type LogPipelineVersion struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	PipelineID  uint           `json:"pipeline_id" gorm:"not null;index;comment:log_pipelines.id"`
	ProjectID   uint           `json:"project_id" gorm:"not null;index"`
	Version     int            `json:"version" gorm:"not null"`
	ContentYAML string         `json:"content_yml" gorm:"type:longtext"`
	Remark      string         `json:"remark" gorm:"size:255"`
	CreatedBy   uint           `json:"created_by" gorm:"not null;default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (LogPipelineVersion) TableName() string { return "log_pipeline_versions" }
