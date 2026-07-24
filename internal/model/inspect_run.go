package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectRun 一次巡检执行记录（摘要入 DB，明细在报告文件）。
type InspectRun struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index;comment:项目ID"`
	PlanID    uint `json:"plan_id" gorm:"not null;index;comment:计划ID"`

	// Status: pending|running|success|failed
	Status string `json:"status" gorm:"size:32;not null;default:pending;index;comment:状态"`
	// Trigger: manual|cron
	Trigger string `json:"trigger" gorm:"size:32;not null;default:manual;comment:触发方式"`

	DatasourceID   uint    `json:"datasource_id" gorm:"comment:数据源ID"`
	DatasourceName string  `json:"datasource_name" gorm:"size:128;comment:数据源名称"`
	Score          float64 `json:"score" gorm:"comment:健康分0-100"`
	Grade          string  `json:"grade" gorm:"size:16;comment:等级"`
	Summary        string  `json:"summary" gorm:"type:text;comment:摘要"`
	ErrorMessage   string  `json:"error_message" gorm:"type:text;comment:失败信息"`

	TotalCount    int `json:"total_count" gorm:"comment:样本总数"`
	CriticalCount int `json:"critical_count" gorm:"comment:严重数"`
	WarningCount  int `json:"warning_count" gorm:"comment:警告数"`
	NormalCount   int `json:"normal_count" gorm:"comment:正常数"`

	ReportHTMLPath string `json:"report_html_path" gorm:"size:512;comment:HTML报告路径或对象键"`
	ReportPDFPath  string `json:"report_pdf_path" gorm:"size:512;comment:PDF报告路径或对象键"`
	EmailSentAt    *time.Time `json:"email_sent_at" gorm:"comment:邮件发送时间"`

	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	CreatedBy  uint       `json:"created_by" gorm:"comment:手动触发用户ID"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (InspectRun) TableName() string {
	return "inspect_runs"
}
