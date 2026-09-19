package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertDatasource 告警数据源：Prometheus / VictoriaMetrics（PromQL 兼容 API）。
type AlertDatasource struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint `json:"project_id" gorm:"not null;index;index:idx_alert_ds_proj_enabled,priority:1;comment:所属项目ID"`

	Name string `json:"name" gorm:"size:128;not null;index;comment:显示名称"`
	Type string `json:"type" gorm:"size:32;not null;default:prometheus;index;comment:类型 prometheus|victoria"`

	BaseURL         string `json:"base_url" gorm:"size:512;not null;comment:API 根地址，如 http://prom:9090 或 http://vmselect:8481/select/0/prometheus"`
	AlertmanagerURL string `json:"alertmanager_url,omitempty" gorm:"size:512;comment:已废弃，保留兼容列"`
	BearerToken     string `json:"bearer_token,omitempty" gorm:"type:text;comment:Bearer Token"`
	BasicUser       string `json:"basic_user,omitempty" gorm:"size:128;comment:Basic 用户名"`
	BasicPassword   string `json:"basic_password,omitempty" gorm:"size:256;comment:Basic 密码"`
	SkipTLSVerify   bool   `json:"skip_tls_verify" gorm:"not null;default:false;comment:跳过 TLS 校验（仅内网调试）"`

	Enabled bool   `json:"enabled" gorm:"not null;default:true;index;index:idx_alert_ds_proj_enabled,priority:2;comment:是否启用"`
	Remark  string `json:"remark" gorm:"size:512;comment:备注"`

	// 采集健康（平台侧探测缓存；非 scrape 本身）
	LastHealthStatus    string     `json:"last_health_status" gorm:"size:16;not null;default:unknown;index;comment:ok|degraded|down|unknown"`
	LastHealthAt        *time.Time `json:"last_health_at" gorm:"comment:最近健康探测时间"`
	LastHealthLatencyMs int64      `json:"last_health_latency_ms" gorm:"not null;default:0;comment:探测延迟毫秒"`
	LastHealthError     string     `json:"last_health_error" gorm:"size:512;comment:最近探测错误摘要"`
	LastUpTotal         int64      `json:"last_up_total" gorm:"not null;default:0;comment:count(up) 快照"`
	LastUpDown          int64      `json:"last_up_down" gorm:"not null;default:0;comment:count(up==0) 快照"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

const (
	DatasourceHealthOK       = "ok"
	DatasourceHealthDegraded = "degraded"
	DatasourceHealthDown     = "down"
	DatasourceHealthUnknown  = "unknown"
)

func (AlertDatasource) TableName() string {
	return "alert_datasources"
}
