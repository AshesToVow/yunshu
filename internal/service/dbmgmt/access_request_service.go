package dbmgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
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
	ExpiresAt        *string  `json:"expires_at,omitempty"`
	QueryLimitNum    int      `json:"query_limit_num"`
	CreateMeta       *AccessRequestMetaItem `json:"create_meta,omitempty"`
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
		Reason: strings.TrimSpace(body.Reason), Status: model.DbAccessRequestStatusPending,
		ExpiresAt: expiresAt,
	}
	if err := s.repo.CreateAccessRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := s.initAccessRequestSteps(ctx, req); err != nil {
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
		dbq := s.db.WithContext(ctx).Model(&model.DbAccessRequest{}).Where("project_id = ?", q.ProjectID)
		if st := strings.TrimSpace(q.Status); st != "" {
			dbq = dbq.Where("status = ?", st)
		}
		scope := strings.TrimSpace(q.MineScope)
		if scope == "" {
			scope = "all"
		}
		switch scope {
		case "pending":
			dbq = s.filterAccessRequestsApprovalPending(dbq, q.MineViewer)
		case "done":
			dbq = s.filterAccessRequestsApprovalDone(dbq, actorUserID(q.MineViewer))
		default:
			dbq = s.filterAccessRequestsApprovalMine(dbq, q.MineViewer)
		}
		var total int64
		if err := dbq.Count(&total).Error; err != nil {
			return nil, err
		}
		var list []model.DbAccessRequest
		if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
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
	g := &model.DbAccessGrant{
		ProjectID: req.ProjectID, InstanceID: req.InstanceID,
		PrincipalKind: model.DbPrincipalUser, PrincipalRef: fmt.Sprintf("%d", req.RequesterUserID),
		DatabaseName: req.DatabaseName, TableNamesJSON: req.TableNamesJSON,
		CanConnect: req.CanConnect, CanQuery: req.CanQuery, CanDML: req.CanDML, CanDDL: req.CanDDL,
		CanExport: req.CanExport, CanImport: req.CanImport,
		QueryLimitNum: req.QueryLimitNum,
		PrivilegesJSON: req.PrivilegesJSON,
		ExpiresAt: req.ExpiresAt, Remark: fmt.Sprintf("来自权限申请 #%d", req.ID),
	}
	g.PrincipalRef = fmt.Sprintf("%d", req.RequesterUserID)
	if err := s.repo.CreateGrant(ctx, g); err != nil {
		return err
	}
	_ = s.provisionDatabaseAfterGrant(ctx, req)
	return nil
}

func (s *Service) ApproveAccessRequest(ctx context.Context, projectID, id uint, comment string, actor *auth.CurrentUser) error {
	req, err := s.repo.GetAccessRequestInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if req.Status != model.DbAccessRequestStatusPending {
		return constants.ErrBadRequestWithMsg("申请已结束")
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
	now := time.Now()
	uid := actorUserID(actor)
	cur.Status = model.DbApprovalStepApproved
	cur.ReviewerUserID = &uid
	cur.ReviewerName = actorUsername(actor)
	cur.ReviewComment = strings.TrimSpace(comment)
	cur.ReviewedAt = &now
	if err := s.repo.UpdateAccessRequestStep(ctx, cur); err != nil {
		return err
	}
	if err := s.advanceAccessRequestAfterApproval(ctx, req, cur); err != nil {
		return err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_approve", map[string]any{
		"request_id": req.ID, "database": req.DatabaseName, "comment": strings.TrimSpace(comment),
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
		ok, _ := s.userCanApproveStep(ctx, actor, cur.UserGroupID)
		if !ok {
			return constants.ErrForbidden
		}
		now := time.Now()
		uid := actorUserID(actor)
		cur.Status = model.DbApprovalStepRejected
		cur.ReviewerUserID = &uid
		cur.ReviewerName = actorUsername(actor)
		cur.ReviewComment = strings.TrimSpace(comment)
		cur.ReviewedAt = &now
		_ = s.repo.UpdateAccessRequestStep(ctx, cur)
	}
	req.Status = model.DbAccessRequestStatusRejected
	if err := s.repo.UpdateAccessRequest(ctx, req); err != nil {
		return err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "access_request_reject", map[string]any{
		"request_id": req.ID, "database": req.DatabaseName, "comment": strings.TrimSpace(comment),
	})
	return nil
}
