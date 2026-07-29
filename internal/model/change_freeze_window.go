package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	FreezeScopeAll     = "all"
	FreezeScopeCicd    = "cicd"
	FreezeScopeK8s     = "k8s"
	FreezeScopeDbmgmt  = "dbmgmt"
	FreezeEnvAll       = ""
	FreezeEnvProd      = "prod"
	FreezeEnvStaging   = "staging"
	FreezeEnvTest      = "test"
	FreezeEnvDev       = "dev"
)

// ChangeFreezeWindow 变更冻结窗口：在窗口内拦截指定来源的写操作。
type ChangeFreezeWindow struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	ProjectID   uint           `json:"project_id" gorm:"not null;index:idx_freeze_proj_time,priority:1"`
	Name        string         `json:"name" gorm:"size:128;not null"`
	Scope       string         `json:"scope" gorm:"size:32;not null;default:'all';comment:all|cicd|k8s|dbmgmt"`
	Env         string         `json:"env" gorm:"size:32;comment:空=全部环境；prod/staging/test/dev"`
	StartsAt    time.Time      `json:"starts_at" gorm:"not null;index:idx_freeze_proj_time,priority:2"`
	EndsAt      time.Time      `json:"ends_at" gorm:"not null"`
	Reason      string         `json:"reason" gorm:"size:512"`
	Enabled     bool           `json:"enabled" gorm:"not null;default:true"`
	CreatedBy   uint           `json:"created_by"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ChangeFreezeWindow) TableName() string { return "change_freeze_windows" }
