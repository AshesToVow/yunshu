package model

import (
	"time"

	"gorm.io/gorm"
)

// Permission 细粒度权限：资源标识 + 操作，可选启用 K8s 命名空间等资源范围控制。
type Permission struct {
	ID              uint      `json:"id" gorm:"primaryKey;index:idx_permissions_deleted_id,priority:2;comment:主键ID"`
	Name            string    `json:"name" gorm:"size:64;not null;comment:权限名称"`
	// 唯一键不含 deleted_at：MySQL 下 NULL 不互斥会导致 seed 反复插入重复活跃行。
	// 软删除后再次 seed 通过 ON CONFLICT 更新并清空 deleted_at 复活。
	Resource        string    `json:"resource" gorm:"size:191;not null;uniqueIndex:idx_perm_resource_action,priority:1;comment:资源标识"`
	Action          string    `json:"action" gorm:"size:32;not null;uniqueIndex:idx_perm_resource_action,priority:2;comment:操作标识"`
	Description     string    `json:"description" gorm:"size:255;comment:权限描述"`
	K8sScopeEnabled bool      `json:"k8s_scope_enabled" gorm:"not null;default:false;comment:是否启用K8s范围控制"`
	CreatedAt       time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"comment:更新时间"`
	// Help `WHERE deleted_at IS NULL ORDER BY id DESC LIMIT ...` on large tables.
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_permissions_deleted_id,priority:1;comment:删除时间"`
}
