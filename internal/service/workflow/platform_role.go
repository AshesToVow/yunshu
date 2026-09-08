package workflow

import (
	"context"
	"slices"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// CanPlatformRoleReview 平台角色审批人（AI 高危操作等全局审批节点）。
func CanPlatformRoleReview(actor *auth.CurrentUser) bool {
	if actor == nil {
		return false
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true
	}
	return slices.ContainsFunc(actor.RoleCodes, func(c string) bool {
		switch c {
		case "admin", "ops-admin", "ai-approver":
			return true
		default:
			return false
		}
	})
}

// EnsureDefaultAIToolApprovalDefinition 确保全局 AI 工具审批流存在（project_id=0）。
func EnsureDefaultAIToolApprovalDefinition(ctx context.Context, wfRepo interfaces.WorkflowRepository) error {
	if wfRepo == nil {
		return nil
	}
	svc := NewService(wfRepo, nil, nil, nil)
	key := DefinitionKey{
		Domain: model.WorkflowDomainAI, ProjectID: 0, TicketType: model.WorkflowTicketTypeToolApproval,
	}
	def, stages, err := svc.loadDefinition(ctx, key)
	if err != nil {
		return err
	}
	if def != nil && len(filterEnabledStages(stages)) > 0 {
		return nil
	}
	_, err = svc.UpsertDefinition(ctx, key, DefinitionUpsertRequest{
		Stages: []StageUpsertItem{{
			StageKey: "ai_ops", StageName: "AI 运维审批", SortOrder: 10,
			Enabled: true, AssigneeRuleType: model.WorkflowAssigneePlatformRole,
		}},
	})
	return err
}

// EnsureDefaultAIToolApprovalDefinitionDB 供 bootstrap / migrate 等仍持有 *gorm.DB 的调用方。
func EnsureDefaultAIToolApprovalDefinitionDB(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return EnsureDefaultAIToolApprovalDefinition(ctx, repository.NewWorkflowRepository(db))
}
