package system

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/exportutil"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

const (
	// auditQueueSize 有界队列：DB 变慢时丢弃新审计而非阻塞请求路径。
	auditQueueSize = 4096
	// auditWriteTimeout 单条审计写库超时，脱离请求 ctx（客户端断开不能丢审计）。
	auditWriteTimeout = 5 * time.Second
)

type OperationLogListQuery struct {
	Method     string `form:"method"`
	Path       string `form:"path"`
	StatusCode *int   `form:"status_code"`
	Start      string `form:"start"`
	End        string `form:"end"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type OperationLogService struct {
	repo    interfaces.OperationLogRepository
	queue   chan model.OperationLog
	started sync.Once
	dropped atomic.Int64
}

// NewOperationLogService 创建相关逻辑。
func NewOperationLogService(repo interfaces.OperationLogRepository) *OperationLogService {
	return &OperationLogService{repo: repo, queue: make(chan model.OperationLog, auditQueueSize)}
}

// Record 将审计条目放入有界队列异步落库，永不阻塞请求路径。
func (s *OperationLogService) Record(_ context.Context, entry model.OperationLog) error {
	s.started.Do(func() { go s.consumeQueue() })
	select {
	case s.queue <- entry:
	default:
		if n := s.dropped.Add(1); n == 1 || n%1000 == 0 {
			logx.With(context.Background(), "component", "operation-log").
				Warn("operation audit queue full, entry dropped", "dropped_total", n)
		}
	}
	return nil
}

func (s *OperationLogService) consumeQueue() {
	for entry := range s.queue {
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		if err := s.repo.Create(ctx, &entry); err != nil {
			logx.With(ctx, "component", "operation-log").
				Error("operation audit persist failed", "error", err, "path", entry.Path)
		}
		cancel()
	}
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
	start, end, err := resolveOperationLogWindow(query.Start, query.End, time.Now())
	if err != nil {
		return repository.OperationLogListParams{}, err
	}
	params.Start, params.End = &start, &end
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

// operationLogTimeLayouts 支持完整时间与纯日期两种入参；纯日期按当天 0 点/23:59:59 处理。
var operationLogTimeLayouts = []string{"2006-01-02 15:04:05", "2006-01-02"}

// resolveOperationLogWindow 解析查询时间窗：都未提供时默认最近 7 天（无界全表扫描
// 是操作历史页慢查询的主要来源），跨度上限 92 天。
func resolveOperationLogWindow(startRaw, endRaw string, now time.Time) (time.Time, time.Time, error) {
	parse := func(raw string, endOfDay bool) (time.Time, bool, error) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return time.Time{}, false, nil
		}
		for i, layout := range operationLogTimeLayouts {
			t, err := time.ParseInLocation(layout, raw, time.Local)
			if err != nil {
				continue
			}
			if i == 1 && endOfDay {
				t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			}
			return t, true, nil
		}
		return time.Time{}, false, constants.ErrBadRequestWithMsg("时间格式应为 2006-01-02 15:04:05 或 2006-01-02")
	}

	start, startSet, err := parse(startRaw, false)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, endSet, err := parse(endRaw, true)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	switch {
	case !startSet && !endSet:
		end = now
		start = now.AddDate(0, 0, -7)
	case !endSet:
		end = now
	case !startSet:
		start = end.AddDate(0, 0, -7)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, constants.ErrBadRequestWithMsg("开始时间不能晚于结束时间")
	}
	if end.Sub(start) > 92*24*time.Hour {
		return time.Time{}, time.Time{}, constants.ErrBadRequestWithMsg("时间跨度不能超过 92 天")
	}
	return start, end, nil
}

// Detail 查看单条操作日志（含请求头/请求体/响应体）：超管可看全部，普通管理员仅可看自己的。
func (s *OperationLogService) Detail(ctx context.Context, actor *auth.CurrentUser, id uint) (*model.OperationLog, error) {
	if actor == nil {
		return nil, constants.ErrUnauthorized
	}
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg("操作日志不存在")
		}
		return nil, bizerrors.Pass(ctx, "operation-log", "Detail", err)
	}
	if !auth.IsSuperAdminRole(actor.RoleCodes) && item.Username != actor.Username {
		return nil, constants.ErrForbiddenWithMsg("仅可查看自己的操作日志详情")
	}
	return item, nil
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
	// 导出走不限页取数；行数上限由下面的 total 校验兜底。
	params.Unbounded = true
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
