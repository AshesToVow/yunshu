package cicd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	workflowsvc "yunshu/internal/service/workflow"
)

func (s *Service) linkChangeTicket(ctx context.Context, releaseID, changeTicketID uint) error {
	release, err := s.repo.GetReleaseRunByID(ctx, releaseID)
	if err != nil {
		return err
	}
	ticket, err := s.workflowRepo.GetTicket(ctx, changeTicketID)
	if err != nil {
		return err
	}
	if ticket.TicketType != model.WorkflowTicketTypeChange {
		return constants.ErrBadRequestWithMsg("目标工单不是变更单")
	}
	return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
		"change_workflow_ticket_id": changeTicketID,
	})
}

func (s *Service) createReleaseWorkflowTickets(ctx context.Context, release *model.CicdReleaseRun) error {
	if release == nil || release.ID == 0 {
		return constants.ErrBadRequestWithMsg("工单不存在")
	}
	wf := s.workflowEngine()
	submitter := uint(0)
	if release.SubmitterUserID != nil {
		submitter = *release.SubmitterUserID
	}
	title := strings.TrimSpace(release.Title)
	if title == "" {
		title = fmt.Sprintf("发布工单 #%d", release.ID)
	}
	releaseTicket, err := wf.CreateLinkedTicket(ctx, workflowsvc.LinkedTicketInput{
		Domain: model.WorkflowDomainCicd, TicketType: model.WorkflowTicketTypeRelease,
		ProjectID: release.ProjectID, Title: title, SubmitterUserID: submitter,
		RefType: model.WorkflowRefCicdReleaseRun, RefID: release.ID,
		Payload: map[string]any{
			"service_id": release.ServiceID, "tenv": release.Tenv, "release_type": release.ReleaseType,
		},
	})
	if err != nil {
		return err
	}
	changeTicket, err := wf.CreateInfoTicket(ctx, workflowsvc.LinkedTicketInput{
		Domain: model.WorkflowDomainOps, TicketType: model.WorkflowTicketTypeChange,
		ProjectID: release.ProjectID, Title: "变更单 · " + title, SubmitterUserID: submitter,
		RefType: model.WorkflowRefCicdReleaseChange, RefID: release.ID,
		Payload: map[string]any{
			"release_run_id": release.ID, "release_ticket_id": releaseTicket.ID,
			"service_id": release.ServiceID, "tenv": release.Tenv,
		},
	}, model.WorkflowTicketStatusApproved)
	if err != nil {
		return err
	}
	stageKey := ""
	if active, err := wf.GetActiveStep(ctx, releaseTicket.ID); err == nil && active != nil {
		stageKey = active.StageKey
	}
	return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
		"workflow_ticket_id":          releaseTicket.ID,
		"change_workflow_ticket_id":   changeTicket.ID,
		"current_stage_key":           stageKey,
	})
}

func (s *Service) reviewReleaseViaWorkflow(ctx context.Context, release *model.CicdReleaseRun, approve bool, comment string, actor *auth.CurrentUser) error {
	if release == nil {
		return constants.ErrBadRequestWithMsg("工单不存在")
	}
	wf := s.workflowEngine()
	if !wf.HasLinkedTicketType(ctx, model.WorkflowRefCicdReleaseRun, release.ID, model.WorkflowTicketTypeRelease) {
		return workflowsvc.ErrNoLinkedTicket
	}
	detail, err := wf.ReviewLinkedStepTyped(ctx, model.WorkflowRefCicdReleaseRun, release.ID, model.WorkflowTicketTypeRelease, approve, comment, actor)
	if err != nil {
		return err
	}
	if !approve {
		_ = s.syncLegacyReleaseSteps(ctx, release.ID, "", false, comment, actor)
		return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
			"status": model.CicdRunStatusRejected, "current_stage_key": "",
			"review_comment": strings.TrimSpace(comment),
		})
	}
	if detail.Status == model.WorkflowTicketStatusApproved {
		_ = s.syncLegacyReleaseSteps(ctx, release.ID, "", true, comment, actor)
		return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
			"status": model.CicdRunStatusPendingExecution, "current_stage_key": "",
		})
	}
	// 同步 current_stage_key 供旧 UI 展示
	if len(detail.Steps) > 0 {
		for _, st := range detail.Steps {
			if st.Status == model.WorkflowStepPending && st.ActivatedAt != nil {
				_ = s.syncLegacyReleaseSteps(ctx, release.ID, st.StageKey, true, comment, actor)
				return s.repo.UpdateReleaseRunFields(ctx, release.ID, map[string]any{
					"current_stage_key": st.StageKey,
				})
			}
		}
	}
	return nil
}

// syncLegacyReleaseSteps 将 workflow 审批结果回写 cicd_release_approval_steps，供 SLA/旧详情兼容。
func (s *Service) syncLegacyReleaseSteps(ctx context.Context, releaseID uint, nextStageKey string, approve bool, comment string, actor *auth.CurrentUser) error {
	step, err := s.repo.GetPendingApprovalStep(ctx, releaseID)
	if err != nil {
		return nil // 无旧步骤则跳过
	}
	now := time.Now()
	reviewerID := uint(0)
	reviewerName := ""
	if actor != nil {
		reviewerID = actor.ID
		reviewerName = strings.TrimSpace(actor.Username)
		if reviewerName == "" {
			reviewerName = strings.TrimSpace(actor.Nickname)
		}
	}
	status := model.CicdApprovalStepRejected
	if approve {
		status = model.CicdApprovalStepApproved
	}
	if err := s.repo.UpdateApprovalStepFields(ctx, step.ID, map[string]any{
		"status": status, "reviewer_user_id": reviewerID, "reviewer_name": reviewerName,
		"review_comment": strings.TrimSpace(comment), "reviewed_at": now,
	}); err != nil {
		return err
	}
	if !approve || nextStageKey == "" {
		return nil
	}
	return s.repo.ActivateApprovalStep(ctx, releaseID, nextStageKey)
}
