package system

import (
	"context"
	"io"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/xuri/excelize/v2"
)

type LoginLogListQuery struct {
	Username string `form:"username"`
	Status   *int   `form:"status"` // 1 成功 0 失败
	Source   string `form:"source"` // password | email
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type LoginLogService struct {
	repo interfaces.LoginLogRepository
}

// NewLoginLogService 创建相关逻辑。
func NewLoginLogService(repo interfaces.LoginLogRepository) *LoginLogService {
	return &LoginLogService{repo: repo}
}

// Record 执行对应的业务逻辑。
func (s *LoginLogService) Record(ctx context.Context, entry model.LoginLog) error {
	return s.repo.Create(ctx, &entry)
}

// loginLogListParams 普通管理员只能查看自己的登录日志，仅内置超级管理员可看全量。
func loginLogListParams(actor *auth.CurrentUser, query LoginLogListQuery, page, pageSize int) (repository.LoginLogListParams, error) {
	if actor == nil {
		return repository.LoginLogListParams{}, constants.ErrUnauthorized
	}
	params := repository.LoginLogListParams{
		Status:   query.Status,
		Source:   query.Source,
		Page:     page,
		PageSize: pageSize,
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		params.Username = strings.TrimSpace(query.Username)
		return params, nil
	}
	username := strings.TrimSpace(actor.Username)
	if username == "" {
		return repository.LoginLogListParams{}, constants.ErrUnauthorized
	}
	params.UsernameExact = username
	return params, nil
}

// List 查询列表相关的业务逻辑。
func (s *LoginLogService) List(ctx context.Context, actor *auth.CurrentUser, query LoginLogListQuery) (*pagination.Result[model.LoginLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params, err := loginLogListParams(actor, query, page, pageSize)
	if err != nil {
		return nil, err
	}
	list, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "login-log", "List", err)
	}
	return &pagination.Result[model.LoginLog]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Delete 删除相关的业务逻辑（仅内置超级管理员）。
func (s *LoginLogService) Delete(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if err := requireSuperAdminForLogMutation(actor, "删除登录日志"); err != nil {
		return err
	}
	return s.repo.DeleteByID(ctx, id)
}

// DeleteBatch 删除相关的业务逻辑（仅内置超级管理员）。
func (s *LoginLogService) DeleteBatch(ctx context.Context, actor *auth.CurrentUser, ids []uint) error {
	if err := requireSuperAdminForLogMutation(actor, "删除登录日志"); err != nil {
		return err
	}
	return s.repo.DeleteByIDs(ctx, ids)
}

// requireSuperAdminForLogMutation 审计日志属于取证数据，仅内置超级管理员可删除。
func requireSuperAdminForLogMutation(actor *auth.CurrentUser, action string) error {
	if actor == nil {
		return constants.ErrUnauthorized
	}
	if !auth.IsSuperAdminRole(actor.RoleCodes) {
		return constants.ErrForbiddenWithMsg("仅内置超级管理员可以" + action)
	}
	return nil
}

// Export writes login logs matching query to writer as Excel.
func (s *LoginLogService) Export(ctx context.Context, actor *auth.CurrentUser, query LoginLogListQuery, w io.Writer) error {
	params, err := loginLogListParams(actor, query, 1, exportutil.MaxExcelExportRows+1)
	if err != nil {
		return err
	}
	list, total, err := s.repo.List(ctx, params)
	if err != nil {
		return bizerrors.Pass(ctx, "login-log", "Export", err)
	}
	if total > exportutil.MaxExcelExportRows {
		return exportutil.ExportRowLimitError(total)
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]any{"ID", "Username", "IP", "Source", "Status", "Detail", "UserAgent", "CreatedAt"})
	for i, l := range list {
		row := []any{l.ID, l.Username, l.IP, l.Source, l.Status, l.Detail, l.UserAgent, l.CreatedAt}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	return f.Write(w)
}
