package model

import "time"

// MenuPermissionBinding 菜单入口与 API 权限绑定（覆盖或补充静态 catalog 映射）。
type MenuPermissionBinding struct {
	ID         uint      `json:"id" gorm:"primaryKey;comment:主键ID"`
	MenuID     uint      `json:"menu_id" gorm:"not null;index:idx_menu_perm_binding_menu,priority:1;comment:菜单ID"`
	Resource   string    `json:"resource" gorm:"size:256;not null;comment:Casbin resource(FullPath)"`
	Action     string    `json:"action" gorm:"size:16;not null;comment:HTTP method"`
	Mode       string    `json:"mode" gorm:"size:8;not null;default:any;comment:any=任一权限即可"`
	CreatedAt  time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"comment:更新时间"`
}

func (MenuPermissionBinding) TableName() string {
	return "menu_permission_bindings"
}
