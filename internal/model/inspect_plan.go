package model

import (
	"time"

	"gorm.io/gorm"
)

// InspectPlan 项目级 Prometheus 巡检计划（每项目一条）。
type InspectPlan struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID    uint   `json:"project_id" gorm:"not null;uniqueIndex;comment:所属项目ID"`
	Enabled      bool   `json:"enabled" gorm:"not null;default:false;index;comment:是否启用定时巡检"`
	CronSpec     string `json:"cron_spec" gorm:"size:100;comment:Cron 表达式"`
	DatasourceID uint   `json:"datasource_id" gorm:"not null;default:0;index;comment:主 Prometheus 数据源ID"`
	// ReportListMode: abnormal_only | summary | all
	ReportListMode string `json:"report_list_mode" gorm:"size:32;not null;default:abnormal_only;comment:报告明细模式"`
	// ReportTemplateID 报告版式模板；0 表示使用全局 default
	ReportTemplateID uint `json:"report_template_id" gorm:"not null;default:0;index;comment:报告版式模板ID"`
	// RetainDays 报告保留天数，0=不自动清理
	RetainDays int `json:"retain_days" gorm:"not null;default:90;comment:报告保留天数"`
	// RecipientsJSON 邮件收件人 JSON 数组，如 ["a@x.com"]
	RecipientsJSON string     `json:"recipients_json" gorm:"type:text;comment:邮件收件人JSON"`
	LastRunAt      *time.Time `json:"last_run_at" gorm:"comment:最近一次执行时间"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (InspectPlan) TableName() string {
	return "inspect_plans"
}
