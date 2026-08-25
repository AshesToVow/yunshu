package system

import (
	"context"
	"io"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

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

// List 查询列表相关的业务逻辑。
func (s *OperationLogService) List(ctx context.Context, query OperationLogListQuery) (*pagination.Result[model.OperationLog], error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, repository.OperationLogListParams{
		Method:     query.Method,
		Path:       query.Path,
		StatusCode: query.StatusCode,
		Page:       page,
		PageSize:   pageSize,
	})
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

// Delete 删除相关的业务逻辑。
func (s *OperationLogService) Delete(ctx context.Context, id uint) error {
	return s.repo.DeleteByID(ctx, id)
}

// DeleteBatch 删除相关的业务逻辑。
func (s *OperationLogService) DeleteBatch(ctx context.Context, ids []uint) error {
	return s.repo.DeleteByIDs(ctx, ids)
}

// Export writes operation logs matching query to writer as Excel.
func (s *OperationLogService) Export(ctx context.Context, query OperationLogListQuery, w io.Writer) error {
	list, total, err := s.repo.List(ctx, repository.OperationLogListParams{
		Method:     query.Method,
		Path:       query.Path,
		StatusCode: query.StatusCode,
		Page:       1,
		PageSize:   exportutil.MaxExcelExportRows + 1,
	})
	if err != nil {
		return bizerrors.Pass(ctx, "operation-log", "Export", err)
	}
	if total > exportutil.MaxExcelExportRows {
		return exportutil.ExportRowLimitError(total)
	}
	f := excelize.NewFile()
	sheet := "Sheet1"
	_ = f.SetSheetRow(sheet, "A1", &[]interface{}{"ID", "Method", "Path", "StatusCode", "LatencyMs", "IP", "CreatedAt", "User"})
	for i, l := range list {
		row := []interface{}{
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
