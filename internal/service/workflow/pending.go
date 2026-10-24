package workflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

)

// PendingListQuery 跨域待办查询。
type PendingListQuery struct {
	Domains    string `form:"domains"`
	ProjectID  *uint  `form:"project_id"`
	MineScope  string `form:"mine_scope"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

// PendingTicketItem 统一待办条目。
type PendingTicketItem struct {
	WorkflowTicketID uint       `json:"workflow_ticket_id"`
	StepID           uint       `json:"step_id"`
	Domain           string     `json:"domain"`
	TicketType       string     `json:"ticket_type"`
	ProjectID        uint       `json:"project_id"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	CurrentStageName string     `json:"current_stage_name"`
	SubmitterUserID  uint       `json:"submitter_user_id"`
	SubmitterName    string     `json:"submitter_name"`
	RefType          string     `json:"ref_type"`
	RefID            uint       `json:"ref_id"`
	DeepLink         string     `json:"deep_link"`
	ActivatedAt      *time.Time `json:"activated_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	MineStatus       string     `json:"mine_status"`
	// Action: review=审批；execute=提交人执行（发布 / SQL）
	Action string `json:"action,omitempty"`
}

const (
	MineScopePending = "pending"
	MineScopeDone    = "done"
	MineScopeAll     = "all"
)

// ListPendingForUser 跨域待办：基于 workflow_ticket_steps 聚合。
func (s *Service) ListPendingForUser(ctx context.Context, q PendingListQuery, actor *auth.CurrentUser) (*pagination.Result[PendingTicketItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	mineScope := strings.ToLower(strings.TrimSpace(q.MineScope))
	if mineScope == "" {
		mineScope = MineScopePending
	}
	domains := parseDomains(q.Domains)
	userID := actorUserID(actor)
	isSuper := actor != nil && auth.IsSuperAdminRole(actor.RoleCodes)
	canPlatformReview := CanPlatformRoleReview(actor)

	rows, total, err := s.repo.ListPending(ctx, repository.WorkflowPendingFilter{
		Domains: domains, ProjectID: q.ProjectID, MineScope: mineScope,
		UserID: userID, IsSuper: isSuper, CanPlatformReview: canPlatformReview,
		Offset: (page - 1) * pageSize, Limit: pageSize,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListPendingForUser", err)
	}

	userNames := map[uint]string{}
	for _, r := range rows {
		if r.SubmitterUserID > 0 {
			userNames[r.SubmitterUserID] = ""
		}
	}
	s.fillUserNames(ctx, userNames)

	items := make([]PendingTicketItem, 0, len(rows))
	executeTickets := map[uint]struct{}{}
	if mineScope == MineScopeAll {
		for _, r := range rows {
			if r.Status == model.WorkflowTicketStatusApproved && r.SubmitterUserID == userID &&
				((r.TicketType == model.WorkflowTicketTypeRelease && r.RefType == model.WorkflowRefCicdReleaseRun) ||
					(r.TicketType == model.WorkflowTicketTypeSql && r.RefType == model.WorkflowRefDbSqlTicket)) {
				executeTickets[r.WorkflowTicketID] = struct{}{}
			}
		}
	}
	seenExecute := map[uint]struct{}{}
	for _, r := range rows {
		mineStatus := "mine_done"
		if r.StepStatus == model.WorkflowStepPending && r.ActivatedAt != nil {
			mineStatus = "mine_pending"
		}
		action := "review"
		stageName := r.CurrentStageName
		if (mineScope == MineScopePending || mineScope == MineScopeAll) &&
			r.Status == model.WorkflowTicketStatusApproved &&
			r.SubmitterUserID == userID {
			switch {
			case r.TicketType == model.WorkflowTicketTypeRelease && r.RefType == model.WorkflowRefCicdReleaseRun:
				action = "execute"
				stageName = "待执行"
				mineStatus = "mine_pending"
			case r.TicketType == model.WorkflowTicketTypeSql && r.RefType == model.WorkflowRefDbSqlTicket:
				action = "execute_sql"
				stageName = "待执行"
				mineStatus = "mine_pending"
			}
		}
		if _, isExecTicket := executeTickets[r.WorkflowTicketID]; isExecTicket {
			if action != "execute" && action != "execute_sql" {
				continue
			}
			if _, ok := seenExecute[r.WorkflowTicketID]; ok {
				continue
			}
			seenExecute[r.WorkflowTicketID] = struct{}{}
		}
		items = append(items, PendingTicketItem{
			WorkflowTicketID: r.WorkflowTicketID, StepID: r.StepID,
			Domain: r.Domain, TicketType: r.TicketType, ProjectID: r.ProjectID,
			Title: r.Title, Status: r.Status, CurrentStageName: stageName,
			SubmitterUserID: r.SubmitterUserID, SubmitterName: userNames[r.SubmitterUserID],
			RefType: r.RefType, RefID: r.RefID, DeepLink: buildDeepLink(r.Domain, r.TicketType, r.ProjectID, r.RefType, r.RefID),
			ActivatedAt: r.ActivatedAt, CreatedAt: r.CreatedAt, MineStatus: mineStatus,
			Action: action,
		})
	}
	return &pagination.Result[PendingTicketItem]{
		List: items, Total: total, Page: page, PageSize: pageSize,
	}, nil
}

func parseDomains(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func buildDeepLink(domain, ticketType string, projectID uint, refType string, refID uint) string {
	switch refType {
	case model.WorkflowRefDbSqlTicket:
		return fmt.Sprintf("/dbmgmt/workflow/tickets/%d?project=%d", refID, projectID)
	case model.WorkflowRefDbAccessRequest:
		return fmt.Sprintf("/dbmgmt/apply/query?project=%d&highlight=%d", projectID, refID)
	case model.WorkflowRefDbAppUserRequest:
		return fmt.Sprintf("/dbmgmt/apply/app-user?project=%d&highlight=%d", projectID, refID)
	case model.WorkflowRefCicdReleaseRun:
		return fmt.Sprintf("/cicd/release-records?project=%d&release=%d", projectID, refID)
	case model.WorkflowRefAlertEvent:
		return fmt.Sprintf("/alert-events?highlight=%d", refID)
	case model.WorkflowRefAiToolApproval:
		return fmt.Sprintf("/ai/approvals?highlight=%d", refID)
	}
	switch domain {
	case model.WorkflowDomainIncident:
		return fmt.Sprintf("/workflow/inbox?ticket=%d", refID) // refID 此处为业务 ref；inbox 同时匹配 workflow_ticket_id / ref_id
	}
	_ = ticketType
	return fmt.Sprintf("/workflow/inbox?project=%d", projectID)
}
