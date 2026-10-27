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

type OperationLogListQuery struct {
	Method     string `form:"method"`
	Path       string `form:"path"`
	StatusCode *int   `form:"status_code"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type OperationLogService struct {
	repo interfaces.OperationLogRepository
}

// NewOperationLogService 创建相关逻辑。
func NewOperationLogService(repo interfaces.OperationLogRepository) *OperationLogService {
	return &OperationLogService{repo: repo}
}

// Record 执行对应的业务逻辑。
func (s *OperationLogService) Record(ctx context.Context, entry model.OperationLog) error {
	return s.repo.Create(ctx, &entry)
}

// operationLogListParams 普通管理员只能查看自己的操作日志，仅内置超级管理员可看全量。
func operationLogListParams(actor *auth.CurrentUser, query OperationLogListQuery, page, pageSize int) (repository.OperationLogListParams, error) {
	if actor == nil {
		return repository.OperationLogListParams{}, constants.ErrUnauthorized
	}
	params := repository.OperationLogListParams{
		Method:     query.Method,
		Path:       query.Path,
		StatusCode: query.StatusCode,
		Page:       page,
		PageSize:   pageSize,
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return params, nil
	}
	username := strings.TrimSpace(actor.Username)
	if username == "" {
		return repository.OperationLogListParams{}, constants.ErrUnauthorized
	}
	params.UsernameExact = username
	return params, nil
}

// List 查询列表相关的业务逻辑。
func (s *OperationLogService) List(ctx context.Context, actor *auth.CurrentUser, query OperationLogListQuery) (*pagination.Result[model.OperationLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params, err := operationLogListParams(actor, query, page, pageSize)
	if err != nil {
		return nil, err
	}
	list, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "operation-log", "List", err)
	}
	return &pagination.Result[model.OperationLog]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Delete 删除相关的业务逻辑（仅内置超级管理员）。
func (s *OperationLogService) Delete(ctx context.Context, actor *auth.CurrentUser, id uint) error {
	if err := requireSuperAdminForLogMutation(actor, "删除操作日志"); err != nil {
		return err
	}
	return s.repo.DeleteByID(ctx, id)
}

// DeleteBatch 删除相关的业务逻辑（仅内置超级管理员）。
func (s *OperationLogService) DeleteBatch(ctx context.Context, actor *auth.CurrentUser, ids []uint) error {
	if err := requireSuperAdminForLogMutation(actor, "删除操作日志"); err != nil {
		return err
	}
	return s.repo.DeleteByIDs(ctx, ids)
}

// Export writes operation logs matching query to writer as Excel.
func (s *OperationLogService) Export(ctx context.Context, actor *auth.CurrentUser, query OperationLogListQuery, w io.Writer) error {
	params, err := operationLogListParams(actor, query, 1, exportutil.MaxExcelExportRows+1)
	if err != nil {
		return err
	}
	list, total, err := s.repo.List(ctx, params)
	if err != nil {
		return bizerrors.Pass(ctx, "operation-log", "Export", err)
	}
	if total > exportutil.MaxExcelExportRows {
		return exportutil.ExportRowLimitError(total)
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]any{"ID", "Method", "Path", "StatusCode", "LatencyMs", "IP", "CreatedAt", "User"})
	for i, l := range list {
		row := []any{
			l.ID,
			l.Method,
			l.Path,
			l.StatusCode,
			l.LatencyMs,
			l.IP,
			l.CreatedAt,
			l.Username,
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		_ = f.SetSheetRow(sheet, cell, &row)
	}
	return f.Write(w)
}
