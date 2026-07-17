package logplatform

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	transport := &kafka.Transport{}
	if dialer.SASLMechanism != nil {
		transport.SASL = dialer.SASLMechanism
	}
	return &kafka.Client{
		Addr:      kafka.TCP(cfg.Brokers...),
		Transport: transport,
	}, nil
}

// EnsureAgentKafkaTopic 确保该 Agent 的 Topic 存在（幂等）。
func EnsureAgentKafkaTopic(ctx context.Context, cfg config.KafkaConfig, serverID uint) (string, error) {
	cfg = cfg.Normalized()
	topic := AgentKafkaTopic(serverID, cfg.TopicPrefix)
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

// listAgentKafkaTopics 列出前缀匹配的 Agent Topic（yunshu-agent-<id>）。
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
	seen := map[string]struct{}{}
	for _, p := range parts {
		name := strings.TrimSpace(p.Topic)
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		if _, ok := ParseServerIDFromAgentKafkaTopic(name, cfg.TopicPrefix); !ok {
			continue
		}
		seen[name] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out, nil
}
