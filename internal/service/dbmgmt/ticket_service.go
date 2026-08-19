package dbmgmt

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/goinception"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	"yunshu/internal/service/changeevent"
)

type ExecuteRequest struct {
	Database   string `json:"database"`
	Sql        string `json:"sql" binding:"required"`
	Reason     string `json:"reason"`
	AuditMode  string `json:"audit_mode"`
	IsBackup   *bool  `json:"is_backup"`
	SqlFileRef string `json:"sql_file_ref"`
}

type ExecuteResponse struct {
	Status    string `json:"status"`
	TicketID  uint   `json:"ticket_id,omitempty"`
	RiskLevel string `json:"risk_level,omitempty"`
	Message   string `json:"message,omitempty"`
	RowsAffected int64 `json:"rows_affected,omitempty"`
}

type ImportRequest struct {
	Database   string `json:"database"`
	Sql        string `json:"sql" binding:"required"`
	Reason     string `json:"reason"`
	AuditMode  string `json:"audit_mode"`
	IsBackup   *bool  `json:"is_backup"`
	SqlFileRef string `json:"sql_file_ref"`
}

type TicketItem struct {
	ID               uint   `json:"id"`
	ProjectID        uint   `json:"project_id"`
	InstanceID       uint   `json:"instance_id"`
	InstanceName     string `json:"instance_name,omitempty"`
	TicketType       string `json:"ticket_type"`
	SubmitterUserID  uint   `json:"submitter_user_id,omitempty"`
	SubmitterName    string `json:"submitter_name"`
	DatabaseName     string `json:"database_name"`
	RiskLevel        string `json:"risk_level"`
	SyntaxType       int    `json:"syntax_type,omitempty"`
	IsBackup         bool   `json:"is_backup,omitempty"`
	Status           string `json:"status"`
	CurrentStageName string `json:"current_stage_name,omitempty"`
	MineStatus       string `json:"mine_status,omitempty"`
	SqlExcerpt       string `json:"sql_excerpt,omitempty"`
	SqlText          string `json:"sql_text,omitempty"`
	ReviewJSON       string `json:"review_json,omitempty"`
	ExecuteJSON      string `json:"execute_json,omitempty"`
	Reason           string `json:"reason,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	SqlFileRef       string `json:"sql_file_ref,omitempty"`
	AuditMode        string `json:"audit_mode,omitempty"`
}

type TicketListQuery struct {
	ProjectID  uint
	Status     string `form:"status"`
	TicketType string `form:"ticket_type"`
	Mine       bool   `form:"mine"`
	MineScope  string `form:"mine_scope"`
	MineTab    string `form:"-"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	MineViewer *auth.CurrentUser `form:"-"`
}

type ReviewRequest struct {
	Comment string `json:"comment"`
}

func (s *Service) toTicketItem(ctx context.Context, t model.DbSqlTicket) TicketItem {
	item := TicketItem{
		ID: t.ID, ProjectID: t.ProjectID, InstanceID: t.InstanceID,
		TicketType: t.TicketType, SubmitterUserID: t.SubmitterUserID, SubmitterName: t.SubmitterName,
		DatabaseName: t.DatabaseName, RiskLevel: t.RiskLevel, SyntaxType: t.SyntaxType, IsBackup: t.IsBackup,
		Status: t.Status, ReviewJSON: t.ReviewJSON, ExecuteJSON: t.ExecuteJSON,
		SqlExcerpt: truncateSQL(t.SqlText, 200), Reason: t.Reason,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
		SqlFileRef: t.SqlFileRef,
		AuditMode: normalizeAuditMode(t.AuditMode),
	}
	if inst, err := s.repo.GetInstance(ctx, t.InstanceID); err == nil {
		item.InstanceName = inst.Name
	}
	return item
}

func (s *Service) GetTicket(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) (*TicketItem, error) {
	t, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return nil, err
	}
	item := s.toTicketItem(ctx, *t)
	if s.canViewTicketSQL(ctx, projectID, t, actor) {
		item.SqlText = t.SqlText
	}
	item.ReviewJSON = sanitizeReviewJSON(t.ReviewJSON)
	item.ExecuteJSON = sanitizeReviewJSON(t.ExecuteJSON)
	return &item, nil
}

func (s *Service) preCheckForTicket(ctx context.Context, inst *model.DbInstance, dbName, sqlText string) (reviewJSON string, syntaxType int, err error) {
	if !s.goInceptionAvailable(ctx,inst) {
		if reDDL.MatchString(strings.ToUpper(sqlText)) {
			return "", goinception.SyntaxDDL, nil
		}
		return "", goinception.SyntaxDML, nil
	}
	rs, checkErr := s.runGoInceptionCheck(ctx, inst, dbName, sqlText)
	if checkErr != nil {
		return "", 0, constants.ErrBadRequestWithMsg("SQL 预检失败: " + checkErr.Error())
	}
	reviewJSON = marshalReviewSet(rs)
	if rs != nil && rs.ErrorCount > 0 {
		msg := firstReviewErrorMessage(rs, fmt.Sprintf("SQL 预检发现 %d 个错误，请修正后重试", rs.ErrorCount))
		return reviewJSON, rs.SyntaxType, constants.ErrBadRequestWithMsg(msg)
	}
	if rs != nil {
		return reviewJSON, rs.SyntaxType, nil
	}
	return "", goinception.SyntaxDML, nil
}

func (s *Service) executeWriteViaEngine(ctx context.Context, inst *model.DbInstance, database, sqlText string, backup bool, actor *auth.CurrentUser, ticketID *uint) (executeJSON string, err error) {
	if s.goInceptionAvailable(ctx,inst) {
		rs, execErr := s.runGoInceptionExecute(ctx, inst, database, sqlText, backup)
		executeJSON = marshalReviewSet(rs)
		if execErr != nil {
			return executeJSON, execErr
		}
		if rs == nil {
			return executeJSON, constants.ErrBadRequestWithMsg("goInception 未返回执行结果")
		}
		if rs.Error != "" {
			return executeJSON, constants.ErrBadRequestWithMsg(rs.Error)
		}
		for _, row := range rs.Rows {
			if row.ErrorLevel >= goinception.ErrLevelError && !strings.Contains(row.StageStatus, "Execute Successfully") {
				return executeJSON, constants.ErrBadRequestWithMsg(fmt.Sprintf("第 %d 行: %s", row.OrderID, row.ErrorMessage))
			}
		}
		return executeJSON, nil
	}
	_, err = s.runWriteSQL(ctx, inst, database, sqlText, actor, ticketID)
	return "", err
}

func (s *Service) needsApproval(inst *model.DbInstance, assessment SQLAssessment, cfg configResolved) bool {
	if assessment.Blocked {
		return false
	}
	// 实例开启「写操作须工单」时，任意变更 SQL 都走审批（含原判定为 low 的边界语句）。
	if inst.RequireTicketForDML {
		return true
	}
	// 生产环境：开启 ProdForceApproval 时一律走工单（含 low 风险），禁止直执。
	if inst.Env == model.DbEnvProd && cfg.ProdForceApproval {
		return true
	}
	return assessment.RiskLevel == model.DbRiskHigh || assessment.RiskLevel == model.DbRiskMedium
}

func normalizeAuditMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), model.DbAuditModeManual) {
		return model.DbAuditModeManual
	}
	return model.DbAuditModeSystem
}

func resolveBackupChoice(choice *bool, defaultBackup bool) bool {
	if choice != nil {
		return *choice
	}
	return defaultBackup
}

func (s *Service) requiresTicketApproval(inst *model.DbInstance, assessment SQLAssessment, cfg configResolved, auditMode string) bool {
	if assessment.Blocked {
		return false
	}
	if normalizeAuditMode(auditMode) == model.DbAuditModeManual {
		return true
	}
	return s.needsApproval(inst, assessment, cfg)
}

type configResolved struct {
	ProdForceApproval bool
}

func (s *Service) ExecuteSQL(ctx context.Context, projectID, instanceID uint, req ExecuteRequest, actor *auth.CurrentUser) (*ExecuteResponse, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	sqlText := strings.TrimSpace(req.Sql)
	instanceDDL := isInstanceLevelDDL(sqlText)
	if !instanceDDL {
		if _, err := resolveQueryDatabase(inst, req.Database); err != nil {
			return nil, err
		}
	}
	if isReadOnlySQL(sqlText) {
		return nil, constants.ErrBadRequestWithMsg("只读语句（SELECT/SHOW 等）请使用「SQL 查询」页面；SQL 审核仅支持 INSERT/UPDATE/DELETE/DDL 等变更语句")
	}
	assess := AssessSQLForWrite(sqlText, inst.Env == model.DbEnvProd, s.goInceptionAvailable(ctx, inst))
	if assess.Blocked {
		return nil, constants.ErrBadRequestWithMsg(assess.Reason)
	}
	needDDL := strings.Contains(strings.ToUpper(sqlText), "CREATE") ||
		strings.Contains(strings.ToUpper(sqlText), "ALTER") ||
		strings.Contains(strings.ToUpper(sqlText), "DROP") ||
		strings.Contains(strings.ToUpper(sqlText), "TRUNCATE")
	if err := s.checkWritePermission(ctx, projectID, inst, req.Database, sqlText, needDDL, actor); err != nil {
		return nil, err
	}
	cfg := s.resolvedConfig(ctx)
	cfgResolved := configResolved{ProdForceApproval: cfg.ProdForceApproval}
	auditMode := normalizeAuditMode(req.AuditMode)
	backup := resolveBackupChoice(req.IsBackup, cfg.GoInceptionBackup)
	if s.requiresTicketApproval(inst, assess, cfgResolved, auditMode) {
		reviewJSON, syntaxType, checkErr := s.preCheckForTicket(ctx, inst, req.Database, sqlText)
		if checkErr != nil {
			return nil, checkErr
		}
		ops, _ := json.Marshal(assess.Ops)
		ticket := &model.DbSqlTicket{
			ProjectID: projectID, InstanceID: instanceID, TicketType: model.DbTicketTypeSqlExecute,
			SubmitterUserID: actorUserID(actor), SubmitterName: actorUsername(actor),
			DatabaseName: req.Database, SqlText: sqlText, RiskLevel: assess.RiskLevel,
			SyntaxType: syntaxType, IsBackup: backup, AuditMode: auditMode,
			SqlFileRef: strings.TrimSpace(req.SqlFileRef),
			ParsedOpsJSON: string(ops), ReviewJSON: reviewJSON, Reason: strings.TrimSpace(req.Reason),
			Status: model.DbTicketStatusDraft,
		}
		if err := s.repo.CreateSqlTicket(ctx, ticket); err != nil {
			return nil, err
		}
		if err := s.initSqlTicketSteps(ctx, ticket); err != nil {
			return nil, err
		}
		s.auditTicketEvent(ctx, projectID, instanceID, actor, "ticket_create", ticket, nil)
		return &ExecuteResponse{
			Status: ticket.Status, TicketID: ticket.ID, RiskLevel: assess.RiskLevel,
			Message: "已创建审批工单",
		}, nil
	}
	var execErr error
	if s.goInceptionAvailable(ctx,inst) {
		_, execErr = s.executeWriteViaEngine(ctx, inst, req.Database, sqlText, backup, actor, nil)
	} else {
		_, execErr = s.runWriteSQL(ctx, inst, req.Database, sqlText, actor, nil)
	}
	if execErr != nil {
		return nil, execErr
	}
	s.auditSQLWrite(ctx, inst, req.Database, sqlText, "sql_execute", actor, nil)
	return &ExecuteResponse{Status: "success", RiskLevel: assess.RiskLevel}, nil
}

// createSqlExecuteTicket 创建 SQL 执行工单；forceTicket=true 时无视风险等级强制走审批流（用于回滚等敏感操作）。
func (s *Service) createSqlExecuteTicket(ctx context.Context, projectID uint, inst *model.DbInstance, database, sqlText, reason string, actor *auth.CurrentUser, forceTicket bool) (*ExecuteResponse, error) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, constants.ErrBadRequestWithMsg("SQL 不能为空")
	}
	instanceDDL := isInstanceLevelDDL(sqlText)
	if !instanceDDL {
		if _, err := resolveQueryDatabase(inst, database); err != nil {
			return nil, err
		}
	}
	assess := AssessSQL(sqlText, inst.Env == model.DbEnvProd)
	if assess.Blocked && s.goInceptionAvailable(ctx, inst) && strings.Contains(assess.Reason, "多语句") {
		assess = SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: detectOps(sqlText)}
	}
	if assess.Blocked {
		return nil, constants.ErrBadRequestWithMsg(assess.Reason)
	}
	needDDL := reDDL.MatchString(strings.ToUpper(sqlText))
	if err := s.checkWritePermission(ctx, projectID, inst, database, sqlText, needDDL, actor); err != nil {
		return nil, err
	}
	cfg := s.resolvedConfig(ctx)
	if !forceTicket && !s.needsApproval(inst, assess, configResolved{ProdForceApproval: cfg.ProdForceApproval}) {
		return nil, constants.ErrBadRequestWithMsg("当前 SQL 无需创建工单")
	}
	reviewJSON, syntaxType, checkErr := s.preCheckForTicket(ctx, inst, database, sqlText)
	if checkErr != nil {
		return nil, checkErr
	}
	ops, _ := json.Marshal(assess.Ops)
	ticket := &model.DbSqlTicket{
		ProjectID: projectID, InstanceID: inst.ID, TicketType: model.DbTicketTypeSqlExecute,
		SubmitterUserID: actorUserID(actor), SubmitterName: actorUsername(actor),
		DatabaseName: database, SqlText: sqlText, RiskLevel: assess.RiskLevel,
		SyntaxType: syntaxType, IsBackup: cfg.GoInceptionBackup,
		ParsedOpsJSON: string(ops), ReviewJSON: reviewJSON, Reason: strings.TrimSpace(reason),
		Status: model.DbTicketStatusDraft,
	}
	if err := s.repo.CreateSqlTicket(ctx, ticket); err != nil {
		return nil, err
	}
	if err := s.initSqlTicketSteps(ctx, ticket); err != nil {
		return nil, err
	}
	s.auditTicketEvent(ctx, projectID, inst.ID, actor, "ticket_create", ticket, map[string]any{"reason": "rollback"})
	status := model.DbTicketStatusPendingApproval
	if ticket.Status == model.DbTicketStatusPendingExecution {
		status = model.DbTicketStatusPendingExecution
	}
	return &ExecuteResponse{
		Status: status, TicketID: ticket.ID, RiskLevel: assess.RiskLevel,
		Message: fmt.Sprintf("已创建回滚工单 #%d", ticket.ID),
	}, nil
}

func (s *Service) ImportSQL(ctx context.Context, projectID, instanceID uint, req ImportRequest, actor *auth.CurrentUser) (*ExecuteResponse, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	stmts := splitSQLStatements(req.Sql)
	if len(stmts) == 0 {
		return nil, constants.ErrBadRequestWithMsg("SQL 文件为空")
	}
	for _, st := range stmts {
		needDDL := reDDL.MatchString(st)
		if err := s.checkWritePermission(ctx, projectID, inst, req.Database, st, needDDL, actor); err != nil {
			return nil, err
		}
	}
	cfg := s.resolvedConfig(ctx)
	perm, perr := s.effectivePermissionForDatabase(ctx, projectID, inst.ID, req.Database, actor)
	if perr != nil {
		return nil, perr
	}
	if !perm.CanManage && !perm.CanImport {
		return nil, constants.ErrForbiddenWithMsg("你无该库的 SQL 导入权限")
	}
	maxBytes := cfg.MaxImportFileMB * 1024 * 1024
	if maxBytes > 0 && len(req.Sql) > maxBytes {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("SQL 文件超过 %d MB 限制", cfg.MaxImportFileMB))
	}
	maxRisk := model.DbRiskLow
	for _, st := range stmts {
		a := AssessSQL(st, inst.Env == model.DbEnvProd)
		if a.Blocked {
			return nil, constants.ErrBadRequestWithMsg(a.Reason)
		}
		if riskRank(a.RiskLevel) > riskRank(maxRisk) {
			maxRisk = a.RiskLevel
		}
	}
	assess := SQLAssessment{RiskLevel: maxRisk}
	cfgResolved := configResolved{ProdForceApproval: cfg.ProdForceApproval}
	auditMode := normalizeAuditMode(req.AuditMode)
	backup := resolveBackupChoice(req.IsBackup, cfg.GoInceptionBackup)
	if s.requiresTicketApproval(inst, assess, cfgResolved, auditMode) {
		reviewJSON, syntaxType, checkErr := s.preCheckForTicket(ctx, inst, req.Database, req.Sql)
		if checkErr != nil {
			return nil, checkErr
		}
		ops, _ := json.Marshal([]string{"import", fmt.Sprintf("%d_statements", len(stmts))})
		ticket := &model.DbSqlTicket{
			ProjectID: projectID, InstanceID: instanceID, TicketType: model.DbTicketTypeSqlImport,
			SubmitterUserID: actorUserID(actor), SubmitterName: actorUsername(actor),
			DatabaseName: req.Database, SqlText: req.Sql, RiskLevel: maxRisk,
			SyntaxType: syntaxType, IsBackup: backup, AuditMode: auditMode,
			SqlFileRef: strings.TrimSpace(req.SqlFileRef),
			ParsedOpsJSON: string(ops), ReviewJSON: reviewJSON, Reason: strings.TrimSpace(req.Reason),
			Status: model.DbTicketStatusDraft,
		}
		if err := s.repo.CreateSqlTicket(ctx, ticket); err != nil {
			return nil, err
		}
		if err := s.initSqlTicketSteps(ctx, ticket); err != nil {
			return nil, err
		}
		s.auditTicketEvent(ctx, projectID, instanceID, actor, "ticket_create", ticket, map[string]any{"statement_count": len(stmts)})
		return &ExecuteResponse{
			Status: ticket.Status, TicketID: ticket.ID, RiskLevel: maxRisk,
			Message: fmt.Sprintf("导入 %d 条语句，已创建审批工单", len(stmts)),
		}, nil
	}
	var total int64
	if s.goInceptionAvailable(ctx,inst) {
		_, execErr := s.executeWriteViaEngine(ctx, inst, req.Database, req.Sql, backup, actor, nil)
		if execErr != nil {
			return nil, execErr
		}
		s.auditSQLWrite(ctx, inst, req.Database, req.Sql, "sql_import", actor, map[string]any{"statement_count": len(stmts)})
		return &ExecuteResponse{Status: "success", Message: fmt.Sprintf("已执行 %d 条语句", len(stmts))}, nil
	}
	for _, st := range stmts {
		n, err := s.runWriteSQL(ctx, inst, req.Database, st, actor, nil)
		if err != nil {
			return nil, err
		}
		total += n
	}
	s.auditSQLWrite(ctx, inst, req.Database, req.Sql, "sql_import", actor, map[string]any{"statement_count": len(stmts), "rows_affected": total})
	return &ExecuteResponse{Status: "success", RowsAffected: total, Message: fmt.Sprintf("已执行 %d 条语句", len(stmts))}, nil
}

func riskRank(level string) int {
	switch level {
	case model.DbRiskBlocked:
		return 5
	case model.DbRiskHigh:
		return 4
	case model.DbRiskMedium:
		return 3
	case model.DbRiskLow:
		return 1
	default:
		return 2
	}
}

func (s *Service) runWriteSQL(ctx context.Context, inst *model.DbInstance, database, sqlText string, actor *auth.CurrentUser, ticketID *uint) (int64, error) {
	release := s.acquireInstance(inst.ID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return 0, err
	}
	defer sess.Close()

	cfg := s.resolvedConfig(ctx)
	timeout := time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	execStmt := func(ctx context.Context, db *sql.DB) (int64, error) {
		res, err := db.ExecContext(ctx, sqlText)
		if err != nil {
			return 0, err
		}
		rows, _ := res.RowsAffected()
		return rows, nil
	}

	var rows int64
	var execErr error
	if isInstanceLevelDDL(sqlText) {
		rows, execErr = execStmt(qctx, sess.DB)
	} else {
		execErr = withDatabase(qctx, sess, inst, database, func(ctx context.Context, db *sql.DB) error {
			var innerErr error
			rows, innerErr = execStmt(ctx, db)
			return innerErr
		})
	}
	if execErr != nil {
		ex := &model.DbSqlExecution{
			ProjectID: inst.ProjectID, TicketID: ticketID, InstanceID: inst.ID,
			ExecutorUserID: actorUserID(actor), ExecutorName: actorUsername(actor),
			DatabaseName: database, SqlExcerpt: truncateSQL(sqlText, 2000),
			ErrorMessage: execErr.Error(), DurationMs: time.Since(start).Milliseconds(),
		}
		_ = s.repo.CreateSqlExecution(ctx, ex)
		return 0, execErr
	}
	hash := sha256.Sum256([]byte(sqlText))
	ex := &model.DbSqlExecution{
		ProjectID: inst.ProjectID, TicketID: ticketID, InstanceID: inst.ID,
		ExecutorUserID: actorUserID(actor), ExecutorName: actorUsername(actor),
		DatabaseName: database, StatementHash: hex.EncodeToString(hash[:]),
		SqlExcerpt: truncateSQL(sqlText, 2000), RowsAffected: rows,
		DurationMs: time.Since(start).Milliseconds(),
	}
	_ = s.repo.CreateSqlExecution(ctx, ex)
	return rows, nil
}

func (s *Service) ListTickets(ctx context.Context, q TicketListQuery) (*pagination.Result[TicketItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if q.Mine && q.MineViewer != nil {
		dbq := s.db.WithContext(ctx).Model(&model.DbSqlTicket{}).Where("project_id = ?", q.ProjectID)
		if tt := strings.TrimSpace(q.TicketType); tt != "" {
			dbq = dbq.Where("ticket_type = ?", tt)
		}
		scope := strings.TrimSpace(q.MineScope)
		if scope == "" {
			scope = "all"
		}
		mineTab := strings.TrimSpace(q.MineTab)
		if mineTab == "" {
			mineTab = "approval"
		}
		if mineTab == "execution" {
			if st := strings.TrimSpace(q.Status); st != "" {
				dbq = dbq.Where("status = ?", st)
			}
			switch scope {
			case "pending":
				dbq = s.filterTicketsExecutionPending(dbq, actorUserID(q.MineViewer))
			case "done":
				dbq = s.filterTicketsExecutionDone(dbq, actorUserID(q.MineViewer))
			default:
				dbq = s.filterTicketsExecutionMine(dbq, actorUserID(q.MineViewer))
			}
		} else {
			if st := strings.TrimSpace(q.Status); st != "" {
				dbq = dbq.Where("status = ?", st)
			}
			switch scope {
			case "pending":
				dbq = s.filterTicketsApprovalPending(dbq, q.MineViewer)
			case "done":
				dbq = s.filterTicketsApprovalDone(dbq, actorUserID(q.MineViewer))
			default:
				dbq = s.filterTicketsApprovalMine(dbq, q.MineViewer)
			}
		}
		var total int64
		if err := dbq.Count(&total).Error; err != nil {
			return nil, err
		}
		var list []model.DbSqlTicket
		if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
			return nil, err
		}
		items := make([]TicketItem, 0, len(list))
		for _, t := range list {
			items = append(items, s.toTicketItem(ctx, t))
		}
		s.enrichTicketMineStatus(ctx, items, q.MineViewer, mineTab)
		return paginate(items, total, page, pageSize), nil
	}
	list, total, err := s.repo.ListSqlTickets(ctx, repository.DbSqlTicketListParams{
		ProjectID: q.ProjectID, Status: q.Status, TicketType: q.TicketType,
		Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]TicketItem, 0, len(list))
	for _, t := range list {
		items = append(items, s.toTicketItem(ctx, t))
	}
	return paginate(items, total, q.Page, q.PageSize), nil
}

func (s *Service) ApproveTicket(ctx context.Context, projectID, ticketID uint, comment string, actor *auth.CurrentUser) error {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	if ticket.Status != model.DbTicketStatusPendingApproval {
		return constants.ErrBadRequestWithMsg("工单已结束，无法审批")
	}
	steps, err := s.repo.ListSqlTicketSteps(ctx, ticketID)
	if err != nil {
		return err
	}
	var cur *model.DbSqlTicketStep
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
	if err != nil || !ok {
		return constants.ErrForbidden
	}
	if err := s.forbidSelfApprove(ctx, actor, ticket.SubmitterUserID); err != nil {
		return err
	}
	now := time.Now()
	uid := actorUserID(actor)
	cur.Status = model.DbApprovalStepApproved
	cur.ReviewerUserID = &uid
	cur.ReviewerName = actorUsername(actor)
	cur.ReviewComment = strings.TrimSpace(comment)
	cur.ReviewedAt = &now
	if err := s.repo.UpdateSqlTicketStep(ctx, cur); err != nil {
		return err
	}
	if err := s.advanceSqlTicketAfterApproval(ctx, ticket, cur); err != nil {
		return err
	}
	s.auditTicketEvent(ctx, projectID, ticket.InstanceID, actor, "ticket_approve", ticket, map[string]any{"comment": strings.TrimSpace(comment)})
	return nil
}

func (s *Service) RejectTicket(ctx context.Context, projectID, ticketID uint, comment string, actor *auth.CurrentUser) error {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	if ticket.Status != model.DbTicketStatusPendingApproval {
		return constants.ErrBadRequestWithMsg("工单已结束，无法驳回")
	}
	steps, err := s.repo.ListSqlTicketSteps(ctx, ticketID)
	if err != nil {
		return err
	}
	for i := range steps {
		if steps[i].Status == model.DbApprovalStepPending {
			ok, err := s.userCanApproveStep(ctx, actor, steps[i].UserGroupID)
			if err != nil {
				return err
			}
			if !ok {
				return constants.ErrForbidden
			}
			now := time.Now()
			uid := actorUserID(actor)
			steps[i].Status = model.DbApprovalStepRejected
			steps[i].ReviewerUserID = &uid
			steps[i].ReviewerName = actorUsername(actor)
			steps[i].ReviewComment = strings.TrimSpace(comment)
			steps[i].ReviewedAt = &now
			if err := s.repo.UpdateSqlTicketStep(ctx, &steps[i]); err != nil {
				return err
			}
			break
		}
	}
	ticket.Status = model.DbTicketStatusRejected
	if err := s.repo.UpdateSqlTicket(ctx, ticket); err != nil {
		return err
	}
	s.auditTicketEvent(ctx, projectID, ticket.InstanceID, actor, "ticket_reject", ticket, map[string]any{"comment": strings.TrimSpace(comment)})
	return nil
}

func (s *Service) ExecuteTicket(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) error {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	if ticket.Status != model.DbTicketStatusPendingExecution && ticket.Status != model.DbTicketStatusApproved {
		return constants.ErrBadRequestWithMsg("工单状态不允许执行")
	}
	if ticket.SubmitterUserID != actorUserID(actor) && !auth.IsSuperAdminRole(actor.RoleCodes) {
		perm, permErr := s.GetEffectivePermission(ctx, projectID, ticket.InstanceID, actor)
		if permErr != nil {
			return permErr
		}
		if perm == nil || !perm.CanManage {
			return constants.ErrForbidden
		}
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, ticket.InstanceID)
	if err != nil {
		return err
	}
	needDDL := reDDL.MatchString(strings.ToUpper(stripSQLComments(ticket.SqlText)))
	if err := s.checkWritePermission(ctx, projectID, inst, ticket.DatabaseName, ticket.SqlText, needDDL, actor); err != nil {
		return err
	}
	claim := s.db.WithContext(ctx).Model(&model.DbSqlTicket{}).
		Where("id = ? AND project_id = ? AND status IN ?", ticketID, projectID,
			[]string{model.DbTicketStatusPendingExecution, model.DbTicketStatusApproved}).
		Update("status", model.DbTicketStatusExecuting)
	if claim.Error != nil {
		return claim.Error
	}
	if claim.RowsAffected != 1 {
		return constants.ErrBadRequestWithMsg("工单状态不允许执行或正在执行中")
	}
	ticket, err = s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	tid := ticket.ID
	cfg := s.resolvedConfig(ctx)
	backup := ticket.IsBackup || cfg.GoInceptionBackup
	var execJSON string
	var execErr error
	if s.goInceptionAvailable(ctx,inst) {
		execJSON, execErr = s.executeWriteViaEngine(ctx, inst, ticket.DatabaseName, ticket.SqlText, backup, actor, &tid)
	} else {
		switch ticket.TicketType {
		case model.DbTicketTypeSqlImport:
			for _, st := range splitSQLStatements(ticket.SqlText) {
				if _, execErr = s.runWriteSQL(ctx, inst, ticket.DatabaseName, st, actor, &tid); execErr != nil {
					break
				}
			}
		default:
			_, execErr = s.runWriteSQL(ctx, inst, ticket.DatabaseName, ticket.SqlText, actor, &tid)
		}
	}
	ticket.ExecuteJSON = execJSON
	if execErr != nil {
		ticket.Status = model.DbTicketStatusFailed
		_ = s.repo.UpdateSqlTicket(ctx, ticket)
		changeevent.Record(ctx, changeevent.Input{
			ProjectID: projectID,
			Source:    model.ChangeSourceDbmgmt,
			Action:    "sql_ticket_execute",
			RiskLevel: model.ChangeRiskHigh,
			Status:    model.ChangeStatusFailed,
			Summary:   fmt.Sprintf("SQL 工单 #%d 执行失败", ticket.ID),
			Payload:   map[string]any{"ticket_id": ticket.ID, "instance_id": ticket.InstanceID},
		})
		return constants.ErrBadRequestWithMsg("SQL 执行失败: " + execErr.Error())
	}
	ticket.Status = model.DbTicketStatusSuccess
	if err := s.repo.UpdateSqlTicket(ctx, ticket); err != nil {
		return err
	}
	changeevent.Record(ctx, changeevent.Input{
		ProjectID: projectID,
		Source:    model.ChangeSourceDbmgmt,
		Action:    "sql_ticket_execute",
		RiskLevel: model.ChangeRiskHigh,
		Status:    model.ChangeStatusSucceeded,
		Summary:   fmt.Sprintf("SQL 工单 #%d 执行成功", ticket.ID),
		Payload:   map[string]any{"ticket_id": ticket.ID, "instance_id": ticket.InstanceID},
	})
	s.auditTicketEvent(ctx, projectID, ticket.InstanceID, actor, "ticket_execute", ticket, nil)
	return nil
}

func (s *Service) ListTicketSteps(ctx context.Context, projectID, ticketID uint) ([]model.DbSqlTicketStep, error) {
	if _, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID); err != nil {
		return nil, err
	}
	return s.repo.ListSqlTicketSteps(ctx, ticketID)
}

func (s *Service) ListAccessRequestSteps(ctx context.Context, projectID, requestID uint) ([]model.DbAccessRequestStep, error) {
	if _, err := s.repo.GetAccessRequestInProject(ctx, projectID, requestID); err != nil {
		return nil, err
	}
	return s.repo.ListAccessRequestSteps(ctx, requestID)
}
