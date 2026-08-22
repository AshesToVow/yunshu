package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertCurEvent 当前告警（firing 实例）。屏蔽命中不入库；resolved 时迁入 AlertHisEvent 并删除本行。
type AlertCurEvent struct {
	ID uint `json:"id" gorm:"primaryKey"`

	Fingerprint string `json:"fingerprint" gorm:"size:512;not null;uniqueIndex;comment:告警指纹"`
	Alertname   string `json:"alertname" gorm:"size:255;not null;index;comment:告警名"`
	Severity    string `json:"severity" gorm:"size:32;not null;default:warning;index"`
	Status      string `json:"status" gorm:"size:32;not null;default:firing;index"`
	Source      string `json:"source" gorm:"size:64;index;comment:来源 platform_monitor/k8s_event 等"`
	Receiver    string `json:"receiver" gorm:"size:128;comment:入站 receiver"`
	Cluster     string `json:"cluster" gorm:"size:128;index"`
	ProjectID   uint   `json:"project_id" gorm:"index"`
	DatasourceID uint  `json:"datasource_id" gorm:"index"`
	GroupKey    string `json:"group_key" gorm:"size:128;index"`

	LabelsJSON      string `json:"labels_json" gorm:"type:text"`
	AnnotationsJSON string `json:"annotations_json" gorm:"type:text"`
	Summary         string `json:"summary" gorm:"size:512"`
	Value           string `json:"value" gorm:"size:128;comment:触发时指标值快照"`

	StartsAt  time.Time `json:"starts_at" gorm:"index"`
	UpdatedAt time.Time `json:"updated_at" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}

func (AlertCurEvent) TableName() string { return "alert_cur_events" }

// AlertHisEvent 历史告警生命周期（从当前告警迁出的 resolved/closed 记录），与 alert_events 投递流水分离。
type AlertHisEvent struct {
	ID uint `json:"id" gorm:"primaryKey"`

	Fingerprint string `json:"fingerprint" gorm:"size:512;not null;index"`
	Alertname   string `json:"alertname" gorm:"size:255;not null;index"`
	Severity    string `json:"severity" gorm:"size:32;not null;default:warning;index"`
	Status      string `json:"status" gorm:"size:32;not null;default:resolved;index"`
	Source      string `json:"source" gorm:"size:64;index"`
	Receiver    string `json:"receiver" gorm:"size:128"`
	Cluster     string `json:"cluster" gorm:"size:128;index"`
	ProjectID   uint   `json:"project_id" gorm:"index"`
	DatasourceID uint  `json:"datasource_id" gorm:"index"`
	GroupKey    string `json:"group_key" gorm:"size:128;index"`

	LabelsJSON      string `json:"labels_json" gorm:"type:text"`
	AnnotationsJSON string `json:"annotations_json" gorm:"type:text"`
	Summary         string `json:"summary" gorm:"size:512"`
	Value           string `json:"value" gorm:"size:128"`

	StartsAt   time.Time      `json:"starts_at" gorm:"index"`
	ResolvedAt time.Time      `json:"resolved_at" gorm:"index"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertHisEvent) TableName() string { return "alert_his_events" }
