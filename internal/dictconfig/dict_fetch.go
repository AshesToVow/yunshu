package dictconfig

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// FetchEnabledDictValue 读取某 dict_type 下 status=1 的最高优先级条目（sort ASC, id DESC）。
// 允许空字符串作为显式覆盖。
func FetchEnabledDictValue(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	if db == nil || strings.TrimSpace(dictType) == "" {
		return "", false
	}
	var row model.DictEntry
	err := db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND status = 1", strings.TrimSpace(dictType)).
		Order("sort ASC, id DESC").
		Limit(1).
		First(&row).Error
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(row.Value), true
}

// FetchEnabledDictValueNonEmpty 同 FetchEnabledDictValue，但忽略空值。
func FetchEnabledDictValueNonEmpty(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	v, ok := FetchEnabledDictValue(ctx, db, dictType)
	if !ok || strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

// 包内别名，供 minio/cicd 等 resolver 复用，避免重复实现。
func fetchEnabledDictValue(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	return FetchEnabledDictValue(ctx, db, dictType)
}

func fetchEnabledDictValueNonEmpty(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	return FetchEnabledDictValueNonEmpty(ctx, db, dictType)
}
