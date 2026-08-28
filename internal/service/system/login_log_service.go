package system

import (
	"context"
	"io"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
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

// List 查询列表相关的业务逻辑。
func (s *LoginLogService) List(ctx context.Context, query LoginLogListQuery) (*pagination.Result[model.LoginLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, repository.LoginLogListParams{
		Username: query.Username,
		Status:   query.Status,
		Source:   query.Source,
		Page:     page,
		PageSize: pageSize,
	})
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

// Delete 删除相关的业务逻辑。
func (s *LoginLogService) Delete(ctx context.Context, id uint) error {
	return s.repo.DeleteByID(ctx, id)
}

// DeleteBatch 删除相关的业务逻辑。
func (s *LoginLogService) DeleteBatch(ctx context.Context, ids []uint) error {
	return s.repo.DeleteByIDs(ctx, ids)
}

// Export writes login logs matching query to writer as Excel.
func (s *LoginLogService) Export(ctx context.Context, query LoginLogListQuery, w io.Writer) error {
	list, total, err := s.repo.List(ctx, repository.LoginLogListParams{
		Username: query.Username,
		Status:   query.Status,
		Source:   query.Source,
		Page:     1,
		PageSize: exportutil.MaxExcelExportRows + 1,
	})
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
