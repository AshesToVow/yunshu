package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	WorkflowDomainDbmgmt   = "dbmgmt"
	WorkflowDomainCicd     = "cicd"
	WorkflowDomainIncident = "incident"
	WorkflowDomainOps      = "ops"
	WorkflowDomainAI       = "ai"

	WorkflowTicketTypeDefault      = "default"
	WorkflowTicketTypeSql          = "sql_ticket"
	WorkflowTicketTypeAccess       = "access_request"
	WorkflowTicketTypeAppUser      = "app_user_apply"
	WorkflowTicketTypeRelease      = "release"
	WorkflowTicketTypeIncident     = "incident"
	WorkflowTicketTypeChange       = "change"
	WorkflowTicketTypeToolApproval = "tool_approval"

	WorkflowAssigneeUserGroup    = "user_group"
	WorkflowAssigneeDuty         = "duty"
	WorkflowAssigneePlatformRole = "platform_role"

	WorkflowTicketStatusDraft          = "draft"
	WorkflowTicketStatusPending        = "pending"
	WorkflowTicketStatusApproved       = "approved"
	WorkflowTicketStatusRejected       = "rejected"
	WorkflowTicketStatusCancelled      = "cancelled"
	WorkflowTicketStatusClosed         = "closed"

	WorkflowStepPending  = "pending"
	WorkflowStepApproved = "approved"
	WorkflowStepRejected = "rejected"
	WorkflowStepSkipped  = "skipped"

	WorkflowRefDbSqlTicket      = "db_sql_ticket"
	WorkflowRefDbAccessRequest  = "db_access_request"
	WorkflowRefDbAppUserRequest = "db_app_user_request"
	WorkflowRefCicdReleaseRun    = "cicd_release_run"
	WorkflowRefCicdReleaseChange = "cicd_release_change"
	WorkflowRefAlertEvent       = "alert_event"
	WorkflowRefAiToolApproval   = "ai_tool_approval"
)

// WorkflowDefinition 跨域工单流程定义（按 domain + project + ticket_type 唯一）。
type WorkflowDefinition struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Domain            string         `json:"domain" gorm:"size:32;not null;uniqueIndex:idx_wf_def_domain_proj_type,priority:1"`
	ProjectID         uint           `json:"project_id" gorm:"not null;default:0;uniqueIndex:idx_wf_def_domain_proj_type,priority:2;comment:0表示全局"`
	TicketType        string         `json:"ticket_type" gorm:"size:32;not null;default:default;uniqueIndex:idx_wf_def_domain_proj_type,priority:3"`
	Name              string         `json:"name" gorm:"size:128;not null;default:''"`
	ForbidSelfApprove bool           `json:"forbid_self_approve" gorm:"not null;default:true"`
	Enabled           bool           `json:"enabled" gorm:"not null;default:true"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkflowDefinition) TableName() string { return "workflow_definitions" }

// WorkflowStage 流程节点：串行审批，支持用户组或排班派单。
type WorkflowStage struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	DefinitionID      uint           `json:"definition_id" gorm:"not null;index;uniqueIndex:idx_wf_stage_def_key,priority:1"`
	StageKey          string         `json:"stage_key" gorm:"size:64;not null;uniqueIndex:idx_wf_stage_def_key,priority:2"`
	StageName         string         `json:"stage_name" gorm:"size:64;not null"`
	SortOrder         int            `json:"sort_order" gorm:"not null;default:0"`
	Enabled           bool           `json:"enabled" gorm:"not null;default:false"`
	AssigneeRuleType  string         `json:"assignee_rule_type" gorm:"size:32;not null;default:user_group"`
	UserGroupID       *uint          `json:"user_group_id,omitempty" gorm:"index"`
	DutyMonitorRuleID *uint          `json:"duty_monitor_rule_id,omitempty" gorm:"index;comment:排班规则ID，assignee_rule_type=duty时使用"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkflowStage) TableName() string { return "workflow_stages" }

// WorkflowTicket 通用工单实例（告警转单、运维变更等）。
type WorkflowTicket struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	DefinitionID    uint           `json:"definition_id" gorm:"not null;index"`
	Domain          string         `json:"domain" gorm:"size:32;not null;index"`
	TicketType      string         `json:"ticket_type" gorm:"size:32;not null;index"`
	ProjectID       uint           `json:"project_id" gorm:"not null;default:0;index"`
	Title           string         `json:"title" gorm:"size:256;not null"`
	Status          string         `json:"status" gorm:"size:32;not null;index"`
	SubmitterUserID uint           `json:"submitter_user_id" gorm:"not null;default:0;index"`
	RefType         string         `json:"ref_type" gorm:"size:64;not null;default:'';index:idx_wf_ticket_ref,priority:1"`
	RefID           uint           `json:"ref_id" gorm:"not null;default:0;index:idx_wf_ticket_ref,priority:2"`
	PayloadJSON     string         `json:"payload_json" gorm:"type:text"`
	Remark          string         `json:"remark" gorm:"size:512"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	ClosedAt        *time.Time     `json:"closed_at,omitempty"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkflowTicket) TableName() string { return "workflow_tickets" }

// WorkflowTicketStep 工单审批步骤。
type WorkflowTicketStep struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	TicketID         uint           `json:"ticket_id" gorm:"not null;index"`
	StageKey         string         `json:"stage_key" gorm:"size:64;not null"`
	StageName        string         `json:"stage_name" gorm:"size:64;not null"`
	SortOrder        int            `json:"sort_order" gorm:"not null;default:0"`
	Status           string         `json:"status" gorm:"size:32;not null;default:pending;index"`
	AssigneeRuleType string         `json:"assignee_rule_type" gorm:"size:32;not null;default:user_group"`
	UserGroupID      *uint          `json:"user_group_id,omitempty"`
	DutyMonitorRuleID *uint         `json:"duty_monitor_rule_id,omitempty"`
	AssigneeUserID   *uint          `json:"assignee_user_id,omitempty" gorm:"index;comment:排班解析出的当班处理人"`
	ReviewerUserID   *uint          `json:"reviewer_user_id,omitempty" gorm:"index"`
	ReviewComment    string         `json:"review_comment" gorm:"size:512"`
	ActivatedAt      *time.Time     `json:"activated_at,omitempty"`
	LastRemindedAt   *time.Time     `json:"last_reminded_at,omitempty" gorm:"comment:上次 SLA 提醒邮件时间"`
	ReviewedAt       *time.Time     `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

func (WorkflowTicketStep) TableName() string { return "workflow_ticket_steps" }
