package model

import "time"

// LoggieAgent 项目服务器上的 Loggie 采集 Agent 登记与心跳。
type LoggieAgent struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	ProjectID       uint       `json:"project_id" gorm:"not null;uniqueIndex:uk_loggie_agent_proj_server;comment:项目ID"`
	ServerID        uint       `json:"server_id" gorm:"not null;uniqueIndex:uk_loggie_agent_proj_server;comment:服务器ID"`
	Token           string     `json:"-" gorm:"size:128;not null;index;comment:心跳鉴权令牌"`
	Version         string     `json:"version" gorm:"size:64;comment:Loggie 版本"`
	HealthStatus    string     `json:"health_status" gorm:"size:32;default:unknown;comment:running/stopped/error"`
	PipelineStatus  string     `json:"pipeline_status" gorm:"size:32;default:unknown;comment:pipeline 状态"`
	EsSinkOK        bool       `json:"es_sink_ok" gorm:"default:false;comment:ES Sink 是否可用"`
	LinesPerMin     int        `json:"lines_per_min" gorm:"default:0;comment:近周期上报行数/分钟"`
	LastError       string     `json:"last_error" gorm:"type:text;comment:最近错误"`
	MonitorPort     int        `json:"monitor_port" gorm:"default:9196;comment:Loggie HTTP 监控端口"`
	MonitorReachable bool      `json:"monitor_reachable" gorm:"default:false;comment:监控端口是否可达"`
	ActivePipelineCount int    `json:"active_pipeline_count" gorm:"default:0;comment:活跃 pipeline 数"`
	ActiveFdCount   int        `json:"active_fd_count" gorm:"default:0;comment:活跃文件句柄数"`
	InactiveFdCount int        `json:"inactive_fd_count" gorm:"default:0;comment:不活跃文件句柄数(追平后常见)"`
	MonitorDetail   string     `json:"monitor_detail" gorm:"type:text;comment:监控快照 JSON"`
	BootstrapConfig string     `json:"-" gorm:"type:text;comment:引导参数 JSON 便于重新生成 pipeline"`
	LastSeenAt      *time.Time `json:"last_seen_at" gorm:"index;comment:最近心跳"`
	LastIngestAt    *time.Time `json:"last_ingest_at" gorm:"comment:最近确认有日志写入 ES"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (LoggieAgent) TableName() string { return "loggie_agents" }
