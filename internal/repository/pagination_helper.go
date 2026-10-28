package repository

import (
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

func listWithPagination[T any](query *gorm.DB, page, pageSize int, order string, out *[]T) (int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	page, pageSize = pagination.Normalize(page, pageSize)
	offset := (page - 1) * pageSize
	if err := query.Order(order).Offset(offset).Limit(pageSize).Find(out).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// listWithPaginationRaw 仅供导出等受信内部路径使用：pageSize 由服务层按导出上限给定，
// 不做 MaxPageSize 钳制；列表 API 必须继续走 listWithPagination。
func listWithPaginationRaw[T any](query *gorm.DB, page, pageSize int, order string, out *[]T) (int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	if page <= 0 {
		page = pagination.DefaultPage
	}
	if pageSize <= 0 {
		pageSize = pagination.MaxPageSize
	}
	offset := (page - 1) * pageSize
	if err := query.Order(order).Offset(offset).Limit(pageSize).Find(out).Error; err != nil {
		return 0, err
	}
	return total, nil
}
