package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type OperationLogListParams struct {
	Method        string
	Path          string
	UsernameExact string // 精确匹配（普通管理员仅能查看自己的日志）
	StatusCode    *int
	Start         *time.Time
	End           *time.Time
	Page          int
	PageSize      int
	// Unbounded 跳过 MaxPageSize 钳制，仅导出路径可置 true。
	Unbounded bool
}

type OperationLogRepository struct {
	db *gorm.DB
}

func NewOperationLogRepository(db *gorm.DB) OperationLogRepo {
	return &OperationLogRepository{db: db}
}

// operationLogListColumns 列表/导出只取元数据列；请求头与请求/响应体可能极大，
// 由 Detail 单条读取（列名须与 model.OperationLog 对齐）。
const operationLogListColumns = "id,created_at,user_id,username,nickname,ip,method,path,status_code,latency_ms"

func (r *OperationLogRepository) Create(ctx context.Context, log *model.OperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *OperationLogRepository) List(ctx context.Context, p OperationLogListParams) ([]model.OperationLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.OperationLog{}).Select(operationLogListColumns)
	if p.Method != "" {
		q = q.Where("method = ?", p.Method)
	}
	if p.Path != "" {
		// 前缀匹配可命中 idx_oplogs 里的 path 索引，%% 双侧 LIKE 无法用索引。
		q = q.Where("path LIKE ?", p.Path+"%")
	}
	if p.UsernameExact != "" {
		q = q.Where("username = ?", p.UsernameExact)
	}
	if p.StatusCode != nil {
		q = q.Where("status_code = ?", *p.StatusCode)
	}
	if p.Start != nil {
		q = q.Where("created_at >= ?", *p.Start)
	}
	if p.End != nil {
		q = q.Where("created_at <= ?", *p.End)
	}

	var list []model.OperationLog
	var total int64
	var err error
	if p.Unbounded {
		total, err = listWithPaginationRaw(q, p.Page, p.PageSize, "id DESC", &list)
	} else {
		total, err = listWithPagination(q, p.Page, p.PageSize, "id DESC", &list)
	}
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *OperationLogRepository) GetByID(ctx context.Context, id uint) (*model.OperationLog, error) {
	var it model.OperationLog
	if err := r.db.WithContext(ctx).First(&it, id).Error; err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *OperationLogRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.OperationLog{}, id).Error
}

func (r *OperationLogRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.OperationLog{}).Error
}

// DeleteOlderThan 分批删除 cutoff 之前的日志并返回总数；每批最多 1000 行，避免长事务锁表。
func (r *OperationLogRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	const batch = 1000
	var total int64
	for {
		res := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Limit(batch).Delete(&model.OperationLog{})
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < batch {
			return total, nil
		}
	}
}
