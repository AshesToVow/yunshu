package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectReportTemplate 巡检报告版式模板（与巡检项模板分离）。
// project_id=0 为全局内置/全局自定义；>0 为项目模板。
type InspectReportTemplate struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint   `json:"project_id" gorm:"not null;default:0;index;uniqueIndex:uk_inspect_report_tpl_proj_code;comment:项目ID,0=全局"`
	Code      string `json:"code" gorm:"size:64;not null;uniqueIndex:uk_inspect_report_tpl_proj_code;comment:模板编码"`
	Name      string `json:"name" gorm:"size:128;not null;comment:展示名称"`
	Engine    string `json:"engine" gorm:"size:32;not null;default:go_html;comment:渲染引擎"`
	// Body 自定义模板全文；内置且为空时使用 embed 文件 templates/{code}.html
	Body      string `json:"body" gorm:"type:longtext;comment:HTML模板正文"`
	IsBuiltin bool   `json:"is_builtin" gorm:"not null;default:false;comment:是否内置"`
	Status    int    `json:"status" gorm:"not null;default:1;index;comment:1启用0停用"`
	Remark    string `json:"remark" gorm:"size:255;comment:备注"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (InspectReportTemplate) TableName() string {
	return "inspect_report_templates"
}
