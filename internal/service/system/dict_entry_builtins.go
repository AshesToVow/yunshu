package system

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

// ensureBuiltins 负责内置字典的历史收敛与幂等 seed 编排。
// 具体种子数据按域拆分到 dict_seed_*.go，本文件只保留流程与清理逻辑。
func (s *DictEntryService) ensureBuiltins(ctx context.Context) {
	// 每次进入字典服务都先做一次历史去重，避免依赖 initOnce 触发时机。
	// 这样即使服务已运行较久、或历史版本已产生重复，也能自动收敛。
	_ = s.repo.CleanupDuplicateTypeLabel(ctx)
	_ = s.repo.CleanupDuplicateTypeValue(ctx)

	s.initOnce.Do(func() {
		s.normalizeLegacyDictData(ctx)
		s.seedBuiltinDictEntries(ctx)
		s.cleanupLegacyDictEntries(ctx)
	})

	// 内置种子处理后再做一次去重，兜底并发/历史脏数据场景。
	_ = s.repo.CleanupDuplicateTypeLabel(ctx)
	_ = s.repo.CleanupDuplicateTypeValue(ctx)
}

// normalizeLegacyDictData 在 seed 之前做历史数据收敛：大小写归一、旧类型迁移与废弃类型删除。
func (s *DictEntryService) normalizeLegacyDictData(ctx context.Context) {
	// 历史收敛：dict_type 统一小写，与代码侧 cicd_* / alert_* 读取一致。
	_ = s.repo.NormalizeDictTypeCase(ctx)

	// 历史收敛：静默 matcher key 已统一到 alert_promql_label_key，先迁移再删除旧类型。
	s.migrateAlertSilenceMatcherKeys(ctx)

	// 不再使用数据字典维护 HTTP 方法；清理历史遗留行，避免与「仅保留敏感配置类字典」目标冲突。
	_ = s.repo.DeleteByTypes(ctx, []string{"http_action"})
	// 旧 gRPC / log-agent 相关字典已废弃。
	_ = s.repo.DeleteByTypes(ctx, []string{"agent_platform_url", "log_agent_health_status"})
}

// seedBuiltinDictEntries 按域汇总内置种子并幂等写入。
// 单值型字典（dictSingletonTypes）按「类型」判重，其余按「类型 + 标签」判重，
// 避免值被人工改动后再次 seed 产生同标签重复项。
func (s *DictEntryService) seedBuiltinDictEntries(ctx context.Context) {
	singletonTypes := dictSingletonTypes()

	for _, item := range builtinDictSeeds() {
		var (
			exists bool
			err    error
		)
		if _, ok := singletonTypes[item.DictType]; ok {
			exists, err = s.repo.ExistsByType(ctx, item.DictType, 0)
		} else {
			exists, err = s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, 0)
		}
		if err != nil || exists {
			continue
		}
		_ = s.repo.Create(ctx, &model.DictEntry{
			DictType: strings.TrimSpace(item.DictType),
			Label:    strings.TrimSpace(item.Label),
			Value:    strings.TrimSpace(item.Value),
			Sort:     dictEntrySort(item.Sort),
			Status:   item.Status,
			Remark:   strings.TrimSpace(item.Remark),
		})
	}
}

// cleanupLegacyDictEntries 清理历史示例条目、重命名旧标签并迁移旧默认值。
func (s *DictEntryService) cleanupLegacyDictEntries(ctx context.Context) {
	// 清理早期内置的长 YAML 示例，避免与占位条目重复。
	_ = s.repo.DeleteByTypeAndLabel(ctx, "k8s_kubeconfig_template", "单集群 Bearer 模板")
	// 清理历史 SMTP 示例条目，避免每次 seed 后出现“示例邮箱/名称”重复项。
	_ = s.repo.DeleteByTypeAndValue(ctx, "mail_username", "root@example.com")
	_ = s.repo.DeleteByTypeAndValue(ctx, "mail_from_email", "root@example.com")
	_ = s.repo.DeleteByTypeAndValue(ctx, "mail_from_name", "YunShu")

	// 应用类型「微服务」已更名为「容器化服务」
	if items, err := s.repo.ListByType(ctx, "cicd_pipeline_type"); err == nil {
		for _, it := range items {
			if strings.TrimSpace(it.Value) == "microservice" && strings.TrimSpace(it.Label) == "微服务" {
				it.Label = "容器化服务"
				_ = s.repo.Update(ctx, &it)
			}
		}
	}

	// 日志索引/Topic 历史默认值迁移
	s.migrateLegacyLogDictDefaults(ctx)
	// 旧类型清理：收敛为单一 alert_promql_label_key 后，不再保留旧 dict_type。
	_ = s.repo.DeleteByTypes(ctx, []string{dictTypeAlertSilenceMatcherName})
}
