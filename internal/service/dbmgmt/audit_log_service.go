package dbmgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
)

// AuditLogItem 审计日志列表项（含实例展示信息）。
type AuditLogItem struct {
	model.DbAuditLog
	InstanceName  string `json:"instance_name,omitempty"`
	InstanceLabel string `json:"instance_label,omitempty"`
}

func formatInstanceLabel(inst model.DbInstance) string {
	driver := strings.ToUpper(strings.TrimSpace(inst.Driver))
	if driver == "" {
		driver = "MYSQL"
	}
	return fmt.Sprintf("%s-%s-%d", driver, inst.Host, inst.Port)
}

func (s *Service) auditSQLWrite(ctx context.Context, inst *model.DbInstance, database, sqlText, action string, actor *auth.CurrentUser, extra map[string]any) {
	if inst == nil || actor == nil {
		return
	}
	detail := map[string]any{
		"database": database,
		"sql":      truncateSQL(sqlText, 500),
	}
	for k, v := range extra {
		detail[k] = v
	}
	_ = s.writeAudit(ctx, inst.ProjectID, &inst.ID, actor, action, detail)
}

func (s *Service) auditTicketEvent(ctx context.Context, projectID uint, instanceID uint, actor *auth.CurrentUser, action string, ticket *model.DbSqlTicket, extra map[string]any) {
	if ticket == nil || actor == nil {
		return
	}
	detail := map[string]any{
		"ticket_id":   ticket.ID,
		"ticket_type": ticket.TicketType,
		"database":    ticket.DatabaseName,
	}
	if sql := strings.TrimSpace(ticket.SqlText); sql != "" {
		detail["sql"] = truncateSQL(sql, 500)
	}
	if ref := strings.TrimSpace(ticket.SqlFileRef); ref != "" {
		detail["sql_file_ref"] = ref
	}
	for k, v := range extra {
		detail[k] = v
	}
	iid := instanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, action, detail)
}

func (s *Service) ListAuditLogs(ctx context.Context, projectID uint, instanceID uint, action string, page, pageSize int, actor *auth.CurrentUser) (*pagination.Result[AuditLogItem], error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return nil, err
	}
	list, total, err := s.repo.ListAuditLogs(ctx, repository.DbAuditLogListParams{
		ProjectID: projectID, InstanceID: instanceID, Action: strings.TrimSpace(action), Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	instMap := map[uint]model.DbInstance{}
	if instances, _, err := s.repo.ListInstances(ctx, repository.DbInstanceListParams{
		ProjectID: projectID, Page: 1, PageSize: 5000,
	}); err == nil {
		for _, inst := range instances {
			instMap[inst.ID] = inst
		}
	}
	items := make([]AuditLogItem, 0, len(list))
	for _, log := range list {
		item := AuditLogItem{DbAuditLog: log}
		if log.InstanceID != nil {
			if inst, ok := instMap[*log.InstanceID]; ok {
				item.InstanceName = inst.Name
				item.InstanceLabel = formatInstanceLabel(inst)
			}
		}
		items = append(items, item)
	}
	return paginate(items, total, page, pageSize), nil
}

// ParseAuditDetail 将 detail_json 解析为 map（供测试或扩展）。
func ParseAuditDetail(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}
