package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ImageRegistryTypeHarbor         = "harbor"
	ImageRegistryTypeDockerRegistry = "docker_registry"
)

// ImageRegistry 镜像仓库注册中心。
type ImageRegistry struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Name           string         `json:"name" gorm:"size:128;not null;uniqueIndex"`
	Type           string         `json:"type" gorm:"size:32;not null;default:'harbor';comment:harbor|docker_registry"`
	URL            string         `json:"url" gorm:"size:256;not null;comment:不含协议的主机名或带协议 URL"`
	HostIP         string         `json:"host_ip" gorm:"size:64;comment:可选解析 IP（Harbor hostAliases）"`
	Username       string         `json:"username" gorm:"size:128"`
	Password       string         `json:"-" gorm:"size:256;comment:明文密码（与字典 cicd_harbor_password 同级敏感）"`
	DefaultProject string         `json:"default_project" gorm:"size:128;comment:默认 Harbor project"`
	IsDefault      bool           `json:"is_default" gorm:"not null;default:false;index"`
	Status         int            `json:"status" gorm:"not null;default:1;comment:1启用 0停用"`
	Remark         string         `json:"remark" gorm:"size:512"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ImageRegistry) TableName() string { return "image_registries" }

// ProjectRegistryBinding 项目绑定的镜像仓库（覆盖全局默认）。
type ProjectRegistryBinding struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	ProjectID     uint           `json:"project_id" gorm:"not null;uniqueIndex"`
	RegistryID    uint           `json:"registry_id" gorm:"not null;index"`
	HarborProject string         `json:"harbor_project" gorm:"size:128;comment:项目级 Harbor project 覆盖"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ProjectRegistryBinding) TableName() string { return "project_registry_bindings" }

// ImageCleanupPolicy 镜像 Tag 清理策略。
type ImageCleanupPolicy struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	RegistryID    uint           `json:"registry_id" gorm:"not null;index"`
	HarborProject string         `json:"harbor_project" gorm:"size:128;comment:空=全部 project"`
	KeepLastN     int            `json:"keep_last_n" gorm:"not null;default:10"`
	RetainDays    int            `json:"retain_days" gorm:"not null;default:30"`
	Enabled       bool           `json:"enabled" gorm:"not null;default:true"`
	CronSpec      string         `json:"cron_spec" gorm:"size:64;default:'0 3 * * *'"`
	LastRunAt     *time.Time     `json:"last_run_at,omitempty"`
	LastResult    string         `json:"last_result" gorm:"size:1024"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ImageCleanupPolicy) TableName() string { return "image_cleanup_policies" }

// CicdPipelineTemplate 多语言流水线模板元数据（指向 Jenkinsfile Script Path）。
type CicdPipelineTemplate struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	LanguageType string         `json:"language_type" gorm:"size:32;not null;uniqueIndex;comment:go|java|frontend|python|custom"`
	Name         string         `json:"name" gorm:"size:128;not null"`
	ScriptPath   string         `json:"script_path" gorm:"size:256;not null"`
	Description  string         `json:"description" gorm:"size:512"`
	Sort         int            `json:"sort" gorm:"not null;default:0"`
	Status       int            `json:"status" gorm:"not null;default:1"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

func (CicdPipelineTemplate) TableName() string { return "cicd_pipeline_templates" }

const (
	CicdLanguageGo       = "go"
	CicdLanguageJava     = "java"
	CicdLanguageFrontend = "frontend"
	CicdLanguagePython   = "python"
	CicdLanguageCustom   = "custom"
)
