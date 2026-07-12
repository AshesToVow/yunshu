package dbmgmt

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
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
	privs := normalizeMySQLPrivs(body.Privileges)
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
		dbq := s.db.WithContext(ctx).Model(&model.DbAppUserRequest{}).Where("project_id = ?", q.ProjectID)
		if st := strings.TrimSpace(q.Status); st != "" {
			dbq = dbq.Where("status = ?", st)
		}
		scope := strings.TrimSpace(q.MineScope)
		if scope == "" {
			scope = "all"
		}
		switch scope {
		case "pending":
			dbq = s.filterAppUserRequestsApprovalPending(dbq, q.MineViewer)
		case "done":
			dbq = s.filterAppUserRequestsApprovalDone(dbq, actorUserID(q.MineViewer))
		default:
			dbq = s.filterAppUserRequestsApprovalMine(dbq, q.MineViewer)
		}
		var total int64
		if err := dbq.Count(&total).Error; err != nil {
			return nil, err
		}
		var list []model.DbAppUserRequest
		if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
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
	now := time.Now()
	uid := actorUserID(actor)
	cur.Status = model.DbApprovalStepApproved
	cur.ReviewerUserID = &uid
	cur.ReviewerName = actorUsername(actor)
	cur.ReviewComment = strings.TrimSpace(comment)
	cur.ReviewedAt = &now
	if err := s.repo.UpdateAppUserRequestStep(ctx, cur); err != nil {
		return err
	}
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
	steps, _ := s.repo.ListAppUserRequestSteps(ctx, id)
	var cur *model.DbAppUserRequestStep
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
		_ = s.repo.UpdateAppUserRequestStep(ctx, cur)
	}
	req.Status = model.DbAccessRequestStatusRejected
	if err := s.repo.UpdateAppUserRequest(ctx, req); err != nil {
		return err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "app_user_request_reject", map[string]any{
		"request_id": req.ID, "apply_type": req.ApplyType, "mysql_user": req.MySQLUser,
		"comment": strings.TrimSpace(comment),
	})
	return nil
}

func (s *Service) initAppUserRequestSteps(ctx context.Context, req *model.DbAppUserRequest) error {
	stages, err := s.loadEnabledFlowStages(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if len(stages) == 0 {
		req.Status = model.DbAccessRequestStatusApproved
		if err := s.repo.UpdateAppUserRequest(ctx, req); err != nil {
			return err
		}
		return s.executeAppUserRequest(ctx, req)
	}
	now := time.Now()
	for i, st := range stages {
		step := &model.DbAppUserRequestStep{
			AppUserRequestID: req.ID, StageKey: st.StageKey, StageName: st.StageName,
			SortOrder: st.SortOrder, Status: model.DbApprovalStepPending, UserGroupID: st.UserGroupID,
		}
		if i == 0 {
			step.ActivatedAt = &now
		}
		if err := s.repo.CreateAppUserRequestStep(ctx, step); err != nil {
			return err
		}
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
	req.Status = model.DbAccessRequestStatusApproved
	if err := s.repo.UpdateAppUserRequest(ctx, req); err != nil {
		return err
	}
	return s.executeAppUserRequest(ctx, req)
}

func (s *Service) executeAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error {
	inst, err := s.repo.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}
	sqlStmts, password, err := buildAppUserSQL(req)
	if err != nil {
		req.ExecuteError = err.Error()
		req.Status = model.DbTicketStatusFailed
		_ = s.repo.UpdateAppUserRequest(ctx, req)
		return err
	}
	release := s.acquireInstance(inst.ID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return err
	}
	defer sess.Close()
	for i, sqlText := range sqlStmts {
		if _, err := sess.DB.ExecContext(ctx, sqlText); err != nil {
			req.ExecuteError = err.Error()
			req.Status = model.DbTicketStatusFailed
			_ = s.repo.UpdateAppUserRequest(ctx, req)
			return constants.ErrBadRequestWithMsg(formatMySQLGrantExecError(err, inst, req, i+1, sqlText))
		}
	}
	if password != "" {
		enc, encErr := cryptox.EncryptString(s.aead, password)
		if encErr == nil {
			hosts := splitGrantHosts(req.GrantHosts, req.MySQLHost)
			rid := req.ID
			for _, h := range hosts {
				if req.ApplyType != model.DbAppUserApplyNewUser &&
					req.ApplyType != model.DbAppUserApplyAddIP &&
					!needsCreateUserForHost(req, h) {
					continue
				}
				_ = s.repo.CreateInstanceAccount(ctx, &model.DbInstanceAccount{
					ProjectID: req.ProjectID, InstanceID: req.InstanceID, AppUserRequestID: &rid,
					Username: req.MySQLUser, Host: h, EncPassword: enc,
					GrantsSummary: strings.Join(sqlStmts, "\n"), Remark: fmt.Sprintf("来自应用用户申请 #%d", req.ID),
				})
			}
		}
	}
	req.Status = model.DbTicketStatusSuccess
	req.ExecuteError = ""
	return s.repo.UpdateAppUserRequest(ctx, req)
}

func buildAppUserSQL(req *model.DbAppUserRequest) (stmts []string, password string, err error) {
	privs := parsePrivilegesJSON(req.PrivilegesJSON)
	if len(privs) == 0 && req.ApplyType != model.DbAppUserApplyRevoke {
		return nil, "", fmt.Errorf("权限列表为空")
	}
	hosts := splitGrantHosts(req.GrantHosts, req.MySQLHost)
	var parts []string
	switch req.ApplyType {
	case model.DbAppUserApplyNewUser:
		password = randomMySQLPassword(16)
		for _, h := range hosts {
			parts = append(parts, fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'",
				escapeMySQLString(req.MySQLUser), escapeMySQLString(h), escapeMySQLString(password)))
			parts = append(parts, buildGrantStmtsForHost(req, h, privs)...)
		}
	case model.DbAppUserApplyAddPriv, model.DbAppUserApplyAddIP:
		for _, h := range hosts {
			if needsCreateUserForHost(req, h) {
				if password == "" {
					password = randomMySQLPassword(16)
				}
				parts = append(parts, fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'",
					escapeMySQLString(req.MySQLUser), escapeMySQLString(h), escapeMySQLString(password)))
			}
			parts = append(parts, buildGrantStmtsForHost(req, h, privs)...)
		}
	case model.DbAppUserApplyRevoke:
		for _, h := range hosts {
			parts = append(parts, buildRevokeStmtsForHost(req, h, privs)...)
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

func normalizeMySQLPrivs(in []string) []string {
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
		if _, ok := seen[up]; ok {
			continue
		}
		seen[up] = struct{}{}
		out = append(out, up)
	}
	return out
}

func randomMySQLPassword(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		return s[:n]
	}
	return s
}

func (s *Service) RevealInstanceAccountPassword(ctx context.Context, projectID, accountID uint, actor *auth.CurrentUser) (string, error) {
	acc, err := s.repo.GetInstanceAccount(ctx, projectID, accountID)
	if err != nil {
		return "", err
	}
	if !auth.IsSuperAdminRole(actor.RoleCodes) {
		perm, err := s.GetEffectivePermission(ctx, projectID, acc.InstanceID, actor)
		if err != nil || !perm.CanManage {
			return "", constants.ErrForbidden
		}
	}
	if acc.EncPassword == "" {
		return "", constants.ErrBadRequestWithMsg("该账号无平台托管密码")
	}
	return cryptox.DecryptString(s.aead, acc.EncPassword)
}
