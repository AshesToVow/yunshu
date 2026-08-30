package system

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

func (s *DictEntryService) migrateLegacyLogDictDefaults(ctx context.Context) {
	rewrite := func(dictType, from, to string) {
		items, err := s.repo.ListByType(ctx, dictType)
		if err != nil {
			return
		}
		for i := range items {
			if strings.TrimSpace(items[i].Value) != from {
				continue
			}
			items[i].Value = to
			_ = s.repo.Update(ctx, &items[i])
		}
	}
	rewrite("elasticsearch_index_pattern", "yunshu-logs-*", "yunshu-agent-*")
	rewrite("elasticsearch_index_pattern", "yunshu-logs", "yunshu-agent-*")
	rewrite("kafka_topic", "yunshu-logs", "yunshu-agent")
	rewrite("kafka_topic_prefix", "yunshu-logs", "yunshu-agent")
}

func (s *DictEntryService) migrateAlertSilenceMatcherKeys(ctx context.Context) {
	oldList, err := s.repo.ListByType(ctx, dictTypeAlertSilenceMatcherName)
	if err != nil || len(oldList) == 0 {
		return
	}
	for _, it := range oldList {
		targetLabel := strings.TrimSpace(it.Label)
		targetValue := strings.TrimSpace(it.Value)
		if targetLabel == "" || targetValue == "" {
			continue
		}
		existsByValue, err := s.repo.ExistsByTypeValue(ctx, dictTypeAlertPromQLLabelKey, targetValue, 0)
		if err == nil && existsByValue {
			continue
		}
		existsByLabel, err := s.repo.ExistsByTypeLabel(ctx, dictTypeAlertPromQLLabelKey, targetLabel, 0)
		if err == nil && existsByLabel {
			continue
		}
		_ = s.repo.Create(ctx, &model.DictEntry{
			DictType: dictTypeAlertPromQLLabelKey,
			Label:    targetLabel,
			Value:    targetValue,
			Sort:     it.Sort,
			Status:   it.Status,
			Remark:   strings.TrimSpace(it.Remark),
		})
	}
}
