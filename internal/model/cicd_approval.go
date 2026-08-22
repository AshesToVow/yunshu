package model

import "time"

const (
	CicdApprovalStageTestLead    = "test_lead"
	CicdApprovalStageRDLead      = "rd_lead"
	CicdApprovalStageProductLead = "product_lead"
	CicdApprovalStageOpsLead     = "ops_lead"

	CicdApprovalStepPending  = "pending"
	CicdApprovalStepApproved = "approved"
	CicdApprovalStepRejected = "rejected"
	CicdApprovalStepSkipped  = "skipped"
)

// CicdApprovalFlowStage 项目级 CD 审批流节点（按 sort_order 串行）。
type CicdApprovalFlowStage struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ProjectID   uint      `json:"project_id" gorm:"not null;uniqueIndex:uk_cicd_flow_stage,priority:1"`
	StageKey    string    `json:"stage_key" gorm:"size:32;not null;uniqueIndex:uk_cicd_flow_stage,priority:2"`
	StageName   string    `json:"stage_name" gorm:"size:64;not null"`
	SortOrder   int       `json:"sort_order" gorm:"not null;default:0"`
	Enabled     bool      `json:"enabled" gorm:"not null;default:true"`
	UserGroupID *uint     `json:"user_group_id,omitempty" gorm:"index;comment:审批用户组"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CicdApprovalFlowStage) TableName() string { return "cicd_approval_flow_stages" }

// CicdReleaseApprovalStep 单次发布工单的多级审批实例。
type CicdReleaseApprovalStep struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	ReleaseRunID   uint       `json:"release_run_id" gorm:"not null;index;index:idx_cicd_rel_step_run_status,priority:1"`
	StageKey       string     `json:"stage_key" gorm:"size:32;not null"`
	StageName      string     `json:"stage_name" gorm:"size:64;not null"`
	SortOrder      int        `json:"sort_order" gorm:"not null;default:0"`
	Status         string     `json:"status" gorm:"size:32;not null;default:'pending';index:idx_cicd_rel_step_run_status,priority:2;index:idx_cicd_rel_step_status_activated,priority:1"`
	UserGroupID    *uint      `json:"user_group_id,omitempty"`
	ReviewerUserID *uint      `json:"reviewer_user_id,omitempty"`
	ReviewerName   string     `json:"reviewer_name" gorm:"size:64"`
	ReviewComment  string     `json:"review_comment" gorm:"size:512"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty" gorm:"index:idx_cicd_rel_step_status_activated,priority:2;comment:当前节点开始等待审批的时间"`
	LastRemindedAt *time.Time `json:"last_reminded_at,omitempty" gorm:"comment:上次 SLA 提醒邮件时间"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (CicdReleaseApprovalStep) TableName() string { return "cicd_release_approval_steps" }
