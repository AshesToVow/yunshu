package model

import (
	"time"

	"gorm.io/gorm"
)

// AiChatSession AI 助手会话（按用户隔离）。
type AiChatSession struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	UserID        uint           `json:"user_id" gorm:"not null;index"`
	Title         string         `json:"title" gorm:"size:128;not null;default:新对话"`
	ProjectID     uint           `json:"project_id" gorm:"index"`
	ClusterID     uint           `json:"cluster_id" gorm:"index"`
	Provider      string         `json:"provider" gorm:"size:64"`
	EnableTools   bool           `json:"enable_tools" gorm:"not null;default:true"`
	EnableWrite   bool           `json:"enable_write" gorm:"not null;default:false"`
	LastMessageAt *time.Time     `json:"last_message_at,omitempty" gorm:"index"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AiChatSession) TableName() string { return "ai_chat_sessions" }

// AiChatMessage AI 助手会话消息。
type AiChatMessage struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	SessionID uint      `json:"session_id" gorm:"not null;index"`
	Role      string    `json:"role" gorm:"size:16;not null;index"` // user|assistant
	Content   string    `json:"content" gorm:"type:longtext"`
	MetaJSON  string    `json:"meta_json,omitempty" gorm:"type:mediumtext"`
	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

func (AiChatMessage) TableName() string { return "ai_chat_messages" }
