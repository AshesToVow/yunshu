package model

import "time"

// AiInvestigation AI 调查任务（告警/Pod/CI/对话等确定性采集 + LLM 分析）。
type AiInvestigation struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserID       uint      `json:"user_id" gorm:"not null;index"`
	Kind         string    `json:"kind" gorm:"size:32;not null;index"` // alert|pod|cicd|chat
	Title        string    `json:"title" gorm:"size:256;not null"`
	Status       string    `json:"status" gorm:"size:32;not null;index;default:collecting"` // collecting|analyzing|recommend|awaiting_approval|done|failed
	ProjectID    uint      `json:"project_id" gorm:"index"`
	ClusterID    uint      `json:"cluster_id" gorm:"index"`
	Namespace    string    `json:"namespace" gorm:"size:128"`
	Resource     string    `json:"resource" gorm:"size:256"`
	Fingerprint  string    `json:"fingerprint" gorm:"size:128;index"`
	InputJSON    string    `json:"input_json" gorm:"type:mediumtext;charset:utf8mb4"`
	CollectJSON  string    `json:"collect_json" gorm:"type:mediumtext;charset:utf8mb4"`
	AnalysisJSON string    `json:"analysis_json" gorm:"type:mediumtext;charset:utf8mb4"`
	ReportJSON   string    `json:"report_json" gorm:"type:mediumtext;charset:utf8mb4"`
	SessionID    *uint     `json:"session_id,omitempty" gorm:"index"`
	ApprovalID   *uint     `json:"approval_id,omitempty" gorm:"index"`
	ErrorMsg     string    `json:"error_msg,omitempty" gorm:"size:1024"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AiInvestigation) TableName() string { return "ai_investigations" }
