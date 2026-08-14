package model

import (
	"time"

	"gorm.io/gorm"
)

// ClusterLogAgent 项目在某 K8s 集群上的日志采集 Agent（DaemonSet）登记。
type ClusterLogAgent struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	ProjectID      uint       `json:"project_id" gorm:"not null;uniqueIndex:uk_cluster_log_agent_proj_cluster;comment:项目ID"`
	ClusterID      uint       `json:"cluster_id" gorm:"not null;uniqueIndex:uk_cluster_log_agent_proj_cluster;index;comment:K8s集群ID"`
	Namespace       string     `json:"namespace" gorm:"size:128;not null;default:yunshu-logging;comment:DaemonSet 命名空间"`
	Status          string     `json:"status" gorm:"size:32;not null;default:unknown;comment:unknown|deployed|failed"`
	DeployRevision  int        `json:"deploy_revision" gorm:"not null;default:0"`
	DesiredReplicas int        `json:"desired_replicas" gorm:"default:0"`
	ReadyReplicas   int        `json:"ready_replicas" gorm:"default:0"`
	RateLimitQPS    int        `json:"rate_limit_qps" gorm:"not null;default:2000;comment:项目在该集群采集 QPS 限流"`
	PipelinesYAML   string     `json:"pipelines_yml,omitempty" gorm:"type:longtext;comment:自定义 pipelines.yml；空则按规则生成"`
	PipelinesCustom bool       `json:"pipelines_custom" gorm:"not null;default:false;comment:true=使用 PipelinesYAML 覆盖自动生成"`
	LastError       string     `json:"last_error" gorm:"size:1024"`
	LastSyncAt      *time.Time `json:"last_sync_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ClusterLogAgent) TableName() string { return "cluster_log_agents" }

// ClusterLogRule K8s 集群日志采集规则（按 ns/workload，不绑 server_id）。
type ClusterLogRule struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	ProjectID        uint           `json:"project_id" gorm:"not null;index;comment:项目ID"`
	ClusterID        uint           `json:"cluster_id" gorm:"not null;index;comment:K8s集群ID"`
	Name             string         `json:"name" gorm:"size:128;not null;comment:规则名"`
	MatchNamespaces  string         `json:"match_namespaces" gorm:"type:text;comment:JSON 字符串数组，空=默认排除系统 ns 后全采"`
	MatchWorkloads   string         `json:"match_workloads" gorm:"type:text;comment:JSON 字符串数组，匹配 pod 名前缀"`
	ExcludeNamespaces string         `json:"exclude_namespaces" gorm:"type:text;comment:JSON，默认 kube-system 等"`
	ParseProfile      string         `json:"parse_profile" gorm:"size:32;not null;default:cri;comment:解析档"`
	RateLimitQPS      int            `json:"rate_limit_qps" gorm:"not null;default:0;comment:规则级 QPS，0=用 Agent 项目级"`
	Enabled           bool           `json:"enabled" gorm:"not null;default:true;index"`
	Remark            string         `json:"remark" gorm:"size:255"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ClusterLogRule) TableName() string { return "cluster_log_rules" }
