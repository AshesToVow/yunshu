package config

import "strings"

// KafkaConfig 日志中转（Loggie → Kafka → Yunshu 消费写 ES）。
// enabled=false 时 Agent 直写 Elasticsearch。
// Topic / TopicPrefix：每个 Agent 独立 Topic，命名为 {prefix}-{server_id}（默认 yunshu-agent-7）。
type KafkaConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Brokers []string `mapstructure:"brokers"`
	// TopicPrefix Agent Topic 前缀；兼容旧键 topic
	TopicPrefix string `mapstructure:"topic_prefix"`
	Topic       string `mapstructure:"topic"` // 兼容旧配置，等同 topic_prefix
	// ConsumerGroup 必填：Yunshu 消费写 ES 使用的消费者组
	ConsumerGroup string `mapstructure:"consumer_group"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	// SASLMechanism：plain / scram-sha-256 / scram-sha-512；空且有用户名时默认 plain
	SASLMechanism string `mapstructure:"sasl_mechanism"`
	// BatchSize 消费批量写入 ES 的条数
	BatchSize int `mapstructure:"batch_size"`
	// TopicPartitions 自动建 Topic 时的分区数
	TopicPartitions int `mapstructure:"topic_partitions"`
	Workers         int `mapstructure:"workers"`
}

func (c KafkaConfig) Normalized() KafkaConfig {
	out := c
	prefix := strings.Trim(strings.TrimSpace(out.TopicPrefix), "-")
	if prefix == "" {
		prefix = strings.Trim(strings.TrimSpace(out.Topic), "-")
	}
	if prefix == "" || prefix == "yunshu-logs" {
		// 旧默认共用 topic 名视为前缀迁移
		prefix = "yunshu-agent"
	}
	out.TopicPrefix = prefix
	out.Topic = prefix
	if strings.TrimSpace(out.ConsumerGroup) == "" {
		out.ConsumerGroup = "yunshu-log-es"
	}
	if out.BatchSize <= 0 {
		out.BatchSize = 200
	}
	if out.Workers <= 0 {
		out.Workers = 1
	}
	if out.TopicPartitions <= 0 {
		out.TopicPartitions = 3
	}
	mech := strings.ToLower(strings.TrimSpace(out.SASLMechanism))
	switch mech {
	case "plain", "scram-sha-256", "scram-sha-512", "none", "":
		out.SASLMechanism = mech
	default:
		out.SASLMechanism = "plain"
	}
	if out.SASLMechanism == "" && strings.TrimSpace(out.Username) != "" {
		out.SASLMechanism = "plain"
	}
	brokers := make([]string, 0, len(out.Brokers))
	for _, b := range out.Brokers {
		b = strings.TrimSpace(b)
		if b != "" {
			brokers = append(brokers, b)
		}
	}
	out.Brokers = brokers
	return out
}

// SinkViaKafka 是否走 Kafka 中转（开关 + broker + 消费组）。
func (c KafkaConfig) SinkViaKafka() bool {
	n := c.Normalized()
	return n.Enabled && len(n.Brokers) > 0 && strings.TrimSpace(n.ConsumerGroup) != ""
}
