package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	PlatformTemplateCategoryCicdSnippet = "cicd_snippet"
	PlatformTemplateCategoryAlert       = "alert"
	PlatformTemplateCategoryInspect     = "inspect"
	PlatformTemplateCategoryLoggie      = "loggie"

	PlatformTemplateFormatShell      = "shell"
	PlatformTemplateFormatYAML       = "yaml"
	PlatformTemplateFormatHTML       = "html"
	PlatformTemplateFormatText       = "text"
	PlatformTemplateFormatGoTemplate = "go_template"

	PlatformTemplateStatusEnabled  = 1
	PlatformTemplateStatusDisabled = 0
)

// PlatformTemplate 平台模板目录（稳定 template_key 供业务引用）。
type PlatformTemplate struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	TemplateKey      string         `json:"template_key" gorm:"size:128;not null;uniqueIndex;comment:稳定引用键"`
	Category         string         `json:"category" gorm:"size:32;not null;index;comment:cicd_snippet|alert|inspect|loggie"`
	Name             string         `json:"name" gorm:"size:128;not null"`
	Format           string         `json:"format" gorm:"size:32;not null;default:text"`
	Description      string         `json:"description" gorm:"size:512"`
	PublishedVersion int            `json:"published_version" gorm:"not null;default:0;comment:当前发布版本号,0=未发布"`
	Status           int            `json:"status" gorm:"not null;default:1;index"`
	IsBuiltin        bool           `json:"is_builtin" gorm:"not null;default:false"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (PlatformTemplate) TableName() string { return "platform_templates" }

// PlatformTemplateVersion 模板版本正文。
// ContentInline 为权威正文（保证无 MinIO 也可解析）；StorageKey 为 MinIO 镜像（可选）。
type PlatformTemplateVersion struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	TemplateID    uint      `json:"template_id" gorm:"not null;uniqueIndex:uk_plat_tpl_ver,priority:1;index"`
	Version       int       `json:"version" gorm:"not null;uniqueIndex:uk_plat_tpl_ver,priority:2"`
	ContentInline string    `json:"content_inline" gorm:"type:longtext;comment:正文权威副本"`
	StorageKey    string    `json:"storage_key" gorm:"size:512;comment:MinIO object key"`
	Checksum      string    `json:"checksum" gorm:"size:64"`
	Remark        string    `json:"remark" gorm:"size:512"`
	CreatedBy     uint      `json:"created_by" gorm:"not null;default:0"`
	CreatedAt     time.Time `json:"created_at"`
}

func (PlatformTemplateVersion) TableName() string { return "platform_template_versions" }
