package dictconfig

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// KafkaDictTypes 数据字典覆盖 kafka.* 的 dict_type。
type KafkaDictTypes struct {
	Enabled       string
	Brokers       string
	Topic         string
	K8sTopic      string
	ConsumerGroup string
	Username      string
	Password      string
	SASLMechanism string
	BatchSize     string
	Workers       string
}

func DefaultKafkaDictTypes() KafkaDictTypes {
	return KafkaDictTypes{
		Enabled:       "kafka_enabled",
		Brokers:       "kafka_brokers",
		Topic:         "kafka_topic_prefix",
		K8sTopic:      "kafka_k8s_topic_prefix",
		ConsumerGroup: "kafka_consumer_group",
		Username:      "kafka_username",
		Password:      "kafka_password",
		SASLMechanism: "kafka_sasl_mechanism",
		BatchSize:     "kafka_batch_size",
		Workers:       "kafka_workers",
	}
}

// ResolveKafkaConfig 字典优先合并 kafka 配置。
// brokers 支持 JSON 数组或逗号/分号/换行分隔（单节点与集群均可）。
// Topic 前缀：优先 kafka_topic_prefix，兼容旧 kafka_topic。
func ResolveKafkaConfig(ctx context.Context, db *gorm.DB, base config.KafkaConfig) config.KafkaConfig {
	if db == nil {
		return base.Normalized()
	}
	types := DefaultKafkaDictTypes()
	out := base
	if v, ok := FetchEnabledDictValue(ctx, db, types.Enabled); ok {
		out.Enabled = strings.EqualFold(strings.TrimSpace(v), "true") || v == "1"
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.Brokers); ok {
		if brokers := parseESAddresses(v); len(brokers) > 0 {
			out.Brokers = brokers
		}
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.Topic); ok {
		out.TopicPrefix = strings.TrimSpace(v)
	} else if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, "kafka_topic"); ok {
		out.TopicPrefix = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.K8sTopic); ok {
		out.K8sTopicPrefix = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.ConsumerGroup); ok {
		out.ConsumerGroup = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.Username); ok {
		out.Username = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.Password); ok {
		out.Password = v
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.SASLMechanism); ok {
		out.SASLMechanism = strings.TrimSpace(v)
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.BatchSize); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			out.BatchSize = n
		}
	}
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, types.Workers); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			out.Workers = n
		}
	}
	return out.Normalized()
}
