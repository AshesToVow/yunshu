package model

import "time"

// UserGroupUser 用户与用户组关联（多对多中间表）。
type UserGroupUser struct {
	UserID      uint      `gorm:"primaryKey;index:idx_ugu_group_user,priority:2;comment:用户ID"`
	UserGroupID uint      `gorm:"primaryKey;index:idx_ugu_group_user,priority:1;comment:用户组ID"`
	CreatedAt   time.Time `json:"created_at"`
}

func (UserGroupUser) TableName() string { return "user_group_users" }
