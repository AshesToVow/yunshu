package model

import (
	"time"

	"gorm.io/gorm"
)

// LogPipeline Loggie pipelines.yml 仓库条目（可版本化编辑、AI 调整、下发）。
type LogPipeline struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	ProjectID     uint           `json:"project_id" gorm:"not null;index;uniqueIndex:uk_log_pipeline_proj_name;comment:项目ID"`
	Name          string         `json:"name" gorm:"size:128;not null;uniqueIndex:uk_log_pipeline_proj_name;comment:仓库内名称"`
	Kind          string         `json:"kind" gorm:"size:16;not null;default:k8s;index;comment:host|k8s|template"`
	ClusterID     uint           `json:"cluster_id" gorm:"not null;default:0;index;comment:K8s 集群 ID，host 时为 0"`
	ServerID      uint           `json:"server_id" gorm:"not null;default:0;index;comment:主机服务器 ID，k8s 时为 0"`
	ParseProfile  string         `json:"parse_profile" gorm:"size:64;comment:建议解析档"`
	ContentYAML   string         `json:"content_yml" gorm:"type:longtext;comment:pipelines.yml 内容"`
	Status        string         `json:"status" gorm:"size:16;not null;default:draft;index;comment:draft|published"`
	Version       int            `json:"version" gorm:"not null;default:1"`
	SourceRef     string         `json:"source_ref" gorm:"size:128;comment:来源引用，如 cluster_log_agent:12"`
	Remark        string         `json:"remark" gorm:"size:255"`
	UpdatedBy     uint           `json:"updated_by" gorm:"not null;default:0"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (LogPipeline) TableName() string { return "log_pipelines" }
