package model

import (
	"time"

	"gorm.io/gorm"
)

// K8sCrTemplate CR/YAML 模板库（project_id=0 为全局）。
type K8sCrTemplate struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ProjectID uint   `json:"project_id" gorm:"not null;default:0;index"`
	Name      string `json:"name" gorm:"size:128;not null;index"`
	GVKGroup  string `json:"gvk_group" gorm:"size:128"`
	GVKVersion string `json:"gvk_version" gorm:"size:32;not null;default:v1"`
	GVKKind   string `json:"gvk_kind" gorm:"size:64;not null;index"`
	Body      string `json:"body" gorm:"type:longtext;not null"`
	SortOrder int    `json:"sort_order" gorm:"not null;default:0"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (K8sCrTemplate) TableName() string { return "k8s_cr_templates" }
