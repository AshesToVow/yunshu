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

	"gorm.io/gorm"
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

	base := s.db.WithContext(ctx).
		Table("workflow_ticket_steps AS s").
		Select(`t.id AS workflow_ticket_id, s.id AS step_id, t.domain, t.ticket_type, t.project_id,
			t.title, t.status, s.stage_name AS current_stage_name, t.submitter_user_id, t.ref_type, t.ref_id,
			s.activated_at, t.created_at, s.status AS step_status, s.reviewer_user_id`).
		Joins("JOIN workflow_tickets t ON t.id = s.ticket_id").
		Where("t.deleted_at IS NULL AND s.deleted_at IS NULL")

	if len(domains) > 0 {
		base = base.Where("t.domain IN ?", domains)
	}
	if q.ProjectID != nil && *q.ProjectID > 0 {
		base = base.Where("t.project_id = ?", *q.ProjectID)
	}

	switch mineScope {
	case MineScopePending:
		base = base.Where("t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL",
			model.WorkflowTicketStatusPending, model.WorkflowStepPending)
		if !isSuper {
			base = base.Where(`(
				s.assignee_user_id = ? OR
				(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
					SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
				))
			)`, userID, userID)
		}
	case MineScopeDone:
		if userID == 0 {
			base = base.Where("1 = 0")
		} else {
			base = base.Where("s.reviewer_user_id = ? AND s.status IN ?", userID,
				[]string{model.WorkflowStepApproved, model.WorkflowStepRejected})
		}
	default: // all
		if !isSuper && userID > 0 {
			base = base.Where(`(
				s.assignee_user_id = ? OR s.reviewer_user_id = ? OR
				(s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
					SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
				))
			)`, userID, userID, userID)
		}
	}

	var total int64
	countQ := base.Session(&gorm.Session{})
	if err := countQ.Count(&total).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "workflow", "ListPendingForUser", err)
	}

	type row struct {
		WorkflowTicketID uint
		StepID           uint
		Domain           string
		TicketType       string
		ProjectID        uint
		Title            string
		Status           string
		CurrentStageName string
		SubmitterUserID  uint
		RefType          string
		RefID            uint
		ActivatedAt      *time.Time
		CreatedAt        time.Time
		StepStatus       string
		ReviewerUserID   *uint
	}
	var rows []row
	if err := base.Order("s.activated_at ASC, t.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
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
	for _, r := range rows {
		mineStatus := "mine_done"
		if r.StepStatus == model.WorkflowStepPending && r.ActivatedAt != nil {
			mineStatus = "mine_pending"
		}
		items = append(items, PendingTicketItem{
			WorkflowTicketID: r.WorkflowTicketID, StepID: r.StepID,
			Domain: r.Domain, TicketType: r.TicketType, ProjectID: r.ProjectID,
			Title: r.Title, Status: r.Status, CurrentStageName: r.CurrentStageName,
			SubmitterUserID: r.SubmitterUserID, SubmitterName: userNames[r.SubmitterUserID],
			RefType: r.RefType, RefID: r.RefID, DeepLink: buildDeepLink(r.Domain, r.TicketType, r.ProjectID, r.RefType, r.RefID),
			ActivatedAt: r.ActivatedAt, CreatedAt: r.CreatedAt, MineStatus: mineStatus,
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
	}
	switch domain {
	case model.WorkflowDomainIncident:
		return fmt.Sprintf("/workflow/inbox?ticket=%d", refID)
	}
	_ = ticketType
	return fmt.Sprintf("/workflow/inbox?project=%d", projectID)
}
