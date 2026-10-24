package model

import (
	"time"

	"gorm.io/gorm"
)

// KafkamgmtConnection Kafka 管理连接（与日志管道 KafkaProvider 独立）。
type KafkamgmtConnection struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"size:128;not null"`
	Brokers       string         `json:"brokers" gorm:"size:1024;not null;comment:逗号分隔"`
	Username      string         `json:"username" gorm:"size:128"`
	PasswordEnc   string         `json:"-" gorm:"column:password_enc;size:512"`
	HasPassword   bool           `json:"has_password" gorm:"-"`
	SASLMechanism string         `json:"sasl_mechanism" gorm:"size:32;not null;default:''"`
	TimeoutSec    int            `json:"timeout_sec" gorm:"not null;default:10"`
	IsDefault     bool           `json:"is_default" gorm:"not null;default:false"`
	OwnerUserID   uint           `json:"owner_user_id,omitempty" gorm:"index"`
	Remark        string         `json:"remark" gorm:"size:255"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (KafkamgmtConnection) TableName() string { return "kafkamgmt_connections" }
