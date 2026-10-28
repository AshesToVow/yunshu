package repository

import (
	"context"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

type LoginLogListParams struct {
	Username      string
	UsernameExact string // 精确匹配（普通管理员仅能查看自己的日志）
	Status        *int
	Source        string
	Page          int
	PageSize      int
	// Unbounded 跳过 MaxPageSize 钳制，仅导出路径可置 true。
	Unbounded bool
}

type LoginLogRepository struct {
	db *gorm.DB
}

func NewLoginLogRepository(db *gorm.DB) LoginLogRepo {
	return &LoginLogRepository{db: db}
}

func (r *LoginLogRepository) Create(ctx context.Context, log *model.LoginLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *LoginLogRepository) List(ctx context.Context, p LoginLogListParams) ([]model.LoginLog, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.LoginLog{})
	if p.UsernameExact != "" {
		q = q.Where("username = ?", p.UsernameExact)
	} else if p.Username != "" {
		q = q.Where("username LIKE ?", "%"+p.Username+"%")
	}
	if p.Status != nil {
		q = q.Where("status = ?", *p.Status)
	}
	if p.Source != "" {
		q = q.Where("source = ?", p.Source)
	}

	var list []model.LoginLog
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

func (r *LoginLogRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.LoginLog{}, id).Error
}

func (r *LoginLogRepository) DeleteByIDs(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.LoginLog{}).Error
}

// DeleteOlderThan 分批删除 cutoff 之前的日志并返回总数；每批最多 1000 行，避免长事务锁表。
func (r *LoginLogRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	const batch = 1000
	var total int64
	for {
		res := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Limit(batch).Delete(&model.LoginLog{})
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < batch {
			return total, nil
		}
	}
}
