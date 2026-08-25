package model

import (
	"time"

	"gorm.io/gorm"
)

// PlatformSavedQuery 用户 PromQL / 查询收藏。
type PlatformSavedQuery struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	UserID       uint   `json:"user_id" gorm:"not null;index:idx_saved_query_user"`
	Name         string `json:"name" gorm:"size:128;not null"`
	Query        string `json:"query" gorm:"type:text;not null"`
	DatasourceID uint   `json:"datasource_id" gorm:"index"`
	Kind         string `json:"kind" gorm:"size:32;not null;default:instant;comment:instant|range"`
	ProjectID    uint   `json:"project_id" gorm:"index;comment:可选项目上下文"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (PlatformSavedQuery) TableName() string { return "platform_saved_queries" }
