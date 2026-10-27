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

// FetchEnabledDictValues 批量读取多个 dict_type 的启用值，仅一次 IN 查询。
// 每个类型取 sort ASC, id DESC 的首条，与 FetchEnabledDictValue 一致；
// 未配置的类型不出现在返回值中，显式空值映射为空字符串。
func FetchEnabledDictValues(ctx context.Context, db *gorm.DB, dictTypes []string) map[string]string {
	if db == nil || len(dictTypes) == 0 {
		return make(map[string]string, len(dictTypes))
	}
	types := normalizeDictTypes(dictTypes)
	if len(types) == 0 {
		return make(map[string]string)
	}
	var rows []model.DictEntry
	err := db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Select("dict_type", "value").
		Where("dict_type IN ? AND status = 1", types).
		Order("dict_type ASC, sort ASC, id DESC").
		Find(&rows).Error
	if err != nil {
		return make(map[string]string, len(types))
	}
	return pickFirstValuePerType(types, rows)
}

func normalizeDictTypes(dictTypes []string) []string {
	types := make([]string, 0, len(dictTypes))
	seen := make(map[string]struct{}, len(dictTypes))
	for _, t := range dictTypes {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		types = append(types, t)
	}
	return types
}

// pickFirstValuePerType 按查询返回顺序（sort ASC, id DESC）为每个类型取首条。
func pickFirstValuePerType(types []string, rows []model.DictEntry) map[string]string {
	values := make(map[string]string, len(types))
	byType := make(map[string]string, len(types))
	for _, row := range rows {
		key := strings.ToLower(row.DictType)
		if _, ok := byType[key]; ok {
			continue
		}
		byType[key] = strings.TrimSpace(row.Value)
	}
	// 以调用方传入的类型回填键名，规避 MySQL 大小写不敏感排序规则导致的行内大小写差异。
	for _, t := range types {
		if v, ok := byType[strings.ToLower(t)]; ok {
			values[t] = v
		}
	}
	return values
}

// dictSnapshot 一次查询得到的字典值快照，供 resolver 把 N 次查询收敛为 1 次。
type dictSnapshot struct {
	values map[string]string
}

func fetchDictSnapshot(ctx context.Context, db *gorm.DB, dictTypes ...string) dictSnapshot {
	return dictSnapshot{values: FetchEnabledDictValues(ctx, db, dictTypes)}
}

// get 对应 FetchEnabledDictValue（允许显式空值）。
func (s dictSnapshot) get(dictType string) (string, bool) {
	if s.values == nil {
		return "", false
	}
	v, ok := s.values[strings.TrimSpace(dictType)]
	return v, ok
}

// getNonEmpty 对应 FetchEnabledDictValueNonEmpty。
func (s dictSnapshot) getNonEmpty(dictType string) (string, bool) {
	v, ok := s.get(dictType)
	if !ok || v == "" {
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
