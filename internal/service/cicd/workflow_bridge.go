package cicd

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	workflowsvc "yunshu/internal/service/workflow"
)

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
	return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
		"workflow_ticket_id":        releaseTicket.ID,
		"change_workflow_ticket_id": changeTicket.ID,
	}).Error
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
		return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
			"status": model.CicdRunStatusRejected, "current_stage_key": "",
			"review_comment": strings.TrimSpace(comment),
		}).Error
	}
	if detail.Status == model.WorkflowTicketStatusApproved {
		return s.db.WithContext(ctx).Model(release).Updates(map[string]any{
			"status": model.CicdRunStatusPendingExecution, "current_stage_key": "",
		}).Error
	}
	// 同步 current_stage_key 供旧 UI 展示
	if len(detail.Steps) > 0 {
		for _, st := range detail.Steps {
			if st.Status == model.WorkflowStepPending && st.ActivatedAt != nil {
				return s.db.WithContext(ctx).Model(release).Update("current_stage_key", st.StageKey).Error
			}
		}
	}
	return nil
}

func (s *Service) linkChangeTicket(ctx context.Context, releaseID, changeTicketID uint) error {
	var release model.CicdReleaseRun
	if err := s.db.WithContext(ctx).First(&release, releaseID).Error; err != nil {
		return err
	}
	var ticket model.WorkflowTicket
	if err := s.db.WithContext(ctx).First(&ticket, changeTicketID).Error; err != nil {
		return err
	}
	if ticket.TicketType != model.WorkflowTicketTypeChange {
		return constants.ErrBadRequestWithMsg("目标工单不是变更单")
	}
	return s.db.WithContext(ctx).Model(&release).Update("change_workflow_ticket_id", changeTicketID).Error
}
