package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertConsulEndpoint Consul 连接（按项目），用于同步监控对象目录。
type AlertConsulEndpoint struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	ProjectID uint   `json:"project_id" gorm:"not null;index;comment:所属项目ID"`
	Name      string `json:"name" gorm:"size:128;not null;comment:显示名称"`
	Address   string `json:"address" gorm:"size:512;not null;comment:Consul HTTP 地址，如 http://consul:8500"`
	Token     string `json:"token,omitempty" gorm:"type:text;comment:ACL Token"`
	Datacenter string `json:"datacenter" gorm:"size:64;comment:数据中心，空则默认"`

	// ServiceTag 非空时只同步带该 tag 的服务（推荐 yunshu-metrics）
	ServiceTag string `json:"service_tag" gorm:"size:128;default:yunshu-metrics;comment:服务 tag 过滤"`
	Enabled    bool   `json:"enabled" gorm:"not null;default:true;index;comment:是否启用"`
	Remark     string `json:"remark" gorm:"size:512;comment:备注"`

	LastSyncAt *time.Time `json:"last_sync_at,omitempty" gorm:"comment:上次同步时间"`
	LastError  string     `json:"last_error,omitempty" gorm:"size:1024;comment:上次同步错误"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertConsulEndpoint) TableName() string {
	return "alert_consul_endpoints"
}

// AlertMonitorObject 从 Consul 同步的监控对象（只读目录，不负责 scrape）。
type AlertMonitorObject struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	EndpointID uint `json:"endpoint_id" gorm:"not null;index;uniqueIndex:uk_alert_mon_obj,priority:1;comment:Consul 端点ID"`
	ProjectID  uint `json:"project_id" gorm:"not null;index;comment:所属项目ID"`

	ServiceName string `json:"service_name" gorm:"size:128;not null;uniqueIndex:uk_alert_mon_obj,priority:2;comment:Consul 服务名"`
	ServiceID   string `json:"service_id" gorm:"size:256;not null;uniqueIndex:uk_alert_mon_obj,priority:3;comment:Consul 服务实例ID"`
	Node        string `json:"node" gorm:"size:128;comment:节点名"`
	Address     string `json:"address" gorm:"size:256;comment:服务地址"`
	Port        int    `json:"port" gorm:"comment:服务端口"`

	TagsJSON       string `json:"tags_json" gorm:"type:text;comment:tags JSON 数组"`
	MetaJSON       string `json:"meta_json" gorm:"type:text;comment:meta JSON 对象"`
	ExporterRole   string `json:"exporter_role" gorm:"size:64;index;comment:来自 meta.exporter_role"`
	YunshuProject  string `json:"yunshu_project" gorm:"size:128;index;comment:来自 meta.yunshu_project"`
	Health         string `json:"health" gorm:"size:32;comment:passing/warning/critical"`
	ProbeURL       string `json:"probe_url" gorm:"size:512;comment:拨测 URL（meta.probe_url）"`

	SyncedAt time.Time `json:"synced_at" gorm:"index;comment:本轮同步时间"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertMonitorObject) TableName() string {
	return "alert_monitor_objects"
}
