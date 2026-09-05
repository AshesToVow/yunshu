package ai

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	workflowsvc "yunshu/internal/service/workflow"
)

func (s *Service) workflowEngine() *workflowsvc.Service {
	return workflowsvc.NewService(s.db, nil, nil, nil)
}

func (s *Service) createAIWorkflowTicket(ctx context.Context, row *model.AiToolApproval) error {
	if row == nil || row.ID == 0 || s.db == nil {
		return nil
	}
	if err := workflowsvc.EnsureDefaultAIToolApprovalDefinition(ctx, s.db); err != nil {
		return err
	}
	title := fmt.Sprintf("AI 高危操作 · %s", strings.TrimSpace(row.ToolName))
	if res := strings.TrimSpace(row.Resource); res != "" {
		title += " · " + res
	}
	_, err := s.workflowEngine().CreateLinkedTicket(ctx, workflowsvc.LinkedTicketInput{
		Domain: model.WorkflowDomainAI, TicketType: model.WorkflowTicketTypeToolApproval,
		ProjectID: 0, Title: title, SubmitterUserID: row.UserID,
		RefType: model.WorkflowRefAiToolApproval, RefID: row.ID,
		Payload: map[string]any{
			"tool_name": row.ToolName, "cluster_id": row.ClusterID,
			"namespace": row.Namespace, "resource": row.Resource,
		},
	})
	return err
}

func (s *Service) reviewAIViaWorkflow(ctx context.Context, row *model.AiToolApproval, approve bool, note string, actor *auth.CurrentUser) (*workflowsvc.TicketDetail, error) {
	wf := s.workflowEngine()
	if !wf.HasLinkedTicket(ctx, model.WorkflowRefAiToolApproval, row.ID) {
		return nil, workflowsvc.ErrNoLinkedTicket
	}
	return wf.ReviewLinkedStep(ctx, model.WorkflowRefAiToolApproval, row.ID, approve, note, actor)
}
