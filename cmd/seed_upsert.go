package cmd

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// upsertByKey 按唯一键查找记录：不存在则 Create（可选 prepareCreate），存在则 merge 后 Save。
func upsertByKey[T any](
	ctx context.Context,
	db *gorm.DB,
	target *T,
	query func(db *gorm.DB) *gorm.DB,
	merge func(existing, incoming *T),
	prepareCreate func(incoming *T) error,
) error {
	var existing T
	err := query(db.WithContext(ctx)).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if prepareCreate != nil {
			if err := prepareCreate(target); err != nil {
				return err
			}
		}
		return db.WithContext(ctx).Create(target).Error
	}
	if err != nil {
		return err
	}
	merge(&existing, target)
	if err := db.WithContext(ctx).Save(&existing).Error; err != nil {
		return err
	}
	*target = existing
	return nil
}
