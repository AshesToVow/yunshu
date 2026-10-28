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

// topicEnsureDayOffsets Agent 与后端可能不在同一时区：Loggie 用本地日期渲染
// ${+YYYY.MM.DD}，后端按 UTC 计算，两者最多相差一天。因此前后各预热一天，
// 避免本地跨过 0 点后 sink 写入一个 broker 上不存在的 Topic。
var topicEnsureDayOffsets = []int{-1, 0, 1}

// EnsureAgentKafkaTopic 确保该 Agent 的 Topic 存在（幂等），返回 UTC 当日 Topic 名。
func EnsureAgentKafkaTopic(ctx context.Context, cfg config.KafkaConfig, serverHost string) (string, error) {
	cfg = cfg.Normalized()
	now := time.Now().UTC()
	topic := AgentKafkaTopicForDay(serverHost, cfg.TopicPrefix, now)
	if err := ensureAgentTopicWindow(ctx, cfg, serverHost, now); err != nil {
		return topic, err
	}
	return topic, nil
}

// EnsureK8sKafkaTopic 确保集群采集 Topic 存在（含项目隔离）。
func EnsureK8sKafkaTopic(ctx context.Context, cfg config.KafkaConfig, clusterID, projectID uint) (string, error) {
	cfg = cfg.Normalized()
	now := time.Now().UTC()
	topic := K8sKafkaTopicForDay(clusterID, projectID, cfg.K8sTopicPrefix, now)
	if err := ensureK8sTopicWindow(ctx, cfg, clusterID, projectID, now); err != nil {
		return topic, err
	}
	return topic, nil
}

func ensureAgentTopicWindow(ctx context.Context, cfg config.KafkaConfig, serverHost string, now time.Time) error {
	topics := make([]string, 0, len(topicEnsureDayOffsets))
	for _, off := range topicEnsureDayOffsets {
		topics = append(topics, AgentKafkaTopicForDay(serverHost, cfg.TopicPrefix, now.AddDate(0, 0, off)))
	}
	return ensureKafkaTopics(ctx, cfg, topics)
}

func ensureK8sTopicWindow(ctx context.Context, cfg config.KafkaConfig, clusterID, projectID uint, now time.Time) error {
	topics := make([]string, 0, len(topicEnsureDayOffsets))
	for _, off := range topicEnsureDayOffsets {
		topics = append(topics, K8sKafkaTopicForDay(clusterID, projectID, cfg.K8sTopicPrefix, now.AddDate(0, 0, off)))
	}
	return ensureKafkaTopics(ctx, cfg, topics)
}

// desiredDailyTopics 按已存在的 Agent/K8s Topic 推导出应存在但缺失的日期 Topic。
// 部署时只建当日 Topic，Agent 连续运行跨天后就没人再建；broker 关闭
// auto.create.topics.enable 时 sink 会持续报 Unknown Topic Or Partition，日志全部写不进去。
func desiredDailyTopics(existing []string, cfg config.KafkaConfig, now time.Time) []string {
	cfg = cfg.Normalized()
	present := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		present[strings.TrimSpace(t)] = struct{}{}
	}
	out := make([]string, 0, len(existing)*len(topicEnsureDayOffsets))
	seen := map[string]struct{}{}
	for _, t := range existing {
		var build func(time.Time) string
		if key, ok := ParseHostKeyFromAgentName(t, cfg.TopicPrefix); ok {
			build = func(day time.Time) string { return AgentKafkaTopicForDay(key, cfg.TopicPrefix, day) }
		} else if clusterID, projectID, ok := ParseK8sKafkaTopicMeta(t, cfg.K8sTopicPrefix); ok {
			build = func(day time.Time) string {
				return K8sKafkaTopicForDay(clusterID, projectID, cfg.K8sTopicPrefix, day)
			}
		} else {
			continue
		}
		for _, off := range topicEnsureDayOffsets {
			want := build(now.AddDate(0, 0, off))
			if _, ok := present[want]; ok {
				continue
			}
			if _, dup := seen[want]; dup {
				continue
			}
			seen[want] = struct{}{}
			out = append(out, want)
		}
	}
	sort.Strings(out)
	return out
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
	if !IsAgentKafkaTopic(topic, cfg.TopicPrefix) && !IsK8sKafkaTopic(topic, cfg.K8sTopicPrefix) {
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
	k8sPrefix := k8sNamePrefix(cfg.K8sTopicPrefix) + "-"
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
		if strings.HasPrefix(name, k8sPrefix) && IsK8sKafkaTopic(name, cfg.K8sTopicPrefix) {
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
