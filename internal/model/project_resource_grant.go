package model

import "time"

const (
	ResourcePrincipalUser = "user"
)

// ServerAccessGrant 项目内服务器资源授权（成员级；owner/admin 隐式全量不依赖本表）。
type ServerAccessGrant struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ProjectID     uint      `json:"project_id" gorm:"not null;uniqueIndex:uk_server_grant,priority:1;index"`
	ServerID      uint      `json:"server_id" gorm:"not null;uniqueIndex:uk_server_grant,priority:2;index"`
	PrincipalKind string    `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_server_grant,priority:3;default:'user'"`
	PrincipalRef  string    `json:"principal_ref" gorm:"size:64;not null;uniqueIndex:uk_server_grant,priority:4;index"`
	CanView       bool      `json:"can_view" gorm:"not null;default:true"`
	CanExec       bool      `json:"can_exec" gorm:"not null;default:false"`
	CanManage     bool      `json:"can_manage" gorm:"not null;default:false"`
	Remark        string    `json:"remark" gorm:"size:512"`
	CreatedBy     *uint     `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ServerAccessGrant) TableName() string { return "server_access_grants" }

// CicdAccessGrant 项目内 CI/CD 应用授权。
type CicdAccessGrant struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	ProjectID     uint      `json:"project_id" gorm:"not null;uniqueIndex:uk_cicd_grant,priority:1;index"`
	ServiceID     uint      `json:"service_id" gorm:"not null;uniqueIndex:uk_cicd_grant,priority:2;index;comment:cicd_services.id"`
	PrincipalKind string    `json:"principal_kind" gorm:"size:16;not null;uniqueIndex:uk_cicd_grant,priority:3;default:'user'"`
	PrincipalRef  string    `json:"principal_ref" gorm:"size:64;not null;uniqueIndex:uk_cicd_grant,priority:4;index"`
	CanView       bool      `json:"can_view" gorm:"not null;default:true"`
	CanBuild      bool      `json:"can_build" gorm:"not null;default:false"`
	CanRelease    bool      `json:"can_release" gorm:"not null;default:false"`
	CanManage     bool      `json:"can_manage" gorm:"not null;default:false"`
	Remark        string    `json:"remark" gorm:"size:512"`
	CreatedBy     *uint     `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CicdAccessGrant) TableName() string { return "cicd_access_grants" }
