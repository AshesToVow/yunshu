package logplatform

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/config"

	"github.com/segmentio/kafka-go"
)

func kafkaClient(cfg config.KafkaConfig) (*kafka.Client, error) {
	cfg = cfg.Normalized()
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers empty")
	}
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		return nil, err
	}
	return &kafka.Client{
		Addr:      kafka.TCP(cfg.Brokers...),
		Timeout:   10 * time.Second,
		Transport: kafkaTransport(dialer),
	}, nil
}

func kafkaTransport(dialer *kafka.Dialer) *kafka.Transport {
	t := &kafka.Transport{}
	if dialer != nil && dialer.SASLMechanism != nil {
		t.SASL = dialer.SASLMechanism
	}
	return t
}

// EnsureAgentKafkaTopic 确保该 Agent 当日 Topic 存在（幂等）：yunshu-agent-{ip}-YYYY.MM.DD
func EnsureAgentKafkaTopic(ctx context.Context, cfg config.KafkaConfig, serverHost string) (string, error) {
	cfg = cfg.Normalized()
	topic := AgentKafkaTopicForDay(serverHost, cfg.TopicPrefix, time.Now().UTC())
	if err := ensureKafkaTopics(ctx, cfg, []string{topic}); err != nil {
		return topic, err
	}
	return topic, nil
}

// EnsureK8sKafkaTopic 确保集群采集当日 Topic 存在（含项目隔离）。
func EnsureK8sKafkaTopic(ctx context.Context, cfg config.KafkaConfig, clusterID, projectID uint) (string, error) {
	cfg = cfg.Normalized()
	topic := K8sKafkaTopicForDay(clusterID, projectID, defaultK8sIndexPrefix, time.Now().UTC())
	if err := ensureKafkaTopics(ctx, cfg, []string{topic}); err != nil {
		return topic, err
	}
	return topic, nil
}

func ensureKafkaTopics(ctx context.Context, cfg config.KafkaConfig, topics []string) error {
	cfg = cfg.Normalized()
	if len(topics) == 0 {
		return nil
	}
	client, err := kafkaClient(cfg)
	if err != nil {
		return err
	}
	reqs := make([]kafka.TopicConfig, 0, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		reqs = append(reqs, kafka.TopicConfig{
			Topic:             t,
			NumPartitions:     cfg.TopicPartitions,
			ReplicationFactor: 1,
		})
	}
	if len(reqs) == 0 {
		return nil
	}
	resp, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{Topics: reqs})
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	for topic, terr := range resp.Errors {
		if terr == nil {
			continue
		}
		msg := strings.ToLower(terr.Error())
		if terr == kafka.TopicAlreadyExists || strings.Contains(msg, "already exists") || strings.Contains(msg, "topic_already_exists") {
			continue
		}
		return fmt.Errorf("create topic %s: %w", topic, terr)
	}
	return nil
}

// DeleteAgentKafkaTopic 删除指定 Agent 或 K8s 平台 Topic。
func DeleteAgentKafkaTopic(ctx context.Context, cfg config.KafkaConfig, topic string) error {
	cfg = cfg.Normalized()
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("topic required")
	}
	if !IsAgentKafkaTopic(topic, cfg.TopicPrefix) && !IsK8sKafkaTopic(topic, defaultK8sIndexPrefix) {
		return fmt.Errorf("拒绝删除非平台 Topic: %s", topic)
	}
	client, err := kafkaClient(cfg)
	if err != nil {
		return err
	}
	resp, err := client.DeleteTopics(ctx, &kafka.DeleteTopicsRequest{Topics: []string{topic}})
	if err != nil {
		return err
	}
	if resp != nil {
		if terr, ok := resp.Errors[topic]; ok && terr != nil {
			msg := strings.ToLower(terr.Error())
			if strings.Contains(msg, "unknown topic") || strings.Contains(msg, "does not exist") {
				return nil
			}
			return fmt.Errorf("delete topic %s: %w", topic, terr)
		}
	}
	return nil
}

// listAgentKafkaTopics 列出前缀匹配的 Agent Topic（新 IP+日期 与旧 server_id）。
func listAgentKafkaTopics(ctx context.Context, cfg config.KafkaConfig) ([]string, error) {
	cfg = cfg.Normalized()
	dialer, err := kafkaDialer(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := dialer.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	parts, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}
	prefix := cfg.TopicPrefix + "-"
	k8sPrefix := defaultK8sIndexPrefix + "-"
	seen := map[string]struct{}{}
	for _, p := range parts {
		name := strings.TrimSpace(p.Topic)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, prefix) && IsAgentKafkaTopic(name, cfg.TopicPrefix) {
			seen[name] = struct{}{}
			continue
		}
		if strings.HasPrefix(name, k8sPrefix) && IsK8sKafkaTopic(name, defaultK8sIndexPrefix) {
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}
