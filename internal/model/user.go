package model

import (
	"time"

	"gorm.io/gorm"
)

// User 系统用户：登录名、邮箱、密码哈希、昵称、启用状态及关联角色（多对多）。
type User struct {
	ID           uint           `json:"id" gorm:"primaryKey;comment:主键ID"`
	Username     string         `json:"username" gorm:"size:64;not null;uniqueIndex;comment:用户名"`
	Email        *string        `json:"email" gorm:"size:128;index;comment:邮箱"`
	Phone        string         `json:"phone" gorm:"size:20;not null;default:'';index;comment:手机号，用于钉钉/企微@等告警触达"`
	Password            string         `json:"-" gorm:"size:255;not null;comment:加密后的登录密码"`
	PasswordChangedAt   *time.Time     `json:"password_changed_at,omitempty" gorm:"comment:最近修改密码时间，用于过期策略"`
	MustChangePassword  bool           `json:"must_change_password" gorm:"not null;default:false;comment:下次登录强制改密"`
	// MFAEnabled 表示已启用 TOTP 二次验证；MFASecret 为 AES-GCM 加密后的 TOTP 密钥，
	// MFARecoveryCodes 为恢复码的 bcrypt 哈希 JSON 数组，二者都不允许出现在任何 JSON 响应中。
	MFAEnabled       bool       `json:"mfa_enabled" gorm:"not null;default:false;comment:是否启用 TOTP 二次验证"`
	MFASecret        string     `json:"-" gorm:"size:512;not null;default:'';comment:TOTP 密钥（AES-GCM 加密）"`
	MFARecoveryCodes string     `json:"-" gorm:"type:text;comment:TOTP 恢复码哈希（JSON 数组）"`
	MFAEnabledAt     *time.Time `json:"mfa_enabled_at,omitempty" gorm:"comment:TOTP 启用时间"`
	Nickname            string         `json:"nickname" gorm:"size:128;not null;comment:昵称"`
	Status              int            `json:"status" gorm:"not null;default:1;index:idx_users_dept_status,priority:2;comment:状态 1启用 0禁用"`
	DepartmentID        *uint          `json:"department_id" gorm:"index:idx_users_dept_status,priority:1;comment:所属部门ID"`
	Department          *Department    `json:"department,omitempty" gorm:"foreignKey:DepartmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Roles               []Role         `json:"roles,omitempty" gorm:"many2many:user_roles;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Groups              []UserGroup    `json:"groups,omitempty" gorm:"many2many:user_group_users;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CreatedAt           time.Time      `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt           time.Time      `json:"updated_at" gorm:"comment:更新时间"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index;comment:删除时间"`
}
