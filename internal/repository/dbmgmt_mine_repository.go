package repository

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

// DbMineListParams 我的审批列表（权限申请 / 应用用户申请）。
type DbMineListParams struct {
	ProjectID    uint
	Status       string
	Scope        string // pending | done | all
	UserID       uint
	IsSuperAdmin bool
	Page         int
	PageSize     int
}

// DbTicketMineListParams SQL 工单「我的」列表。
type DbTicketMineListParams struct {
	DbMineListParams
	TicketType string
	MineTab    string // approval | execution
}

// WorkflowApprovalReminderRow 统一工单超时催办查询行。
type WorkflowApprovalReminderRow struct {
	StepID           uint
	TicketID         uint
	StageName        string
	ActivatedAt      time.Time
	LastRemindedAt   *time.Time
	UserGroupID      *uint
	AssigneeUserID   *uint
	AssigneeRuleType string
	Domain           string
	RefID            uint
	Title            string
	TicketType       string
}

func (r *DbmgmtRepository) GetAccessRequest(ctx context.Context, id uint) (*model.DbAccessRequest, error) {
	var req model.DbAccessRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	return &req, err
}

func (r *DbmgmtRepository) GetSqlTicket(ctx context.Context, id uint) (*model.DbSqlTicket, error) {
	var t model.DbSqlTicket
	err := r.db.WithContext(ctx).First(&t, id).Error
	return &t, err
}

func (r *DbmgmtRepository) GetAppUserRequest(ctx context.Context, id uint) (*model.DbAppUserRequest, error) {
	var req model.DbAppUserRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	return &req, err
}

func (r *DbmgmtRepository) ClaimSqlTicketExecuting(ctx context.Context, projectID, ticketID uint) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbSqlTicket{}).
		Where("id = ? AND project_id = ? AND status IN ?", ticketID, projectID,
			[]string{model.DbTicketStatusPendingExecution, model.DbTicketStatusApproved}).
		Update("status", model.DbTicketStatusExecuting)
	return res.RowsAffected, res.Error
}

func (r *DbmgmtRepository) ListAccessRequestStepsByRequestIDs(ctx context.Context, ids []uint) ([]model.DbAccessRequestStep, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var steps []model.DbAccessRequestStep
	err := r.db.WithContext(ctx).Where("access_request_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	return steps, err
}

func (r *DbmgmtRepository) ListSqlTicketStepsByTicketIDs(ctx context.Context, ids []uint) ([]model.DbSqlTicketStep, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var steps []model.DbSqlTicketStep
	err := r.db.WithContext(ctx).Where("ticket_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	return steps, err
}

func (r *DbmgmtRepository) ListAppUserRequestStepsByRequestIDs(ctx context.Context, ids []uint) ([]model.DbAppUserRequestStep, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var steps []model.DbAppUserRequestStep
	err := r.db.WithContext(ctx).Where("app_user_request_id IN ?", ids).Order("sort_order ASC, id ASC").Find(&steps).Error
	return steps, err
}

func (r *DbmgmtRepository) ListAccessRequestsMine(ctx context.Context, p DbMineListParams) ([]model.DbAccessRequest, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	dbq := r.db.WithContext(ctx).Model(&model.DbAccessRequest{}).Where("project_id = ?", p.ProjectID)
	if st := strings.TrimSpace(p.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	scope := strings.TrimSpace(p.Scope)
	if scope == "" {
		scope = "all"
	}
	switch scope {
	case "pending":
		dbq = dbq.Where("EXISTS (?)", r.accessRequestPendingSubquery(p.UserID, p.IsSuperAdmin))
	case "done":
		dbq = dbq.Where("EXISTS (?)", r.accessRequestDoneSubquery(p.UserID))
	default:
		dbq = dbq.Where("EXISTS (?) OR EXISTS (?)",
			r.accessRequestPendingSubquery(p.UserID, p.IsSuperAdmin),
			r.accessRequestDoneSubquery(p.UserID))
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbAccessRequest
	err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) ListAppUserRequestsMine(ctx context.Context, p DbMineListParams) ([]model.DbAppUserRequest, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	dbq := r.db.WithContext(ctx).Model(&model.DbAppUserRequest{}).Where("project_id = ?", p.ProjectID)
	if st := strings.TrimSpace(p.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	scope := strings.TrimSpace(p.Scope)
	if scope == "" {
		scope = "all"
	}
	switch scope {
	case "pending":
		dbq = dbq.Where("EXISTS (?)", r.appUserRequestPendingSubquery(p.UserID, p.IsSuperAdmin))
	case "done":
		dbq = dbq.Where("EXISTS (?)", r.appUserRequestDoneSubquery(p.UserID))
	default:
		dbq = dbq.Where("EXISTS (?) OR EXISTS (?)",
			r.appUserRequestPendingSubquery(p.UserID, p.IsSuperAdmin),
			r.appUserRequestDoneSubquery(p.UserID))
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbAppUserRequest
	err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) ListSqlTicketsMine(ctx context.Context, p DbTicketMineListParams) ([]model.DbSqlTicket, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	dbq := r.db.WithContext(ctx).Model(&model.DbSqlTicket{}).Where("project_id = ?", p.ProjectID)
	if tt := strings.TrimSpace(p.TicketType); tt != "" {
		dbq = dbq.Where("ticket_type = ?", tt)
	}
	if st := strings.TrimSpace(p.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	scope := strings.TrimSpace(p.Scope)
	if scope == "" {
		scope = "all"
	}
	mineTab := strings.TrimSpace(p.MineTab)
	if mineTab == "" {
		mineTab = "approval"
	}
	if mineTab == "execution" {
		switch scope {
		case "pending":
			if p.UserID == 0 {
				dbq = dbq.Where("1 = 0")
			} else {
				dbq = dbq.Where("status = ? AND submitter_user_id = ?", model.DbTicketStatusPendingExecution, p.UserID)
			}
		case "done":
			if p.UserID == 0 {
				dbq = dbq.Where("1 = 0")
			} else {
				dbq = dbq.Where("submitter_user_id = ?", p.UserID).
					Where("status IN ?", []string{model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting})
			}
		default:
			pending := r.db.Table("db_sql_tickets AS r").
				Select("1").
				Where("r.id = db_sql_tickets.id").
				Where("r.submitter_user_id = ?", p.UserID).
				Where("r.status = ?", model.DbTicketStatusPendingExecution)
			done := r.db.Table("db_sql_tickets AS r").
				Select("1").
				Where("r.id = db_sql_tickets.id").
				Where("r.submitter_user_id = ?", p.UserID).
				Where("r.status IN ?", []string{model.DbTicketStatusSuccess, model.DbTicketStatusFailed, model.DbTicketStatusExecuting})
			dbq = dbq.Where("EXISTS (?) OR EXISTS (?)", pending, done)
		}
	} else {
		switch scope {
		case "pending":
			dbq = dbq.Where("EXISTS (?)", r.ticketPendingSubquery(p.UserID, p.IsSuperAdmin))
		case "done":
			dbq = dbq.Where("EXISTS (?)", r.ticketDoneSubquery(p.UserID))
		default:
			dbq = dbq.Where("EXISTS (?) OR EXISTS (?)",
				r.ticketPendingSubquery(p.UserID, p.IsSuperAdmin),
				r.ticketDoneSubquery(p.UserID))
		}
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbSqlTicket
	err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) accessRequestPendingSubquery(userID uint, isSuperAdmin bool) *gorm.DB {
	if isSuperAdmin {
		return r.db.Table("db_access_requests AS r").
			Select("1").
			Where("r.id = db_access_requests.id").
			Where("r.status = ?", model.DbAccessRequestStatusPending)
	}
	wf := r.workflowPendingSubquery(userID, model.WorkflowRefDbAccessRequest, "db_access_requests.id")
	currentStep := r.db.Table("db_access_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.access_request_id = db_access_requests.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_access_request_steps s2
			WHERE s2.access_request_id = db_access_requests.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return r.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (r *DbmgmtRepository) accessRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return r.db.Table("db_access_requests AS r").Select("1").Where("1 = 0")
	}
	wf := r.workflowDoneSubquery(userID, model.WorkflowRefDbAccessRequest, "db_access_requests.id")
	acted := r.db.Table("db_access_request_steps AS s").
		Select("1").
		Where("s.access_request_id = db_access_requests.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return r.db.Table("db_access_requests AS r").
		Select("1").
		Where("r.id = db_access_requests.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (r *DbmgmtRepository) ticketPendingSubquery(userID uint, isSuperAdmin bool) *gorm.DB {
	if isSuperAdmin {
		return r.db.Table("db_sql_tickets AS r").
			Select("1").
			Where("r.id = db_sql_tickets.id").
			Where("r.status = ?", model.DbTicketStatusPendingApproval)
	}
	wf := r.workflowPendingSubquery(userID, model.WorkflowRefDbSqlTicket, "db_sql_tickets.id")
	currentStep := r.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.ticket_id = db_sql_tickets.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_sql_ticket_steps s2
			WHERE s2.ticket_id = db_sql_tickets.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return r.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("r.status = ?", model.DbTicketStatusPendingApproval).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (r *DbmgmtRepository) ticketDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return r.db.Table("db_sql_tickets AS r").Select("1").Where("1 = 0")
	}
	wf := r.workflowDoneSubquery(userID, model.WorkflowRefDbSqlTicket, "db_sql_tickets.id")
	acted := r.db.Table("db_sql_ticket_steps AS s").
		Select("1").
		Where("s.ticket_id = db_sql_tickets.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return r.db.Table("db_sql_tickets AS r").
		Select("1").
		Where("r.id = db_sql_tickets.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (r *DbmgmtRepository) appUserRequestPendingSubquery(userID uint, isSuperAdmin bool) *gorm.DB {
	if isSuperAdmin {
		return r.db.Table("db_app_user_requests AS r").
			Select("1").
			Where("r.id = db_app_user_requests.id").
			Where("r.status = ?", model.DbAccessRequestStatusPending)
	}
	wf := r.workflowPendingSubquery(userID, model.WorkflowRefDbAppUserRequest, "db_app_user_requests.id")
	currentStep := r.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Joins("JOIN user_group_users AS ugu ON ugu.user_group_id = s.user_group_id AND ugu.user_id = ?", userID).
		Where("s.app_user_request_id = db_app_user_requests.id").
		Where("s.status = ?", model.DbApprovalStepPending).
		Where("s.user_group_id IS NOT NULL AND s.user_group_id > 0").
		Where(`s.sort_order = (
			SELECT MIN(s2.sort_order) FROM db_app_user_request_steps s2
			WHERE s2.app_user_request_id = db_app_user_requests.id AND s2.status = ?
		)`, model.DbApprovalStepPending)
	return r.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("r.status = ?", model.DbAccessRequestStatusPending).
		Where("EXISTS (?) OR EXISTS (?)", wf, currentStep)
}

func (r *DbmgmtRepository) appUserRequestDoneSubquery(userID uint) *gorm.DB {
	if userID == 0 {
		return r.db.Table("db_app_user_requests AS r").Select("1").Where("1 = 0")
	}
	wf := r.workflowDoneSubquery(userID, model.WorkflowRefDbAppUserRequest, "db_app_user_requests.id")
	acted := r.db.Table("db_app_user_request_steps AS s").
		Select("1").
		Where("s.app_user_request_id = db_app_user_requests.id").
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.DbApprovalStepApproved, model.DbApprovalStepRejected})
	return r.db.Table("db_app_user_requests AS r").
		Select("1").
		Where("r.id = db_app_user_requests.id").
		Where("EXISTS (?) OR EXISTS (?)", wf, acted)
}

func (r *DbmgmtRepository) workflowPendingSubquery(userID uint, refType, refIDCol string) *gorm.DB {
	return r.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = "+refIDCol+" AND t.deleted_at IS NULL", refType).
		Where("t.status = ? AND s.status = ? AND s.activated_at IS NOT NULL",
			model.WorkflowTicketStatusPending, model.WorkflowStepPending).
		Where(`(s.assignee_user_id = ? OR (s.user_group_id IS NOT NULL AND s.user_group_id > 0 AND EXISTS (
			SELECT 1 FROM user_group_users ugu WHERE ugu.user_group_id = s.user_group_id AND ugu.user_id = ?
		)))`, userID, userID)
}

func (r *DbmgmtRepository) workflowDoneSubquery(userID uint, refType, refIDCol string) *gorm.DB {
	return r.db.Table("workflow_tickets AS t").
		Select("1").
		Joins("JOIN workflow_ticket_steps AS s ON s.ticket_id = t.id AND s.deleted_at IS NULL").
		Where("t.ref_type = ? AND t.ref_id = "+refIDCol+" AND t.deleted_at IS NULL", refType).
		Where("s.reviewer_user_id = ?", userID).
		Where("s.status IN ?", []string{model.WorkflowStepApproved, model.WorkflowStepRejected})
}

func (r *DbmgmtRepository) GetActiveWorkflowStageName(ctx context.Context, refType string, refID uint) (string, error) {
	var stage string
	err := r.db.WithContext(ctx).Raw(`
SELECT s.stage_name FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.ref_type = ? AND t.ref_id = ? AND s.status = ? AND s.activated_at IS NOT NULL AND s.deleted_at IS NULL
ORDER BY s.sort_order ASC, s.id ASC LIMIT 1
`, refType, refID, model.WorkflowStepPending).Scan(&stage).Error
	return stage, err
}

func (r *DbmgmtRepository) ListWorkflowStepsByRef(ctx context.Context, refType string, refID uint) ([]model.WorkflowTicketStep, error) {
	var steps []model.WorkflowTicketStep
	err := r.db.WithContext(ctx).Raw(`
SELECT s.* FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.ref_type = ? AND t.ref_id = ? AND s.deleted_at IS NULL
ORDER BY s.sort_order ASC, s.id ASC
`, refType, refID).Scan(&steps).Error
	return steps, err
}

func (r *DbmgmtRepository) CountPendingWorkflowSteps(ctx context.Context, ticketID uint) (int64, error) {
	var pending int64
	err := r.db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).
		Where("ticket_id = ? AND status = ?", ticketID, model.WorkflowStepPending).
		Count(&pending).Error
	return pending, err
}

func (r *DbmgmtRepository) ListWorkflowApprovalReminderRows(ctx context.Context, domains []string) ([]WorkflowApprovalReminderRow, error) {
	if len(domains) == 0 {
		return nil, nil
	}
	var list []WorkflowApprovalReminderRow
	err := r.db.WithContext(ctx).Raw(`
SELECT s.id AS step_id, s.ticket_id, s.stage_name, s.activated_at, s.last_reminded_at,
       s.user_group_id, s.assignee_user_id, s.assignee_rule_type, t.domain, t.ref_id, t.title, t.ticket_type
FROM workflow_ticket_steps s
JOIN workflow_tickets t ON t.id = s.ticket_id AND t.deleted_at IS NULL
WHERE t.domain IN ? AND t.status = ?
  AND s.status = ? AND s.activated_at IS NOT NULL AND s.deleted_at IS NULL
`, domains, model.WorkflowTicketStatusPending, model.WorkflowStepPending).Scan(&list).Error
	return list, err
}

func (r *DbmgmtRepository) UpdateWorkflowStepLastRemindedAt(ctx context.Context, stepID uint, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.WorkflowTicketStep{}).
		Where("id = ?", stepID).
		Update("last_reminded_at", at).Error
}
