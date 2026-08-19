package dbmgmt

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/goinception"
)

type RollbackItem = goinception.RollbackItem

type OSCControlRequest struct {
	Command string `json:"command" binding:"required"`
}

type OSCJobItem struct {
	OrderID     int    `json:"order_id"`
	SQL         string `json:"sql"`
	SQLSHA1     string `json:"sqlsha1"`
	Stage       string `json:"stage"`
	StageStatus string `json:"stage_status"`
}

func (s *Service) GetTicketRollback(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) ([]RollbackItem, error) {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.Status != model.DbTicketStatusSuccess && ticket.Status != model.DbTicketStatusFailed {
		return nil, constants.ErrBadRequestWithMsg("仅已执行完成的工单可查询回滚 SQL")
	}
	if !ticket.IsBackup {
		return nil, constants.ErrBadRequestWithMsg("该工单未开启备份，无法生成回滚 SQL")
	}
	rows := goinception.ParseExecuteRows(ticket.ExecuteJSON)
	if len(rows) == 0 {
		return nil, constants.ErrBadRequestWithMsg("无执行结果，无法查询回滚 SQL")
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, ticket.InstanceID)
	if err != nil {
		return nil, err
	}
	if err := s.checkWritePermission(ctx, projectID, inst, ticket.DatabaseName, ticket.SqlText, true, actor); err != nil {
		if err2 := s.checkWritePermission(ctx, projectID, inst, ticket.DatabaseName, ticket.SqlText, false, actor); err2 != nil {
			if ticket.SubmitterUserID != actorUserID(actor) && !auth.IsSuperAdminRole(actor.RoleCodes) {
				if qerr := s.checkQueryPermission(ctx, projectID, inst, ticket.DatabaseName, actor); qerr != nil {
					return nil, constants.ErrForbidden
				}
			}
		}
	}
	release := s.acquireInstance(inst.ID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	items, err := goinception.FetchRollbackFromExecuteRows(ctx, sess.DB, rows)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("查询回滚 SQL 失败: " + err.Error())
	}
	if len(items) == 0 {
		return nil, constants.ErrBadRequestWithMsg("未在 goInception 备份库中找到回滚 SQL，请确认：1) goInception config.toml 已配置 backup 且执行时 --backup=1；2) 实例账号对备份库（如 10_10_10_103_3306_test）有 SELECT 权限")
	}
	return items, nil
}

func (s *Service) ListTicketOSCJobs(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) ([]OSCJobItem, error) {
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTicketOSCAccess(ctx, projectID, ticket, actor); err != nil {
		return nil, err
	}
	rows := goinception.ParseExecuteRows(ticket.ExecuteJSON)
	if len(rows) == 0 {
		rows = goinception.ParseExecuteRows(ticket.ReviewJSON)
	}
	jobs := goinception.ExtractOSCJobs(rows)
	out := make([]OSCJobItem, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, OSCJobItem{
			OrderID: j.OrderID, SQL: j.SQL, SQLSHA1: j.SQLSHA1,
			Stage: j.Stage, StageStatus: j.StageStatus,
		})
	}
	return out, nil
}

func (s *Service) ensureTicketOSCAccess(ctx context.Context, projectID uint, ticket *model.DbSqlTicket, actor *auth.CurrentUser) error {
	if actor == nil {
		return constants.ErrForbidden
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if ticket.SubmitterUserID == actorUserID(actor) {
		return nil
	}
	if ok, _ := s.isProjectAdminOrOwner(ctx, projectID, actor); ok {
		return nil
	}
	perm, _ := s.GetEffectivePermission(ctx, projectID, ticket.InstanceID, actor)
	if perm != nil && perm.CanManage {
		return nil
	}
	return constants.ErrForbidden
}

func (s *Service) GetTicketOSCPercent(ctx context.Context, projectID, ticketID uint, sqlsha1 string, actor *auth.CurrentUser) (*goinception.ReviewSet, error) {
	if err := s.ensureOSCAccess(ctx, projectID, ticketID, sqlsha1, actor); err != nil {
		return nil, err
	}
	return s.goInceptionClient(ctx).OSCControl(ctx, sqlsha1, "get")
}

func (s *Service) ControlTicketOSC(ctx context.Context, projectID, ticketID uint, sqlsha1 string, command string, actor *auth.CurrentUser) (*goinception.ReviewSet, error) {
	if err := s.ensureOSCAccess(ctx, projectID, ticketID, sqlsha1, actor); err != nil {
		return nil, err
	}
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd != "kill" && cmd != "pause" && cmd != "resume" {
		return nil, constants.ErrBadRequestWithMsg("不支持的 OSC 命令: " + command)
	}
	return s.goInceptionClient(ctx).OSCControl(ctx, sqlsha1, cmd)
}

func (s *Service) ensureOSCAccess(ctx context.Context, projectID, ticketID uint, sqlsha1 string, actor *auth.CurrentUser) error {
	if !s.resolvedConfig(ctx).GoInceptionEnabled {
		return constants.ErrBadRequestWithMsg("未启用 goInception")
	}
	ticket, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return err
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, ticket.InstanceID)
	if err != nil {
		return err
	}
	if !s.goInceptionAvailable(ctx, inst) {
		return constants.ErrBadRequestWithMsg("当前实例不支持 goInception OSC")
	}
	if ticket.SubmitterUserID != actorUserID(actor) && !auth.IsSuperAdminRole(actor.RoleCodes) {
		perm, _ := s.GetEffectivePermission(ctx, projectID, ticket.InstanceID, actor)
		if perm == nil || !perm.CanManage {
			if ok, _ := s.isProjectAdminOrOwner(ctx, projectID, actor); !ok {
				return constants.ErrForbidden
			}
		}
	}
	rows := goinception.ParseExecuteRows(ticket.ExecuteJSON)
	if len(rows) == 0 {
		rows = goinception.ParseExecuteRows(ticket.ReviewJSON)
	}
	jobs := goinception.ExtractOSCJobs(rows)
	for _, j := range jobs {
		if j.SQLSHA1 == sqlsha1 {
			return nil
		}
	}
	return constants.ErrBadRequestWithMsg("无效的 OSC 任务")
}

type RollbackPreviewResponse struct {
	SourceTicketID uint   `json:"source_ticket_id"`
	ItemCount      int    `json:"item_count"`
	SQL            string `json:"sql"`
	DatabaseName   string `json:"database_name"`
	InstanceID     uint   `json:"instance_id"`
}

type SubmitRollbackTicketRequest struct {
	Comment string `json:"comment"`
}

func (s *Service) PreviewRollbackTicket(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) (*RollbackPreviewResponse, error) {
	source, sqlText, err := s.buildRollbackSQL(ctx, projectID, ticketID, actor)
	if err != nil {
		return nil, err
	}
	items, _ := s.GetTicketRollback(ctx, projectID, ticketID, actor)
	return &RollbackPreviewResponse{
		SourceTicketID: ticketID,
		ItemCount:      len(items),
		SQL:            sqlText,
		DatabaseName:   source.DatabaseName,
		InstanceID:     source.InstanceID,
	}, nil
}

func (s *Service) SubmitRollbackTicket(ctx context.Context, projectID, ticketID uint, req SubmitRollbackTicketRequest, actor *auth.CurrentUser) (*ExecuteResponse, error) {
	source, sqlText, err := s.buildRollbackSQL(ctx, projectID, ticketID, actor)
	if err != nil {
		return nil, err
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, source.InstanceID)
	if err != nil {
		return nil, err
	}
	reason := fmt.Sprintf("回滚工单，来源 #%d", source.ID)
	if c := strings.TrimSpace(req.Comment); c != "" {
		reason = reason + "：" + c
	}
	return s.createSqlExecuteTicket(ctx, projectID, inst, source.DatabaseName, sqlText, reason, actor, true)
}

func (s *Service) buildRollbackSQL(ctx context.Context, projectID, ticketID uint, actor *auth.CurrentUser) (*model.DbSqlTicket, string, error) {
	source, err := s.repo.GetSqlTicketInProject(ctx, projectID, ticketID)
	if err != nil {
		return nil, "", err
	}
	items, err := s.GetTicketRollback(ctx, projectID, ticketID, actor)
	if err != nil {
		return nil, "", err
	}
	if len(items) == 0 {
		return nil, "", constants.ErrBadRequestWithMsg("未找到可回滚的 SQL")
	}
	var parts []string
	for _, item := range items {
		stmt := strings.TrimSpace(item.RollbackSQL)
		if stmt != "" {
			parts = append(parts, stmt)
		}
	}
	if len(parts) == 0 {
		return nil, "", constants.ErrBadRequestWithMsg("回滚 SQL 为空")
	}
	return source, strings.Join(parts, "\n"), nil
}
