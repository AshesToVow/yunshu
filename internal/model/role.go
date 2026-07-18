package model

import (
	"time"

	"gorm.io/gorm"
)

// Role 角色定义：名称、唯一编码、描述及启用状态，用于 RBAC 与菜单权限过滤。
type Role struct {
	ID          uint      `json:"id" gorm:"primaryKey;comment:主键ID"`
	Name        string    `json:"name" gorm:"size:64;not null;uniqueIndex:idx_roles_name_deleted,priority:1;comment:角色名称"`
	Code        string    `json:"code" gorm:"size:64;not null;uniqueIndex:idx_roles_code_deleted,priority:1;comment:角色编码"`
	Description string    `json:"description" gorm:"size:255;comment:角色描述"`
	Status      int       `json:"status" gorm:"not null;default:1;comment:状态 1启用 0禁用"`
	CreatedAt   time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"comment:更新时间"`
	// deleted_at 参与唯一索引：软删除后可复用同名/同编码（NULL 视为不同值）。
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:idx_roles_name_deleted,priority:2;uniqueIndex:idx_roles_code_deleted,priority:2;comment:删除时间"`
}

// ExtractRoleCodes 从角色列表提取角色编码，用于鉴权与策略匹配。
func ExtractRoleCodes(roles []Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}
