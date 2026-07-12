package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type DbmgmtRepository struct {
	db *gorm.DB
}

func NewDbmgmtRepository(db *gorm.DB) DbmgmtRepo {
	return &DbmgmtRepository{db: db}
}

// --- Instance ---

func (r *DbmgmtRepository) CreateInstance(ctx context.Context, inst *model.DbInstance) error {
	return r.db.WithContext(ctx).Create(inst).Error
}

func (r *DbmgmtRepository) UpdateInstance(ctx context.Context, inst *model.DbInstance) error {
	return r.db.WithContext(ctx).Save(inst).Error
}

func (r *DbmgmtRepository) DeleteInstance(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DbInstance{}, id).Error
}

func (r *DbmgmtRepository) GetInstance(ctx context.Context, id uint) (*model.DbInstance, error) {
	var inst model.DbInstance
	err := r.db.WithContext(ctx).First(&inst, id).Error
	return &inst, err
}

func (r *DbmgmtRepository) GetInstanceInProject(ctx context.Context, projectID, id uint) (*model.DbInstance, error) {
	var inst model.DbInstance
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&inst, id).Error
	return &inst, err
}

type DbInstanceListParams struct {
	ProjectID uint
	Env       string
	Keyword   string
	Page      int
	PageSize  int
}

func (r *DbmgmtRepository) ListInstances(ctx context.Context, p DbInstanceListParams) ([]model.DbInstance, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbInstance{}).Where("project_id = ?", p.ProjectID)
	if p.Env != "" {
		q = q.Where("env = ?", p.Env)
	}
	if p.Keyword != "" {
		like := "%" + p.Keyword + "%"
		q = q.Where("name LIKE ? OR host LIKE ? OR tags LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbInstance
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) ListAllInstances(ctx context.Context) ([]model.DbInstance, error) {
	var list []model.DbInstance
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

// --- Grant ---

func (r *DbmgmtRepository) CreateGrant(ctx context.Context, g *model.DbAccessGrant) error {
	return r.db.WithContext(ctx).Create(g).Error
}

func (r *DbmgmtRepository) UpdateGrant(ctx context.Context, g *model.DbAccessGrant) error {
	return r.db.WithContext(ctx).Save(g).Error
}

func (r *DbmgmtRepository) DeleteGrant(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.DbAccessGrant{}, id).Error
}

func (r *DbmgmtRepository) GetGrantInProject(ctx context.Context, projectID, id uint) (*model.DbAccessGrant, error) {
	var g model.DbAccessGrant
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&g, id).Error
	return &g, err
}

func (r *DbmgmtRepository) ListGrants(ctx context.Context, projectID uint, instanceID uint) ([]model.DbAccessGrant, error) {
	var list []model.DbAccessGrant
	q := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if instanceID > 0 {
		q = q.Where("instance_id = ?", instanceID)
	}
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) ListGrantsForPrincipal(ctx context.Context, projectID, instanceID uint, kind, ref string) ([]model.DbAccessGrant, error) {
	var list []model.DbAccessGrant
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND instance_id = ? AND principal_kind = ? AND principal_ref = ?", projectID, instanceID, kind, ref).
		Find(&list).Error
	return list, err
}

// --- Access Request ---

func (r *DbmgmtRepository) CreateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *DbmgmtRepository) UpdateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *DbmgmtRepository) GetAccessRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAccessRequest, error) {
	var req model.DbAccessRequest
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&req, id).Error
	return &req, err
}

type DbAccessRequestListParams struct {
	ProjectID       uint
	Status          string
	RequesterUserID uint
	Page            int
	PageSize        int
}

func (r *DbmgmtRepository) ListAccessRequests(ctx context.Context, p DbAccessRequestListParams) ([]model.DbAccessRequest, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbAccessRequest{}).Where("project_id = ?", p.ProjectID)
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.RequesterUserID > 0 {
		q = q.Where("requester_user_id = ?", p.RequesterUserID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbAccessRequest
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) CreateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DbmgmtRepository) ListAccessRequestSteps(ctx context.Context, requestID uint) ([]model.DbAccessRequestStep, error) {
	var list []model.DbAccessRequestStep
	err := r.db.WithContext(ctx).Where("access_request_id = ?", requestID).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) UpdateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

// --- SQL Ticket ---

func (r *DbmgmtRepository) CreateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *DbmgmtRepository) UpdateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *DbmgmtRepository) GetSqlTicketInProject(ctx context.Context, projectID, id uint) (*model.DbSqlTicket, error) {
	var t model.DbSqlTicket
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&t, id).Error
	return &t, err
}

type DbSqlTicketListParams struct {
	ProjectID       uint
	Status          string
	SubmitterUserID uint
	TicketType      string
	Page            int
	PageSize        int
}

func (r *DbmgmtRepository) ListSqlTickets(ctx context.Context, p DbSqlTicketListParams) ([]model.DbSqlTicket, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbSqlTicket{}).Where("project_id = ?", p.ProjectID)
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.SubmitterUserID > 0 {
		q = q.Where("submitter_user_id = ?", p.SubmitterUserID)
	}
	if p.TicketType != "" {
		q = q.Where("ticket_type = ?", p.TicketType)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbSqlTicket
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) CreateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DbmgmtRepository) ListSqlTicketSteps(ctx context.Context, ticketID uint) ([]model.DbSqlTicketStep, error) {
	var list []model.DbSqlTicketStep
	err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) UpdateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

// --- Execution & Audit ---

func (r *DbmgmtRepository) CreateSqlExecution(ctx context.Context, ex *model.DbSqlExecution) error {
	return r.db.WithContext(ctx).Create(ex).Error
}

type DbSqlExecutionListParams struct {
	ProjectID      uint
	InstanceID     uint
	ExecutorUserID uint
	QueryOnly      bool
	Page           int
	PageSize       int
}

func (r *DbmgmtRepository) ListSqlExecutions(ctx context.Context, p DbSqlExecutionListParams) ([]model.DbSqlExecution, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbSqlExecution{}).Where("project_id = ?", p.ProjectID)
	if p.InstanceID > 0 {
		q = q.Where("instance_id = ?", p.InstanceID)
	}
	if p.ExecutorUserID > 0 {
		q = q.Where("executor_user_id = ?", p.ExecutorUserID)
	}
	if p.QueryOnly {
		q = q.Where("risk_level = ?", model.DbRiskLow)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbSqlExecution
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) CreateAuditLog(ctx context.Context, log *model.DbAuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

type DbAuditLogListParams struct {
	ProjectID  uint
	InstanceID uint
	Action     string
	Page       int
	PageSize   int
}

func (r *DbmgmtRepository) ListAuditLogs(ctx context.Context, p DbAuditLogListParams) ([]model.DbAuditLog, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbAuditLog{}).Where("project_id = ?", p.ProjectID)
	if p.InstanceID > 0 {
		q = q.Where("instance_id = ?", p.InstanceID)
	}
	if p.Action != "" {
		q = q.Where("action = ?", p.Action)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbAuditLog
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// --- Approval Flow ---

func (r *DbmgmtRepository) ListApprovalFlowStages(ctx context.Context, projectID uint) ([]model.DbApprovalFlowStage, error) {
	var list []model.DbApprovalFlowStage
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) UpsertApprovalFlowStage(ctx context.Context, stage *model.DbApprovalFlowStage) error {
	return r.db.WithContext(ctx).Save(stage).Error
}

func (r *DbmgmtRepository) DeleteApprovalFlowStagesNotIn(ctx context.Context, projectID uint, keys []string) error {
	q := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if len(keys) > 0 {
		q = q.Where("stage_key NOT IN ?", keys)
	}
	return q.Delete(&model.DbApprovalFlowStage{}).Error
}

// --- App User Request ---

func (r *DbmgmtRepository) CreateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *DbmgmtRepository) UpdateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *DbmgmtRepository) GetAppUserRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAppUserRequest, error) {
	var req model.DbAppUserRequest
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&req, id).Error
	return &req, err
}

type DbAppUserRequestListParams struct {
	ProjectID       uint
	Status          string
	RequesterUserID uint
	Page            int
	PageSize        int
}

func (r *DbmgmtRepository) ListAppUserRequests(ctx context.Context, p DbAppUserRequestListParams) ([]model.DbAppUserRequest, int64, error) {
	page, pageSize := pagination.Normalize(p.Page, p.PageSize)
	q := r.db.WithContext(ctx).Model(&model.DbAppUserRequest{}).Where("project_id = ?", p.ProjectID)
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.RequesterUserID > 0 {
		q = q.Where("requester_user_id = ?", p.RequesterUserID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.DbAppUserRequest
	err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func (r *DbmgmtRepository) CreateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error {
	return r.db.WithContext(ctx).Create(step).Error
}

func (r *DbmgmtRepository) ListAppUserRequestSteps(ctx context.Context, requestID uint) ([]model.DbAppUserRequestStep, error) {
	var list []model.DbAppUserRequestStep
	err := r.db.WithContext(ctx).Where("app_user_request_id = ?", requestID).Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) UpdateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error {
	return r.db.WithContext(ctx).Save(step).Error
}

// --- Instance Account ---

func (r *DbmgmtRepository) CreateInstanceAccount(ctx context.Context, acc *model.DbInstanceAccount) error {
	return r.db.WithContext(ctx).Create(acc).Error
}

func (r *DbmgmtRepository) ListInstanceAccounts(ctx context.Context, projectID, instanceID uint) ([]model.DbInstanceAccount, error) {
	var list []model.DbInstanceAccount
	q := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if instanceID > 0 {
		q = q.Where("instance_id = ?", instanceID)
	}
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) GetInstanceAccount(ctx context.Context, projectID, id uint) (*model.DbInstanceAccount, error) {
	var acc model.DbInstanceAccount
	err := r.db.WithContext(ctx).Where("project_id = ?", projectID).First(&acc, id).Error
	return &acc, err
}

// ListPendingAccessStepsForReminder 列出待 SLA 提醒的权限申请步骤。
func (r *DbmgmtRepository) ListPendingAccessStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbAccessRequestStep, error) {
	cutoff := time.Now().Add(-sla)
	var list []model.DbAccessRequestStep
	err := r.db.WithContext(ctx).
		Where("status = ? AND activated_at IS NOT NULL AND activated_at <= ?", model.DbApprovalStepPending, cutoff).
		Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) ListPendingTicketStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbSqlTicketStep, error) {
	cutoff := time.Now().Add(-sla)
	var list []model.DbSqlTicketStep
	err := r.db.WithContext(ctx).
		Where("status = ? AND activated_at IS NOT NULL AND activated_at <= ?", model.DbApprovalStepPending, cutoff).
		Find(&list).Error
	return list, err
}
