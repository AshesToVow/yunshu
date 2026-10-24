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

// EnsureDefaultIncidentDefinition 确保全局故障单流程存在（project_id=0）。
// 使用 platform_role，无需预先绑定用户组/值班规则，告警「转工单」开箱可用。
func EnsureDefaultIncidentDefinition(ctx context.Context, wfRepo interfaces.WorkflowRepository) error {
	if wfRepo == nil {
		return nil
	}
	svc := NewService(wfRepo, nil, nil, nil)
	key := DefinitionKey{
		Domain: model.WorkflowDomainIncident, ProjectID: 0, TicketType: model.WorkflowTicketTypeIncident,
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
			StageKey: "incident_ops", StageName: "故障处置", SortOrder: 10,
			Enabled: true, AssigneeRuleType: model.WorkflowAssigneePlatformRole,
		}},
	})
	return err
}

// EnsureDefaultIncidentDefinitionDB 供 bootstrap / migrate 等仍持有 *gorm.DB 的调用方。
func EnsureDefaultIncidentDefinitionDB(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return EnsureDefaultIncidentDefinition(ctx, repository.NewWorkflowRepository(db))
}
