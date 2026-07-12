package dbmgmt

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "dbmgmt" }
func (m *module) Description() string { return "多类型数据库接入、SQL 控制台与审批工单" }

func (m *module) Models() []any {
	return []any{
		&model.DbInstance{},
		&model.DbAccessGrant{},
		&model.DbApprovalFlowStage{},
		&model.DbAccessRequest{},
		&model.DbAccessRequestStep{},
		&model.DbSqlTicket{},
		&model.DbSqlTicketStep{},
		&model.DbSqlExecution{},
		&model.DbAuditLog{},
		&model.DbAppUserRequest{},
		&model.DbAppUserRequestStep{},
		&model.DbInstanceAccount{},
	}
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil {
		return nil
	}
	if svc := rt.DbmgmtSvc(); svc != nil {
		go svc.RunBackgroundWorkers(bgCtx)
	}
	return nil
}
