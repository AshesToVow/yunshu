package dictconfig

import (
	"context"
	"errors"
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

// UpsertEnabledDictValue 按 dict_type 写入（或更新）一条启用状态的字典值。
// 若已存在多条，更新 sort 最小、id 最大的那条（与 FetchEnabledDictValue 选取规则一致）。
func UpsertEnabledDictValue(ctx context.Context, db *gorm.DB, dictType, label, value, remark string) error {
	dictType = strings.TrimSpace(dictType)
	if db == nil || dictType == "" {
		return errors.New("dict upsert: invalid args")
	}
	if strings.TrimSpace(label) == "" {
		label = dictType
	}
	var row model.DictEntry
	err := db.WithContext(ctx).
		Where("dict_type = ?", dictType).
		Order("sort ASC, id DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.WithContext(ctx).Create(&model.DictEntry{
			DictType: dictType,
			Label:    label,
			Value:    value,
			Sort:     1,
			Status:   1,
			Remark:   remark,
		}).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{
		"value":  value,
		"status": 1,
		"label":  label,
	}
	if strings.TrimSpace(remark) != "" {
		updates["remark"] = remark
	}
	return db.WithContext(ctx).Model(&row).Updates(updates).Error
}

// 包内别名，供 minio/cicd 等 resolver 复用，避免重复实现。
func fetchEnabledDictValue(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	return FetchEnabledDictValue(ctx, db, dictType)
}

func fetchEnabledDictValueNonEmpty(ctx context.Context, db *gorm.DB, dictType string) (string, bool) {
	return FetchEnabledDictValueNonEmpty(ctx, db, dictType)
}
