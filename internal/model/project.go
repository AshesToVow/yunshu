package model

import (
	"time"

	"gorm.io/gorm"
)

// Project 业务项目：名称、唯一编码、描述，用于资源与成员的租户隔离。
type Project struct {
	ID uint `json:"id" gorm:"primaryKey;comment:主键ID"`

	Name        string  `json:"name" gorm:"size:128;not null;index;comment:项目名称"`
	Code        string  `json:"code" gorm:"size:64;not null;uniqueIndex;comment:项目编码"`
	Description *string `json:"description" gorm:"type:text;comment:项目描述"`
	Status      int     `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	ProjectType string  `json:"project_type" gorm:"size:32;not null;default:'business';index;comment:项目类型 business/platform/infra/research"`
	// LifecycleStatus 项目生命周期状态，与 Status（启用/停用）独立。
	LifecycleStatus string `json:"lifecycle_status" gorm:"size:32;not null;default:'active';index;comment:项目生命周期 planning/active/suspended/archived"`

	// HarborURL 项目级 Harbor 地址（如 harbor.example.com）；空则回退全局 cicd 配置。
	HarborURL string `json:"harbor_url" gorm:"size:256;comment:项目Harbor地址，空则用全局配置"`
	// HarborProject Harbor 中的项目/命名空间名（如 registry、team-a）；对应 Jenkins PROJECT_GROUP。
	HarborProject string `json:"harbor_project" gorm:"size:128;comment:项目Harbor项目名，空则用全局配置"`

	// ApolloMeta 项目级 Apollo Meta 地址（可逗号分隔多个）；发布时作为 Jenkins APOLLO_META 注入 launch/K8s 模板。
	ApolloMeta string `json:"apollo_meta" gorm:"size:1024;comment:项目Apollo Meta地址(可逗号分隔多个)，空则不覆盖Jenkins默认"`
	// ApolloEnv Apollo 环境名（如 DEV/FAT/PRO）；空则按发布 Tenv 推导。
	ApolloEnv string `json:"apollo_env" gorm:"size:32;comment:项目Apollo环境，空则按Tenv推导"`
	// ApolloNamespaces Apollo bootstrap namespaces（逗号分隔）。
	ApolloNamespaces string `json:"apollo_namespaces" gorm:"size:512;comment:项目Apollo namespaces"`

	// OwnerDepartmentID 可选归属部门，用于组织维度筛选与报表（不自动决定成员权限）。
	OwnerDepartmentID *uint       `json:"owner_department_id,omitempty" gorm:"index;comment:可选归属部门ID"`
	OwnerDepartment   *Department `json:"owner_department,omitempty" gorm:"foreignKey:OwnerDepartmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	CreatedAt time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}

// TableName 指定 GORM 表名为 projects。
func (Project) TableName() string { return "projects" }
