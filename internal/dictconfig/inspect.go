package dictconfig

import (
	"context"
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// InspectReportDictTypes 巡检报告相关数据字典（dict_type=inspect_report，label 为配置键）。
type InspectReportDictTypes struct {
	Type         string
	RequireMinIO string
}

func DefaultInspectReportDictTypes() InspectReportDictTypes {
	return InspectReportDictTypes{
		Type:         "inspect_report",
		RequireMinIO: "require_minio",
	}
}

// InspectReportConfig 巡检报告运行时配置。
type InspectReportConfig struct {
	RequireMinIO bool
}

// ResolveInspectReportConfig 从数据字典 inspect_report 读取配置；require_minio 默认 false。
func ResolveInspectReportConfig(ctx context.Context, db *gorm.DB, types InspectReportDictTypes) InspectReportConfig {
	cfg := InspectReportConfig{}
	if db == nil {
		return cfg
	}
	dictType := strings.TrimSpace(types.Type)
	if dictType == "" {
		dictType = "inspect_report"
	}
	label := strings.TrimSpace(types.RequireMinIO)
	if label == "" {
		label = "require_minio"
	}
	if v, ok := fetchDictValueByTypeLabel(ctx, db, dictType, label); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.RequireMinIO = bv
		}
	}
	return cfg
}

func fetchDictValueByTypeLabel(ctx context.Context, db *gorm.DB, dictType, label string) (string, bool) {
	if db == nil || strings.TrimSpace(dictType) == "" || strings.TrimSpace(label) == "" {
		return "", false
	}
	var row model.DictEntry
	err := db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND label = ? AND status = 1", strings.TrimSpace(dictType), strings.TrimSpace(label)).
		Order("sort ASC, id DESC").
		Limit(1).
		First(&row).Error
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(row.Value), true
}
