package config

import "strings"

// KafkaConfig 日志中转（Loggie → Kafka → Yunshu 消费写 ES）。
// enabled=false 时 Agent 直写 Elasticsearch。
type KafkaConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	Brokers       []string `mapstructure:"brokers"`
	Topic         string   `mapstructure:"topic"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	Username      string   `mapstructure:"username"`
	Password      string   `mapstructure:"password"`
	// SASLMechanism：plain / scram-sha-256 / scram-sha-512；空且有用户名时默认 plain
	SASLMechanism string `mapstructure:"sasl_mechanism"`
	// BatchSize 消费批量写入 ES 的条数
	BatchSize int `mapstructure:"batch_size"`
	// Workers 并行消费协程数（按分区分配由 kafka-go 处理时仍作批大小参考）
	Workers int `mapstructure:"workers"`
}

func (c KafkaConfig) Normalized() KafkaConfig {
	out := c
	if strings.TrimSpace(out.Topic) == "" {
		out.Topic = "yunshu-logs"
	}
	if strings.TrimSpace(out.ConsumerGroup) == "" {
		out.ConsumerGroup = "yunshu-log-es"
	}
	if out.BatchSize <= 0 {
		out.BatchSize = 200
	}
	if out.Workers <= 0 {
		out.Workers = 1
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

// SinkViaKafka 是否走 Kafka 中转（开关开启且有可用 broker）。
func (c KafkaConfig) SinkViaKafka() bool {
	n := c.Normalized()
	return n.Enabled && len(n.Brokers) > 0 && strings.TrimSpace(n.Topic) != ""
}
