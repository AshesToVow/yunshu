package repository

import (
	"context"
	"strings"
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

// DeleteInstance 删除实例并清理其全部业务数据，避免 grants/账号/工单成为孤儿。
// 审计日志（db_audit_logs）保留，instance_id 可空且属于历史留痕。
func (r *DbmgmtRepository) DeleteInstance(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ticketIDs []uint
		if err := tx.Model(&model.DbSqlTicket{}).Where("instance_id = ?", id).Pluck("id", &ticketIDs).Error; err != nil {
			return err
		}
		if len(ticketIDs) > 0 {
			if err := tx.Where("ticket_id IN ?", ticketIDs).Delete(&model.DbSqlTicketStep{}).Error; err != nil {
				return err
			}
		}
		var accessIDs []uint
		if err := tx.Model(&model.DbAccessRequest{}).Where("instance_id = ?", id).Pluck("id", &accessIDs).Error; err != nil {
			return err
		}
		if len(accessIDs) > 0 {
			if err := tx.Where("access_request_id IN ?", accessIDs).Delete(&model.DbAccessRequestStep{}).Error; err != nil {
				return err
			}
		}
		var appUserIDs []uint
		if err := tx.Model(&model.DbAppUserRequest{}).Where("instance_id = ?", id).Pluck("id", &appUserIDs).Error; err != nil {
			return err
		}
		if len(appUserIDs) > 0 {
			if err := tx.Where("app_user_request_id IN ?", appUserIDs).Delete(&model.DbAppUserRequestStep{}).Error; err != nil {
				return err
			}
		}
		for _, m := range []any{
			&model.DbSqlExecution{},
			&model.DbSqlTicket{},
			&model.DbAccessRequest{},
			&model.DbAppUserRequest{},
			&model.DbAccessGrant{},
			&model.DbInstanceAccount{},
			&model.DbColumnMaskRule{},
		} {
			if err := tx.Where("instance_id = ?", id).Delete(m).Error; err != nil {
				return err
			}
		}
		// 解除从库绑定，避免残留指向已删除主库的引用。
		if err := tx.Model(&model.DbInstance{}).Where("primary_instance_id = ?", id).Update("primary_instance_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&model.DbInstance{}, id).Error
	})
}

// ListInstanceLinkedRefIDs 列出实例下所有工单/申请 ID，供删除前关闭统一工单。
func (r *DbmgmtRepository) ListInstanceLinkedRefIDs(ctx context.Context, instanceID uint) (sqlTicketIDs, accessRequestIDs, appUserRequestIDs []uint, err error) {
	db := r.db.WithContext(ctx)
	if err = db.Model(&model.DbSqlTicket{}).Where("instance_id = ?", instanceID).Pluck("id", &sqlTicketIDs).Error; err != nil {
		return nil, nil, nil, err
	}
	if err = db.Model(&model.DbAccessRequest{}).Where("instance_id = ?", instanceID).Pluck("id", &accessRequestIDs).Error; err != nil {
		return nil, nil, nil, err
	}
	if err = db.Model(&model.DbAppUserRequest{}).Where("instance_id = ?", instanceID).Pluck("id", &appUserRequestIDs).Error; err != nil {
		return nil, nil, nil, err
	}
	return sqlTicketIDs, accessRequestIDs, appUserRequestIDs, nil
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

// ListInstancesForVisibility 返回全部匹配项（不分页），供服务层先按可见性过滤再分页，
// 避免「先分页后过滤」导致页面条数与 total 失真。
func (r *DbmgmtRepository) ListInstancesForVisibility(ctx context.Context, p DbInstanceListParams) ([]model.DbInstance, error) {
	q := r.db.WithContext(ctx).Model(&model.DbInstance{}).Where("project_id = ?", p.ProjectID)
	if p.Env != "" {
		q = q.Where("env = ?", p.Env)
	}
	if p.Keyword != "" {
		like := "%" + p.Keyword + "%"
		q = q.Where("name LIKE ? OR host LIKE ? OR tags LIKE ?", like, like, like)
	}
	var list []model.DbInstance
	err := q.Order("id DESC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) ListAllInstances(ctx context.Context) ([]model.DbInstance, error) {
	var list []model.DbInstance
	err := r.db.WithContext(ctx).Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) CountReplicasByPrimary(ctx context.Context, projectID, primaryID uint) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.DbInstance{}).
		Where("project_id = ? AND primary_instance_id = ? AND role = ?", projectID, primaryID, model.DbInstanceRoleReplica).
		Count(&n).Error
	return n, err
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

// GetGrantBySourceRequest 按“来源权限申请”查授权，用于终审幂等。
func (r *DbmgmtRepository) GetGrantBySourceRequest(ctx context.Context, projectID, requestID uint) (*model.DbAccessGrant, error) {
	var g model.DbAccessGrant
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND source_request_id = ?", projectID, requestID).
		First(&g).Error
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

// ClaimAccessRequestStep 原子认领待审批步骤：仅 pending 可更新，并发下只允许一个审批人成功。
func (r *DbmgmtRepository) ClaimAccessRequestStep(ctx context.Context, stepID, requestID uint, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbAccessRequestStep{}).
		Where("id = ? AND access_request_id = ? AND status = ?", stepID, requestID, model.DbApprovalStepPending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateAccessRequestFieldsIfStatus 条件更新申请主表，避免并发审批互相覆盖终态。
func (r *DbmgmtRepository) UpdateAccessRequestFieldsIfStatus(ctx context.Context, requestID uint, status string, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbAccessRequest{}).
		Where("id = ? AND status = ?", requestID, status).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateAccessRequestFields 定向更新申请字段（如建库失败回写），不整行 Save。
func (r *DbmgmtRepository) UpdateAccessRequestFields(ctx context.Context, requestID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.DbAccessRequest{}).
		Where("id = ?", requestID).
		Updates(fields).Error
}

// --- SQL Ticket ---

func (r *DbmgmtRepository) CreateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error {
	// IsBackup=false 是合法值；GORM 对带 default 的零值会跳过写入并落到库默认 true，须显式 Select。
	return r.db.WithContext(ctx).
		Select(
			"ProjectID", "InstanceID", "TicketType", "SubmitterUserID", "SubmitterName",
			"DatabaseName", "SqlText", "SqlFileRef", "AuditMode", "RiskLevel", "SyntaxType",
			"IsBackup", "ParsedOpsJSON", "ReviewJSON", "ExecuteJSON", "Reason", "Status", "RequestJSON",
		).
		Create(t).Error
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
	Keyword         string
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
	if kw := strings.TrimSpace(p.Keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(reason LIKE ? OR sql_text LIKE ?)", like, like)
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

// ClaimSqlTicketStep 原子认领待审批步骤：仅 pending 可更新。
func (r *DbmgmtRepository) ClaimSqlTicketStep(ctx context.Context, stepID, ticketID uint, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbSqlTicketStep{}).
		Where("id = ? AND ticket_id = ? AND status = ?", stepID, ticketID, model.DbApprovalStepPending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateSqlTicketFieldsIfStatus 条件更新工单主表，避免并发审批/执行互相覆盖终态。
func (r *DbmgmtRepository) UpdateSqlTicketFieldsIfStatus(ctx context.Context, ticketID uint, status string, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbSqlTicket{}).
		Where("id = ? AND status = ?", ticketID, status).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// FailStaleExecutingSqlTickets 把长时间停在 executing 的工单标记为「结果未知」，
// 覆盖进程崩溃/HTTP 超时导致的状态卡死；不自动重试，避免重复执行。
func (r *DbmgmtRepository) FailStaleExecutingSqlTickets(ctx context.Context, olderThan time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbSqlTicket{}).
		Where("status = ? AND updated_at < ?", model.DbTicketStatusExecuting, olderThan).
		Updates(map[string]any{"status": model.DbTicketStatusResultUnknown, "updated_at": time.Now()})
	return res.RowsAffected, res.Error
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

// ClaimAppUserRequestStep 原子认领待审批步骤：仅 pending 可更新。
func (r *DbmgmtRepository) ClaimAppUserRequestStep(ctx context.Context, stepID, requestID uint, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbAppUserRequestStep{}).
		Where("id = ? AND app_user_request_id = ? AND status = ?", stepID, requestID, model.DbApprovalStepPending).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateAppUserRequestFieldsIfStatus 条件更新申请主表，避免并发审批互相覆盖终态。
func (r *DbmgmtRepository) UpdateAppUserRequestFieldsIfStatus(ctx context.Context, requestID uint, status string, fields map[string]any) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.DbAppUserRequest{}).
		Where("id = ? AND status = ?", requestID, status).
		Updates(fields)
	return res.RowsAffected, res.Error
}

// UpdateAppUserRequestFields 按字段更新申请（map 更新不会自动写 updated_at）。
func (r *DbmgmtRepository) UpdateAppUserRequestFields(ctx context.Context, requestID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.DbAppUserRequest{}).
		Where("id = ?", requestID).Updates(fields).Error
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

func (r *DbmgmtRepository) ListColumnMaskRules(ctx context.Context, instanceID uint) ([]model.DbColumnMaskRule, error) {
	var list []model.DbColumnMaskRule
	err := r.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("id ASC").Find(&list).Error
	return list, err
}

func (r *DbmgmtRepository) GetColumnMaskRule(ctx context.Context, instanceID, id uint) (*model.DbColumnMaskRule, error) {
	var row model.DbColumnMaskRule
	err := r.db.WithContext(ctx).Where("instance_id = ? AND id = ?", instanceID, id).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DbmgmtRepository) FindColumnMaskRule(ctx context.Context, instanceID uint, schema, table, column string) (*model.DbColumnMaskRule, error) {
	var row model.DbColumnMaskRule
	err := r.db.WithContext(ctx).Where(
		"instance_id = ? AND schema_name = ? AND table_name = ? AND column_name = ?",
		instanceID, schema, table, column,
	).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DbmgmtRepository) CreateColumnMaskRule(ctx context.Context, rule *model.DbColumnMaskRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *DbmgmtRepository) UpdateColumnMaskRule(ctx context.Context, rule *model.DbColumnMaskRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *DbmgmtRepository) DeleteColumnMaskRule(ctx context.Context, instanceID, id uint) error {
	res := r.db.WithContext(ctx).Where("instance_id = ? AND id = ?", instanceID, id).Delete(&model.DbColumnMaskRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
