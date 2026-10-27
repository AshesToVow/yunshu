package dbmgmt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	workflowsvc "yunshu/internal/service/workflow"

	"gorm.io/gorm"
)

type AccessRequestItem struct {
	ID              uint     `json:"id"`
	ProjectID       uint     `json:"project_id"`
	InstanceID      uint     `json:"instance_id"`
	InstanceName    string   `json:"instance_name,omitempty"`
	RequesterUserID uint     `json:"requester_user_id"`
	RequesterName   string   `json:"requester_name"`
	DatabaseName    string   `json:"database_name"`
	TableNames      []string `json:"table_names"`
	ScopeType       string   `json:"scope_type,omitempty"`
	CanConnect      bool     `json:"can_connect"`
	CanQuery        bool     `json:"can_query"`
	CanDML          bool     `json:"can_dml"`
	CanDDL          bool     `json:"can_ddl"`
	CanExport       bool     `json:"can_export"`
	CanImport       bool     `json:"can_import"`
	Privileges      []string `json:"privileges,omitempty"`
	Reason           string   `json:"reason"`
	Status           string   `json:"status"`
	CurrentStageName string   `json:"current_stage_name,omitempty"`
	MineStatus       string   `json:"mine_status,omitempty"`
	IsFinalApproval  bool     `json:"is_final_approval,omitempty"`
	ExpiresAt        *string  `json:"expires_at,omitempty"`
	QueryLimitNum    int      `json:"query_limit_num"`
	CreateMeta       *AccessRequestMetaItem `json:"create_meta,omitempty"`
	ProvisionError   string   `json:"provision_error,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type AccessRequestCreateRequest struct {
	InstanceID   uint     `json:"instance_id" binding:"required"`
	ScopeType    string   `json:"scope_type"`
	DatabaseName string   `json:"database_name"`
	TableNames   []string `json:"table_names"`
	CanConnect   bool     `json:"can_connect"`
	CanQuery     bool     `json:"can_query"`
	CanDML       bool     `json:"can_dml"`
	CanDDL       bool     `json:"can_ddl"`
	CanExport    bool     `json:"can_export"`
	CanImport    bool     `json:"can_import"`
	Privileges   []string `json:"privileges,omitempty"`
	Reason       string   `json:"reason" binding:"required"`
	ExpiresAt    *string  `json:"expires_at,omitempty"`
	QueryLimitNum int     `json:"query_limit_num"`
	Charset      string   `json:"charset"`
	Collation    string   `json:"collation"`
	DevOwnerUserID uint   `json:"dev_owner_user_id"`
	DbaUserID    uint     `json:"dba_user_id"`
	GrantHosts   string   `json:"grant_hosts"`
}

type AccessApproveRequest struct {
	Comment   string  `json:"comment"`
	ExpiresAt *string `json:"expires_at"`
}

type AccessRequestListQuery struct {
	ProjectID       uint
	Status          string `form:"status"`
	RequesterUserID uint   `form:"requester_user_id"`
	Mine            bool   `form:"mine"`
	MineScope       string `form:"mine_scope"`
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	MineViewer      *auth.CurrentUser `form:"-"`
}

func (s *Service) toAccessRequestItem(ctx context.Context, req model.DbAccessRequest) AccessRequestItem {
	privs := parsePrivilegesJSON(req.PrivilegesJSON)
	if len(privs) == 0 {
		privs = privilegesFromFlags(req.CanQuery, req.CanDML, req.CanDDL, req.CanExport, req.CanImport)
	}
	scopeType := "database"
	if hasPrivilege(privs, "create_database") && len(parseTableNamesJSON(req.TableNamesJSON)) == 0 {
		scopeType = "new_database"
	} else if len(parseTableNamesJSON(req.TableNamesJSON)) > 0 {
		scopeType = "table"
	}
	item := AccessRequestItem{
		ID: req.ID, ProjectID: req.ProjectID, InstanceID: req.InstanceID,
		RequesterUserID: req.RequesterUserID, RequesterName: req.RequesterName,
		DatabaseName: req.DatabaseName, TableNames: parseTableNamesJSON(req.TableNamesJSON),
		ScopeType: scopeType,
		CanConnect: req.CanConnect, CanQuery: req.CanQuery, CanDML: req.CanDML, CanDDL: req.CanDDL,
		CanExport: req.CanExport, CanImport: req.CanImport, Privileges: privs,
		Reason: req.Reason, Status: req.Status,
		QueryLimitNum: req.QueryLimitNum,
		ExpiresAt: formatTimeRFC3339(req.ExpiresAt),
		ProvisionError: req.ProvisionError,
		CreatedAt: req.CreatedAt.Format(time.RFC3339),
	}
	if inst, err := s.repo.GetInstance(ctx, req.InstanceID); err == nil {
		item.InstanceName = inst.Name
	}
	item.CreateMeta = toAccessRequestMetaItem(parseAccessRequestMeta(req.MetaJSON))
	return item
}

func (s *Service) CreateAccessRequest(ctx context.Context, projectID uint, body AccessRequestCreateRequest, actor *auth.CurrentUser) (*AccessRequestItem, error) {
	if _, err := s.repo.GetInstanceInProject(ctx, projectID, body.InstanceID); err != nil {
		return nil, err
	}
	expiresAt, err := parseOptionalExpiresAt(body.ExpiresAt)
	if err != nil {
		return nil, err
	}
	privs := normalizePrivileges(body.Privileges)
	scope := strings.TrimSpace(body.ScopeType)
	if scope == "" {
		scope = "database"
	}
	switch scope {
	case "new_database":
		dbName := strings.TrimSpace(body.DatabaseName)
		if !isValidDbIdentifier(dbName) {
			return nil, constants.ErrBadRequestWithMsg("请填写合法的新建库名（字母/数字/下划线）")
		}
		if len(dbName) > 50 {
			return nil, constants.ErrBadRequestWithMsg("库名最长 50 个字符")
		}
		if body.DevOwnerUserID == 0 {
			return nil, constants.ErrBadRequestWithMsg("请选择开发负责人")
		}
		if body.DbaUserID == 0 {
			return nil, constants.ErrBadRequestWithMsg("请选择 DBA")
		}
		if strings.TrimSpace(body.GrantHosts) == "" {
			return nil, constants.ErrBadRequestWithMsg("请填写授权 IP")
		}
		charset := strings.TrimSpace(body.Charset)
		if charset == "" {
			charset = "utf8mb4"
		}
		collation := strings.TrimSpace(body.Collation)
		if collation == "" {
			collation = defaultCollationForCharset(charset)
		}
		if err := validateDatabaseCreateMeta(charset, collation); err != nil {
			return nil, err
		}
		if !hasPrivilege(privs, "create_database") {
			privs = append(privs, "create_database")
		}
		body.DatabaseName = dbName
		body.TableNames = nil
	case "table":
		if strings.TrimSpace(body.DatabaseName) == "" {
			return nil, constants.ErrBadRequestWithMsg("请选择目标库")
		}
		if len(body.TableNames) == 0 {
			return nil, constants.ErrBadRequestWithMsg("表级申请须选择至少一个表")
		}
	default:
		if strings.TrimSpace(body.DatabaseName) == "" {
			return nil, constants.ErrBadRequestWithMsg("请选择目标库")
		}
		body.TableNames = nil
	}
	canQuery, canDML, canDDL, canExport, canImport := flagsFromPrivileges(privs)
	if len(privs) == 0 {
		canQuery, canDML, canDDL, canExport, canImport = body.CanQuery, body.CanDML, body.CanDDL, body.CanExport, body.CanImport
		privs = privilegesFromFlags(canQuery, canDML, canDDL, canExport, canImport)
	}
	if len(privs) == 0 {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一项权限")
	}
	queryLimit := body.QueryLimitNum
	if queryLimit <= 0 {
		queryLimit = 1000
	}
	if !canQuery || canDML || canDDL || canExport || canImport {
		queryLimit = 0
	} else if queryLimit > 100000 {
		return nil, constants.ErrBadRequestWithMsg("查询行数上限不能超过 100000")
	}
	tb, _ := json.Marshal(body.TableNames)
	meta := accessRequestMeta{
		Charset: strings.TrimSpace(body.Charset), Collation: strings.TrimSpace(body.Collation),
		DevOwnerUserID: body.DevOwnerUserID, DbaUserID: body.DbaUserID,
		GrantHosts: strings.TrimSpace(body.GrantHosts),
	}
	if scope == "new_database" {
		if meta.Charset == "" {
			meta.Charset = "utf8mb4"
		}
		if meta.Collation == "" {
			meta.Collation = defaultCollationForCharset(meta.Charset)
		}
		if u, err := s.userRepo.GetByID(ctx, body.DevOwnerUserID); err == nil {
			meta.DevOwnerName = userDisplayName(u)
		}
		if u, err := s.userRepo.GetByID(ctx, body.DbaUserID); err == nil {
			meta.DbaName = userDisplayName(u)
		}
	}
	req := &model.DbAccessRequest{
		ProjectID: projectID, InstanceID: body.InstanceID,
		RequesterUserID: actorUserID(actor), RequesterName: actorUsername(actor),
		DatabaseName: body.DatabaseName, TableNamesJSON: string(tb),
		CanConnect: body.CanConnect, CanQuery: canQuery, CanDML: canDML, CanDDL: canDDL,
		CanExport: canExport, CanImport: canImport, PrivilegesJSON: marshalPrivilegesJSON(privs),
		QueryLimitNum: queryLimit,
		MetaJSON:      marshalAccessRequestMeta(meta),
		Reason:        strings.TrimSpace(body.Reason), Status: model.DbAccessRequestStatusPending,
		ExpiresAt: expiresAt,
	}
	if err := s.repo.CreateAccessRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := s.initAccessRequestSteps(ctx, req); err != nil {
		req.Status = model.DbAccessRequestStatusRejected
		if uerr := s.repo.UpdateAccessRequest(ctx, req); uerr != nil {
			slog.Default().With("component", "dbmgmt").Error("mark access request rejected failed", "request_id", req.ID, "error", uerr)
		}
		return nil, err
	}
	iid := body.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_create", map[string]any{
		"request_id": req.ID, "database": body.DatabaseName, "scope": scope, "privileges": privs,
	})
	item := s.toAccessRequestItem(ctx, *req)
	return &item, nil
}

func (s *Service) ListAccessRequests(ctx context.Context, q AccessRequestListQuery) (*pagination.Result[AccessRequestItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if q.Mine && q.MineViewer != nil {
		scope := strings.TrimSpace(q.MineScope)
		if scope == "" {
			scope = "all"
		}
		list, total, err := s.repo.ListAccessRequestsMine(ctx, repository.DbMineListParams{
			ProjectID:    q.ProjectID,
			Status:       q.Status,
			Scope:        scope,
			UserID:       actorUserID(q.MineViewer),
			IsSuperAdmin: auth.IsSuperAdminRole(q.MineViewer.RoleCodes),
			Page:         page,
			PageSize:     pageSize,
		})
		if err != nil {
			return nil, err
		}
		items := make([]AccessRequestItem, 0, len(list))
		for _, req := range list {
			items = append(items, s.toAccessRequestItem(ctx, req))
		}
		s.enrichAccessRequestMineStatus(ctx, items, q.MineViewer)
		return paginate(items, total, page, pageSize), nil
	}
	list, total, err := s.repo.ListAccessRequests(ctx, repository.DbAccessRequestListParams{
		ProjectID: q.ProjectID, Status: q.Status, RequesterUserID: q.RequesterUserID,
		Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]AccessRequestItem, 0, len(list))
	for _, req := range list {
		items = append(items, s.toAccessRequestItem(ctx, req))
	}
	return paginate(items, total, q.Page, q.PageSize), nil
}

func (s *Service) grantFromAccessRequest(ctx context.Context, req *model.DbAccessRequest) error {
	// 幂等：同一申请只生成一条授权（source_request_id 唯一索引兜底并发终审）。
	if _, err := s.repo.GetGrantBySourceRequest(ctx, req.ProjectID, req.ID); err == nil {
		return s.applyProvisionAfterGrant(ctx, req)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	rid := req.ID
	g := &model.DbAccessGrant{
		ProjectID: req.ProjectID, InstanceID: req.InstanceID,
		PrincipalKind: model.DbPrincipalUser, PrincipalRef: fmt.Sprintf("%d", req.RequesterUserID),
		DatabaseName: req.DatabaseName, TableNamesJSON: req.TableNamesJSON,
		CanConnect: req.CanConnect, CanQuery: req.CanQuery, CanDML: req.CanDML, CanDDL: req.CanDDL,
		CanExport: req.CanExport, CanImport: req.CanImport,
		QueryLimitNum:  req.QueryLimitNum,
		PrivilegesJSON: req.PrivilegesJSON,
		ExpiresAt:      req.ExpiresAt, Remark: fmt.Sprintf("来自权限申请 #%d", req.ID),
		SourceRequestID: &rid,
	}
	if err := s.repo.CreateGrant(ctx, g); err != nil {
		// 并发终审触发唯一索引冲突：复用已生成的授权继续建库，保持幂等。
		if existing, getErr := s.repo.GetGrantBySourceRequest(ctx, req.ProjectID, req.ID); getErr == nil && existing != nil {
			return s.applyProvisionAfterGrant(ctx, req)
		}
		return err
	}
	return s.applyProvisionAfterGrant(ctx, req)
}

// applyProvisionAfterGrant 执行建库等落地动作，并把失败结果回写到申请上。
// 授权此时已生效，建库失败不能整体回滚授权，但必须让审批人可见并可重试。
func (s *Service) applyProvisionAfterGrant(ctx context.Context, req *model.DbAccessRequest) error {
	if err := s.provisionDatabaseAfterGrant(ctx, req); err != nil {
		msg := truncateSQL(err.Error(), 900)
		if updErr := s.repo.UpdateAccessRequestFields(ctx, req.ID, map[string]any{"provision_error": msg}); updErr != nil {
			return errors.Join(err, updErr)
		}
		iid := req.InstanceID
		_ = s.writeAudit(ctx, req.ProjectID, &iid, &auth.CurrentUser{Username: "system"}, "access_request_provision_failed", map[string]any{
			"request_id": req.ID, "database": req.DatabaseName, "error": msg,
		})
		return constants.ErrBadRequestWithMsg("授权已生效，但自动建库失败：" + err.Error())
	}
	if strings.TrimSpace(req.ProvisionError) != "" {
		_ = s.repo.UpdateAccessRequestFields(ctx, req.ID, map[string]any{"provision_error": ""})
	}
	return nil
}

func (s *Service) ApproveAccessRequest(ctx context.Context, projectID, id uint, body AccessApproveRequest, actor *auth.CurrentUser) error {
	req, err := s.repo.GetAccessRequestInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if req.Status != model.DbAccessRequestStatusPending {
		return constants.ErrBadRequestWithMsg("申请已结束")
	}
	comment := strings.TrimSpace(body.Comment)
	hasWorkflow := s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbAccessRequest, id)
	// 有效期调整只允许最后一环审批人操作。这里只做「是否最后一环」与格式校验，
	// 真正写入必须等审批人身份校验通过（工作流分支在最终审批回调里落库）。
	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		if hasWorkflow {
			if !s.isFinalWorkflowApproval(ctx, model.WorkflowRefDbAccessRequest, id) {
				return constants.ErrBadRequestWithMsg("仅最后一环审批人可调整授权有效期")
			}
		} else {
			steps, err := s.repo.ListAccessRequestSteps(ctx, id)
			if err != nil {
				return err
			}
			var cur *model.DbAccessRequestStep
			for i := range steps {
				if steps[i].Status == model.DbApprovalStepPending {
					cur = &steps[i]
					break
				}
			}
			if cur == nil {
				return constants.ErrBadRequestWithMsg("无待审批步骤")
			}
			if !isFinalAccessApprovalStep(steps, cur) {
				return constants.ErrBadRequestWithMsg("仅最后一环审批人可调整授权有效期")
			}
		}
		expiresAt, err = parseOptionalExpiresAt(body.ExpiresAt)
		if err != nil {
			return err
		}
	}
	if hasWorkflow {
		err := s.reviewDbmgmtViaWorkflow(ctx, model.WorkflowRefDbAccessRequest, id, true, comment, actor,
			func(d *workflowsvc.TicketDetail) error {
				if d.Status == model.WorkflowTicketStatusApproved {
					return s.finalizeAccessRequestApproval(ctx, projectID, id, expiresAt)
				}
				return nil
			})
		if err != nil {
			return workflowOrLegacyErr(err)
		}
		iid := req.InstanceID
		auditExpires := req.ExpiresAt
		if expiresAt != nil {
			auditExpires = expiresAt
		}
		_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_approve", map[string]any{
			"request_id": req.ID, "database": req.DatabaseName, "comment": comment, "expires_at": formatTimeRFC3339(auditExpires),
		})
		return nil
	}
	steps, err := s.repo.ListAccessRequestSteps(ctx, id)
	if err != nil {
		return err
	}
	var cur *model.DbAccessRequestStep
	for i := range steps {
		if steps[i].Status == model.DbApprovalStepPending {
			cur = &steps[i]
			break
		}
	}
	if cur == nil {
		return constants.ErrBadRequestWithMsg("无待审批步骤")
	}
	ok, err := s.userCanApproveStep(ctx, actor, cur.UserGroupID)
	if err != nil {
		return err
	}
	if !ok {
		return constants.ErrForbidden
	}
	if err := s.forbidSelfApprove(ctx, actor, req.RequesterUserID); err != nil {
		return err
	}
	now := time.Now()
	uid := actorUserID(actor)
	// 乐观锁：仅 pending 步骤可被认领，避免同一节点被并发审批两次。
	claimed, err := s.repo.ClaimAccessRequestStep(ctx, cur.ID, id, map[string]any{
		"status": model.DbApprovalStepApproved, "reviewer_user_id": uid,
		"reviewer_name": actorUsername(actor), "review_comment": comment, "reviewed_at": now,
		"updated_at": now,
	})
	if err != nil {
		return err
	}
	if claimed == 0 {
		return constants.ErrBadRequestWithMsg("该审批节点已被处理，请刷新后重试")
	}
	cur.Status = model.DbApprovalStepApproved
	cur.ReviewerUserID = &uid
	cur.ReviewerName = actorUsername(actor)
	cur.ReviewComment = comment
	cur.ReviewedAt = &now
	if err := s.advanceAccessRequestAfterApproval(ctx, req, cur, expiresAt); err != nil {
		return err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_approve", map[string]any{
		"request_id": req.ID, "database": req.DatabaseName, "comment": comment, "expires_at": formatTimeRFC3339(req.ExpiresAt),
	})
	return nil
}

func (s *Service) RejectAccessRequest(ctx context.Context, projectID, id uint, comment string, actor *auth.CurrentUser) error {
	req, err := s.repo.GetAccessRequestInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if req.Status != model.DbAccessRequestStatusPending {
		return constants.ErrBadRequestWithMsg("申请已结束")
	}
	if s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbAccessRequest, id) {
		err := s.reviewDbmgmtViaWorkflow(ctx, model.WorkflowRefDbAccessRequest, id, false, comment, actor, nil)
		if err != nil {
			return workflowOrLegacyErr(err)
		}
		iid := req.InstanceID
		_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_reject", map[string]any{
			"request_id": req.ID, "database": req.DatabaseName, "comment": strings.TrimSpace(comment),
		})
		return nil
	}
	steps, err := s.repo.ListAccessRequestSteps(ctx, id)
	if err != nil {
		return err
	}
	var cur *model.DbAccessRequestStep
	for i := range steps {
		if steps[i].Status == model.DbApprovalStepPending {
			cur = &steps[i]
			break
		}
	}
	if cur != nil {
		ok, err := s.userCanApproveStep(ctx, actor, cur.UserGroupID)
		if err != nil {
			return err
		}
		if !ok {
			return constants.ErrForbidden
		}
		now := time.Now()
		uid := actorUserID(actor)
		claimed, err := s.repo.ClaimAccessRequestStep(ctx, cur.ID, id, map[string]any{
			"status": model.DbApprovalStepRejected, "reviewer_user_id": uid,
			"reviewer_name": actorUsername(actor), "review_comment": strings.TrimSpace(comment),
			"reviewed_at": now, "updated_at": now,
		})
		if err != nil {
			return err
		}
		if claimed == 0 {
			return constants.ErrBadRequestWithMsg("该审批节点已被处理，请刷新后重试")
		}
	}
	n, err := s.repo.UpdateAccessRequestFieldsIfStatus(ctx, id, model.DbAccessRequestStatusPending, map[string]any{
		"status": model.DbAccessRequestStatusRejected, "updated_at": time.Now(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrBadRequestWithMsg("申请状态已变更，请刷新后重试")
	}
	req.Status = model.DbAccessRequestStatusRejected
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_reject", map[string]any{
		"request_id": req.ID, "database": req.DatabaseName, "comment": strings.TrimSpace(comment),
	})
	return nil
}
