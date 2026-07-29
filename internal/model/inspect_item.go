package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectItem 巡检项：project_id=0 为全局模板，>0 为项目覆盖/自建项。
type InspectItem struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID     uint    `json:"project_id" gorm:"not null;default:0;index;comment:项目ID，0=全局模板"`
	Type          string  `json:"type" gorm:"size:200;not null;index;comment:分类"`
	Name          string  `json:"name" gorm:"size:200;not null;comment:名称"`
	Description   string  `json:"description" gorm:"size:500;comment:说明"`
	Query         string  `json:"query" gorm:"type:text;not null;comment:PromQL"`
	Threshold     float64 `json:"threshold" gorm:"comment:阈值"`
	ThresholdType string  `json:"threshold_type" gorm:"size:50;comment:比较类型 greater/less/equal 等"`
	Unit          string  `json:"unit" gorm:"size:50;comment:单位"`
	LabelsJSON    string  `json:"labels_json" gorm:"type:text;comment:标签JSON"`
	Enabled       bool    `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	SortOrder     int     `json:"sort_order" gorm:"not null;default:0;comment:排序"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (InspectItem) TableName() string {
	return "inspect_items"
}
