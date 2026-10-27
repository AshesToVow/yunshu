package dbmgmt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	workflowsvc "yunshu/internal/service/workflow"
)

type AppUserRequestItem struct {
	ID              uint     `json:"id"`
	ProjectID       uint     `json:"project_id"`
	InstanceID      uint     `json:"instance_id"`
	InstanceName    string   `json:"instance_name,omitempty"`
	RequesterUserID uint     `json:"requester_user_id"`
	RequesterName   string   `json:"requester_name"`
	ApplyType       string   `json:"apply_type"`
	MySQLUser       string   `json:"mysql_user"`
	MySQLHost       string   `json:"mysql_host"`
	DatabaseName    string   `json:"database_name"`
	PrivLevel       string   `json:"priv_level"`
	Privileges      []string `json:"privileges"`
	GrantHosts      string   `json:"grant_hosts"`
	Reason          string   `json:"reason"`
	Status          string   `json:"status"`
	ExecuteError    string   `json:"execute_error,omitempty"`
	CurrentStageName string  `json:"current_stage_name,omitempty"`
	MineStatus      string   `json:"mine_status,omitempty"`
	CreatedAt       string   `json:"created_at"`
}

type AppUserRequestCreateRequest struct {
	InstanceID   uint     `json:"instance_id" binding:"required"`
	ApplyType    string   `json:"apply_type" binding:"required"`
	MySQLUser    string   `json:"mysql_user" binding:"required"`
	MySQLHost    string   `json:"mysql_host"`
	DatabaseName string   `json:"database_name"`
	PrivLevel    string   `json:"priv_level"`
	Privileges   []string `json:"privileges" binding:"required"`
	GrantHosts   string   `json:"grant_hosts"`
	Reason       string   `json:"reason" binding:"required"`
}

type AppUserRequestListQuery struct {
	ProjectID       uint
	Status          string `form:"status"`
	RequesterUserID uint   `form:"requester_user_id"`
	Mine            bool   `form:"mine"`
	MineScope       string `form:"mine_scope"`
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	MineViewer      *auth.CurrentUser `form:"-"`
}

func (s *Service) toAppUserRequestItem(ctx context.Context, req model.DbAppUserRequest) AppUserRequestItem {
	item := AppUserRequestItem{
		ID: req.ID, ProjectID: req.ProjectID, InstanceID: req.InstanceID,
		RequesterUserID: req.RequesterUserID, RequesterName: req.RequesterName,
		ApplyType: req.ApplyType, MySQLUser: req.MySQLUser, MySQLHost: req.MySQLHost,
		DatabaseName: req.DatabaseName, PrivLevel: req.PrivLevel,
		Privileges: parsePrivilegesJSON(req.PrivilegesJSON),
		GrantHosts: req.GrantHosts, Reason: req.Reason, Status: req.Status,
		ExecuteError: req.ExecuteError,
		CreatedAt:    req.CreatedAt.Format(time.RFC3339),
	}
	if inst, err := s.repo.GetInstance(ctx, req.InstanceID); err == nil {
		item.InstanceName = inst.Name
	}
	return item
}

func (s *Service) CreateAppUserRequest(ctx context.Context, projectID uint, body AppUserRequestCreateRequest, actor *auth.CurrentUser) (*AppUserRequestItem, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, body.InstanceID)
	if err != nil {
		return nil, err
	}
	if inst.Driver != model.DbDriverMySQL {
		return nil, constants.ErrBadRequestWithMsg("应用用户权限申请仅支持 MySQL 实例")
	}
	applyType := strings.TrimSpace(body.ApplyType)
	switch applyType {
	case model.DbAppUserApplyNewUser, model.DbAppUserApplyAddPriv, model.DbAppUserApplyAddIP, model.DbAppUserApplyRevoke:
	default:
		return nil, constants.ErrBadRequestWithMsg("无效的申请类型")
	}
	user := strings.TrimSpace(body.MySQLUser)
	if user == "" {
		return nil, constants.ErrBadRequestWithMsg("请填写应用用户名")
	}
	privLevel := strings.TrimSpace(body.PrivLevel)
	if privLevel == "" {
		privLevel = model.DbAppUserPrivDatabase
	}
	if privLevel == model.DbAppUserPrivDatabase && strings.TrimSpace(body.DatabaseName) == "" && applyType != model.DbAppUserApplyRevoke {
		return nil, constants.ErrBadRequestWithMsg("库级权限须指定数据库")
	}
	privs := normalizeMySQLPrivs(body.Privileges, inst.Env == model.DbEnvProd)
	if applyType == model.DbAppUserApplyAddIP && len(privs) == 0 {
		privs = []string{"USAGE"}
	}
	if len(privs) == 0 && applyType != model.DbAppUserApplyRevoke && applyType != model.DbAppUserApplyAddIP {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一项 MySQL 权限")
	}
	host := strings.TrimSpace(body.MySQLHost)
	if host == "" {
		host = "%"
	}
	req := &model.DbAppUserRequest{
		ProjectID: projectID, InstanceID: body.InstanceID,
		RequesterUserID: actorUserID(actor), RequesterName: actorUsername(actor),
		ApplyType: applyType, MySQLUser: user, MySQLHost: host,
		DatabaseName: strings.TrimSpace(body.DatabaseName), PrivLevel: privLevel,
		PrivilegesJSON: marshalPrivilegesJSON(privs),
		GrantHosts: strings.TrimSpace(body.GrantHosts),
		Reason: strings.TrimSpace(body.Reason), Status: model.DbAccessRequestStatusPending,
	}
	if err := s.repo.CreateAppUserRequest(ctx, req); err != nil {
		return nil, err
	}
	if err := s.initAppUserRequestSteps(ctx, req); err != nil {
		req.Status = model.DbAccessRequestStatusRejected
		if uerr := s.repo.UpdateAppUserRequest(ctx, req); uerr != nil {
			slog.Default().With("component", "dbmgmt").Error("mark app user request rejected failed", "request_id", req.ID, "error", uerr)
		}
		return nil, err
	}
	iid := body.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_create", map[string]any{
		"request_id": req.ID, "apply_type": applyType, "mysql_user": user, "database": strings.TrimSpace(body.DatabaseName),
	})
	item := s.toAppUserRequestItem(ctx, *req)
	return &item, nil
}

func (s *Service) ListAppUserRequests(ctx context.Context, q AppUserRequestListQuery) (*pagination.Result[AppUserRequestItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if q.Mine && q.MineViewer != nil {
		scope := strings.TrimSpace(q.MineScope)
		if scope == "" {
			scope = "all"
		}
		list, total, err := s.repo.ListAppUserRequestsMine(ctx, repository.DbMineListParams{
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
		items := make([]AppUserRequestItem, 0, len(list))
		for _, req := range list {
			items = append(items, s.toAppUserRequestItem(ctx, req))
		}
		s.enrichAppUserRequestMineStatus(ctx, items, q.MineViewer)
		return paginate(items, total, page, pageSize), nil
	}
	list, total, err := s.repo.ListAppUserRequests(ctx, repository.DbAppUserRequestListParams{
		ProjectID: q.ProjectID, Status: q.Status, RequesterUserID: q.RequesterUserID,
		Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]AppUserRequestItem, 0, len(list))
	for _, req := range list {
		items = append(items, s.toAppUserRequestItem(ctx, req))
	}
	return paginate(items, total, page, pageSize), nil
}

func (s *Service) ApproveAppUserRequest(ctx context.Context, projectID, id uint, comment string, actor *auth.CurrentUser) error {
	req, err := s.repo.GetAppUserRequestInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if req.Status != model.DbAccessRequestStatusPending {
		return constants.ErrBadRequestWithMsg("申请已结束")
	}
	if s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbAppUserRequest, id) {
		err := s.reviewDbmgmtViaWorkflow(ctx, model.WorkflowRefDbAppUserRequest, id, true, comment, actor,
			func(d *workflowsvc.TicketDetail) error {
				if d.Status == model.WorkflowTicketStatusApproved {
					return s.finalizeAppUserRequestApproval(ctx, projectID, id)
				}
				return nil
			})
		if err != nil {
			return workflowOrLegacyErr(err)
		}
		iid := req.InstanceID
		_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_approve", map[string]any{
			"request_id": req.ID, "apply_type": req.ApplyType, "mysql_user": req.MySQLUser,
			"comment": strings.TrimSpace(comment),
		})
		return nil
	}
	steps, err := s.repo.ListAppUserRequestSteps(ctx, id)
	if err != nil {
		return err
	}
	var cur *model.DbAppUserRequestStep
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
	claimed, err := s.repo.ClaimAppUserRequestStep(ctx, cur.ID, req.ID, map[string]any{
		"status": model.DbApprovalStepApproved, "reviewer_user_id": uid,
		"reviewer_name": actorUsername(actor), "review_comment": strings.TrimSpace(comment),
		"reviewed_at": now, "updated_at": now,
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
	cur.ReviewComment = strings.TrimSpace(comment)
	cur.ReviewedAt = &now
	if err := s.advanceAppUserRequestAfterApproval(ctx, req, cur); err != nil {
		return err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_approve", map[string]any{
		"request_id": req.ID, "apply_type": req.ApplyType, "mysql_user": req.MySQLUser,
		"comment": strings.TrimSpace(comment),
	})
	return nil
}

func (s *Service) RejectAppUserRequest(ctx context.Context, projectID, id uint, comment string, actor *auth.CurrentUser) error {
	req, err := s.repo.GetAppUserRequestInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if req.Status != model.DbAccessRequestStatusPending {
		return constants.ErrBadRequestWithMsg("申请已结束")
	}
	if s.workflowEngine().HasLinkedTicket(ctx, model.WorkflowRefDbAppUserRequest, id) {
		err := s.reviewDbmgmtViaWorkflow(ctx, model.WorkflowRefDbAppUserRequest, id, false, comment, actor, nil)
		if err != nil {
			return workflowOrLegacyErr(err)
		}
		iid := req.InstanceID
		_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_reject", map[string]any{
			"request_id": req.ID, "apply_type": req.ApplyType, "mysql_user": req.MySQLUser,
			"comment": strings.TrimSpace(comment),
		})
		return nil
	}
	steps, err := s.repo.ListAppUserRequestSteps(ctx, id)
	if err != nil {
		return err
	}
	var cur *model.DbAppUserRequestStep
	for i := range steps {
		if steps[i].Status == model.DbApprovalStepPending {
			cur = &steps[i]
			break
		}
	}
	if cur != nil {
		ok, okErr := s.userCanApproveStep(ctx, actor, cur.UserGroupID)
		if okErr != nil {
			return okErr
		}
		if !ok {
			return constants.ErrForbidden
		}
		now := time.Now()
		uid := actorUserID(actor)
		claimed, claimErr := s.repo.ClaimAppUserRequestStep(ctx, cur.ID, req.ID, map[string]any{
			"status": model.DbApprovalStepRejected, "reviewer_user_id": uid,
			"reviewer_name": actorUsername(actor), "review_comment": strings.TrimSpace(comment),
			"reviewed_at": now, "updated_at": now,
		})
		if claimErr != nil {
			return claimErr
		}
		if claimed == 0 {
			return constants.ErrBadRequestWithMsg("该审批节点已被处理，请刷新后重试")
		}
	}
	n, err := s.repo.UpdateAppUserRequestFieldsIfStatus(ctx, req.ID, model.DbAccessRequestStatusPending, map[string]any{
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
	_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_reject", map[string]any{
		"request_id": req.ID, "apply_type": req.ApplyType, "mysql_user": req.MySQLUser,
		"comment": strings.TrimSpace(comment),
	})
	return nil
}

func (s *Service) initAppUserRequestSteps(ctx context.Context, req *model.DbAppUserRequest) error {
	if err := s.createDbmgmtWorkflowTicket(ctx, model.WorkflowTicketTypeAppUser, model.WorkflowRefDbAppUserRequest,
		appUserRequestWorkflowTitle(req), req.ProjectID, req.ID, req.RequesterUserID); err != nil {
		return err
	}
	req.Status = model.DbAccessRequestStatusPending
	return s.repo.UpdateAppUserRequest(ctx, req)
}

func (s *Service) advanceAppUserRequestAfterApproval(ctx context.Context, req *model.DbAppUserRequest, step *model.DbAppUserRequestStep) error {
	steps, err := s.repo.ListAppUserRequestSteps(ctx, req.ID)
	if err != nil {
		return err
	}
	var next *model.DbAppUserRequestStep
	for i := range steps {
		if steps[i].SortOrder > step.SortOrder && steps[i].Status == model.DbApprovalStepPending {
			next = &steps[i]
			break
		}
	}
	if next != nil {
		now := time.Now()
		next.ActivatedAt = &now
		return s.repo.UpdateAppUserRequestStep(ctx, next)
	}
	n, err := s.repo.UpdateAppUserRequestFieldsIfStatus(ctx, req.ID, model.DbAccessRequestStatusPending, map[string]any{
		"status": model.DbAccessRequestStatusApproved, "updated_at": time.Now(),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrBadRequestWithMsg("申请状态已变更，请刷新后重试")
	}
	req.Status = model.DbAccessRequestStatusApproved
	return s.executeAppUserRequest(ctx, req)
}

type appUserSQLStmt struct {
	SQL         string
	Host        string
	CreatesUser bool
}

// appUserExistence 记录执行前账号是否存在；Known=false 表示查询 mysql.user 失败，无法确认。
type appUserExistence struct {
	Known  bool
	Exists bool
}

func (s *Service) executeAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error {
	inst, err := s.repo.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}
	stmts, password, err := buildAppUserSQL(req)
	if err != nil {
		req.ExecuteError = err.Error()
		req.Status = model.DbAccessRequestStatusRejected
		if uerr := s.repo.UpdateAppUserRequestFields(ctx, req.ID, map[string]any{
			"status": req.Status, "execute_error": req.ExecuteError, "updated_at": time.Now(),
		}); uerr != nil {
			slog.Default().With("component", "dbmgmt").Warn("persist app user request failure failed", "request_id", req.ID, "error", uerr)
		}
		return err
	}
	release := s.acquireInstance(inst.ID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return err
	}
	defer sess.Close()

	// 预检账号存在性：只有确认执行前不存在、且回滚时 DROP 成功的账号才能认为已清理；
	// 预检不可用时无法判断 CREATE USER IF NOT EXISTS 是否真的新建了账号。
	existence := precheckAppUserExistence(ctx, sess.DB, req.MySQLUser, stmts)
	if req.ApplyType == model.DbAppUserApplyNewUser {
		for _, st := range stmts {
			if ex, ok := existence[st.Host]; ok && st.CreatesUser && ex.Known && ex.Exists {
				return s.failAppUserRequest(ctx, req, fmt.Sprintf(
					"用户 '%s'@'%s' 已存在，请改用「新增权限」或「新增授权IP」重新申请", req.MySQLUser, st.Host))
			}
		}
	}

	executed := make([]appUserSQLStmt, 0, len(stmts))
	for i, st := range stmts {
		if _, execErr := sess.DB.ExecContext(ctx, st.SQL); execErr != nil {
			return s.handleAppUserExecFailure(ctx, sess.DB, req, executed, st, i+1, password, existence, inst, execErr)
		}
		executed = append(executed, st)
	}
	return s.finishAppUserRequest(ctx, req, password, executed, existence)
}

func precheckAppUserExistence(ctx context.Context, db *sql.DB, user string, stmts []appUserSQLStmt) map[string]appUserExistence {
	out := make(map[string]appUserExistence)
	for _, st := range stmts {
		if !st.CreatesUser {
			continue
		}
		if _, done := out[st.Host]; done {
			continue
		}
		var n int
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mysql.user WHERE user = ? AND host = ?", user, st.Host).Scan(&n)
		if err != nil {
			out[st.Host] = appUserExistence{}
			continue
		}
		out[st.Host] = appUserExistence{Known: true, Exists: n > 0}
	}
	return out
}

func (s *Service) failAppUserRequest(ctx context.Context, req *model.DbAppUserRequest, msg string) error {
	req.Status = model.DbAccessRequestStatusRejected
	req.ExecuteError = msg
	if err := s.repo.UpdateAppUserRequestFields(ctx, req.ID, map[string]any{
		"status": req.Status, "execute_error": msg, "updated_at": time.Now(),
	}); err != nil {
		slog.Default().With("component", "dbmgmt").Error("persist app user request failure failed", "request_id", req.ID, "error", err)
	}
	return constants.ErrBadRequestWithMsg(msg)
}

func (s *Service) handleAppUserExecFailure(ctx context.Context, db *sql.DB, req *model.DbAppUserRequest,
	executed []appUserSQLStmt, failed appUserSQLStmt, stmtIndex int, password string,
	existence map[string]appUserExistence, inst *model.DbInstance, execErr error) error {
	msg := formatMySQLGrantExecError(execErr, inst, req, stmtIndex, failed.SQL)

	// DDL 无法事务回滚：逐 host 撤销失败前已成功的 CREATE USER，仅限确认原本不存在的账号。
	var keptHosts []string
	for _, st := range executed {
		if !st.CreatesUser {
			continue
		}
		ex := existence[st.Host]
		switch {
		case ex.Known && ex.Exists:
			// 账号原本就存在，CREATE USER IF NOT EXISTS 未改密码，无需处理。
		case ex.Known && !ex.Exists:
			drop := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'",
				escapeMySQLString(req.MySQLUser), escapeMySQLString(st.Host))
			if _, dropErr := db.ExecContext(ctx, drop); dropErr != nil {
				keptHosts = append(keptHosts, st.Host)
				msg += fmt.Sprintf("；回滚账号 '%s'@'%s' 失败：%v", req.MySQLUser, st.Host, dropErr)
			}
		default:
			// 预检不可用：账号可能已被创建，保留凭据供人工核实。
			keptHosts = append(keptHosts, st.Host)
		}
	}
	if failed.CreatesUser && isExecutionResultUnknown(execErr) {
		ex := existence[failed.Host]
		if !ex.Known || !ex.Exists {
			keptHosts = append(keptHosts, failed.Host)
			msg += fmt.Sprintf("；账号 '%s'@'%s' 的创建结果未知，可能已生效", req.MySQLUser, failed.Host)
		}
	}
	if len(keptHosts) > 0 && password != "" {
		remark := fmt.Sprintf("申请 #%d 执行部分失败，账号可能已创建，请核实", req.ID)
		if saveErr := s.saveAppUserAccounts(ctx, req, password, keptHosts, executed, remark); saveErr != nil {
			msg += "；保存已创建账号凭据失败：" + saveErr.Error()
		} else {
			msg += "；已保留账号凭据，请到实例账号列表核实"
		}
	}
	var applied []string
	for _, st := range executed {
		if !st.CreatesUser {
			applied = append(applied, st.SQL)
		}
	}
	if len(applied) > 0 {
		msg += "；已执行的授权语句：" + truncateSQL(strings.Join(applied, "; "), 1500)
	}
	return s.failAppUserRequest(ctx, req, msg)
}

func (s *Service) finishAppUserRequest(ctx context.Context, req *model.DbAppUserRequest, password string,
	executed []appUserSQLStmt, existence map[string]appUserExistence) error {
	var createdHosts []string
	for _, st := range executed {
		if !st.CreatesUser {
			continue
		}
		if ex := existence[st.Host]; ex.Known && ex.Exists {
			continue
		}
		createdHosts = append(createdHosts, st.Host)
	}
	var warning string
	if password != "" && len(createdHosts) > 0 {
		if err := s.saveAppUserAccounts(ctx, req, password, createdHosts, executed,
			fmt.Sprintf("来自应用用户申请 #%d", req.ID)); err != nil {
			// 授权已生效，不能把申请判失败；把凭据问题暴露给审批人，由管理员重置密码。
			slog.Default().With("component", "dbmgmt").Error("save app user account credentials failed",
				"request_id", req.ID, "error", err)
			warning = "授权已生效，但账号凭据保存失败，请管理员重置密码：" + err.Error()
		}
	}
	if err := s.repo.UpdateAppUserRequestFields(ctx, req.ID, map[string]any{
		"status": model.DbAccessRequestStatusApproved, "execute_error": warning, "updated_at": time.Now(),
	}); err != nil {
		return err
	}
	req.Status = model.DbAccessRequestStatusApproved
	req.ExecuteError = warning
	return nil
}

func (s *Service) saveAppUserAccounts(ctx context.Context, req *model.DbAppUserRequest, password string,
	hosts []string, executed []appUserSQLStmt, remark string) error {
	enc, err := cryptox.EncryptString(s.aead, password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}
	lines := make([]string, 0, len(executed))
	for _, st := range executed {
		lines = append(lines, st.SQL)
	}
	grantsSummary := strings.Join(lines, "\n")
	rid := req.ID
	seen := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		if err := s.repo.CreateInstanceAccount(ctx, &model.DbInstanceAccount{
			ProjectID: req.ProjectID, InstanceID: req.InstanceID, AppUserRequestID: &rid,
			Username: req.MySQLUser, Host: h, EncPassword: enc,
			GrantsSummary: grantsSummary, Remark: remark,
		}); err != nil {
			return err
		}
	}
	return nil
}

func buildAppUserSQL(req *model.DbAppUserRequest) (stmts []appUserSQLStmt, password string, err error) {
	privs := parsePrivilegesJSON(req.PrivilegesJSON)
	if len(privs) == 0 && req.ApplyType != model.DbAppUserApplyRevoke {
		return nil, "", fmt.Errorf("权限列表为空")
	}
	hosts := splitGrantHosts(req.GrantHosts, req.MySQLHost)
	var parts []appUserSQLStmt
	appendGrants := func(h string) {
		for _, s := range buildGrantStmtsForHost(req, h, privs) {
			parts = append(parts, appUserSQLStmt{SQL: s, Host: h})
		}
	}
	appendCreate := func(h string) {
		parts = append(parts, appUserSQLStmt{
			SQL: fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'",
				escapeMySQLString(req.MySQLUser), escapeMySQLString(h), escapeMySQLString(password)),
			Host: h, CreatesUser: true,
		})
	}
	switch req.ApplyType {
	case model.DbAppUserApplyNewUser:
		password, err = randomMySQLPassword(16)
		if err != nil {
			return nil, "", err
		}
		for _, h := range hosts {
			appendCreate(h)
			appendGrants(h)
		}
	case model.DbAppUserApplyAddPriv, model.DbAppUserApplyAddIP:
		for _, h := range hosts {
			if needsCreateUserForHost(req, h) {
				if password == "" {
					password, err = randomMySQLPassword(16)
					if err != nil {
						return nil, "", err
					}
				}
				appendCreate(h)
			}
			appendGrants(h)
		}
	case model.DbAppUserApplyRevoke:
		for _, h := range hosts {
			for _, s := range buildRevokeStmtsForHost(req, h, privs) {
				parts = append(parts, appUserSQLStmt{SQL: s, Host: h})
			}
		}
	default:
		return nil, "", fmt.Errorf("unsupported apply type")
	}
	if len(parts) == 0 {
		return nil, "", fmt.Errorf("未生成有效的授权语句，请检查权限与级别是否匹配")
	}
	return parts, password, nil
}

func grantObject(req *model.DbAppUserRequest) string {
	if req.PrivLevel == model.DbAppUserPrivGlobal {
		return "*.*"
	}
	db := strings.TrimSpace(req.DatabaseName)
	if db == "" {
		return "*.*"
	}
	return fmt.Sprintf("`%s`.*", escapeMySQLIdent(db))
}

func needsCreateUserForHost(req *model.DbAppUserRequest, host string) bool {
	switch req.ApplyType {
	case model.DbAppUserApplyAddIP:
		return true
	case model.DbAppUserApplyAddPriv:
		return host != strings.TrimSpace(req.MySQLHost)
	default:
		return false
	}
}

func splitGrantHosts(raw, fallback string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if fallback == "" {
			return []string{"%"}
		}
		return []string{fallback}
	}
	sep := ";"
	if strings.Contains(raw, "|") && !strings.Contains(raw, ";") {
		sep = "|"
	}
	var out []string
	for _, p := range strings.Split(raw, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"%"}
	}
	return out
}

func normalizeMySQLPrivs(in []string, prodEnv bool) []string {
	allowed := map[string]struct{}{
		"SELECT": {}, "INSERT": {}, "UPDATE": {}, "DELETE": {},
		"CREATE": {}, "ALTER": {}, "INDEX": {}, "DROP": {},
		"CREATE TEMPORARY TABLES": {}, "LOCK TABLES": {}, "EXECUTE": {},
		"CREATE VIEW": {}, "SHOW VIEW": {}, "CREATE ROUTINE": {}, "ALTER ROUTINE": {},
		"EVENT": {}, "TRIGGER": {}, "REFERENCES": {},
		"GRANT": {}, "SUPER": {}, "PROCESS": {}, "RELOAD": {}, "SHUTDOWN": {},
		"SHOW DATABASES": {}, "REPLICATION CLIENT": {}, "REPLICATION SLAVE": {}, "CREATE USER": {},
		"USAGE": {},
	}
	dangerous := map[string]struct{}{
		"GRANT": {}, "SUPER": {}, "PROCESS": {}, "RELOAD": {}, "SHUTDOWN": {},
		"CREATE USER": {}, "REPLICATION SLAVE": {}, "REPLICATION CLIENT": {},
	}
	var out []string
	seen := map[string]struct{}{}
	for _, p := range in {
		up := strings.ToUpper(strings.TrimSpace(p))
		if up == "" {
			continue
		}
		if _, ok := allowed[up]; !ok {
			continue
		}
		if prodEnv {
			if _, blocked := dangerous[up]; blocked {
				continue
			}
		}
		if _, ok := seen[up]; ok {
			continue
		}
		seen[up] = struct{}{}
		out = append(out, up)
	}
	return out
}

func randomMySQLPassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		return s[:n], nil
	}
	return s, nil
}

func (s *Service) RevealInstanceAccountPassword(ctx context.Context, projectID, accountID uint, actor *auth.CurrentUser) (string, error) {
	if actor == nil {
		return "", constants.ErrForbidden
	}
	acc, err := s.repo.GetInstanceAccount(ctx, projectID, accountID)
	if err != nil {
		return "", err
	}
	if !isSuperAdmin(actor) {
		perm, err := s.GetEffectivePermission(ctx, projectID, acc.InstanceID, actor)
		if err != nil || perm == nil || !perm.CanManage {
			return "", constants.ErrForbidden
		}
	}
	if acc.EncPassword == "" {
		return "", constants.ErrBadRequestWithMsg("该账号无平台托管密码")
	}
	pw, err := cryptox.DecryptString(s.aead, acc.EncPassword)
	if err != nil {
		return "", constants.ErrBadRequestWithMsg(constants.ErrMsgDbInstancePasswordDecryptFailed)
	}
	return pw, nil
}
