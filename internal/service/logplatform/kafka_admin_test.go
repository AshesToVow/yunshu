package logplatform

import (
	"reflect"
	"testing"
	"time"

	"yunshu/internal/config"
)

// 回归：Loggie 按本地日期渲染 Topic，后端按 UTC 建 Topic 且只在部署时建一次。
// 本地跨过 0 点后 sink 会写入一个 broker 上不存在的 Topic（Unknown Topic Or Partition）。
func TestDesiredDailyTopics_CoversLocalDateAheadOfUTC(t *testing.T) {
	t.Parallel()
	cfg := config.KafkaConfig{TopicPrefix: "yunshu-agent", K8sTopicPrefix: "yunshu-k8s"}
	// UTC 2026-10-28 16:05 = 本地(+0800) 2026-10-29 00:05
	now := time.Date(2026, 10, 28, 16, 5, 0, 0, time.UTC)
	existing := []string{"yunshu-agent-10-10-10-5-2026.10.28"}

	got := desiredDailyTopics(existing, cfg, now)
	want := []string{
		"yunshu-agent-10-10-10-5-2026.10.27",
		"yunshu-agent-10-10-10-5-2026.10.29",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDesiredDailyTopics_K8sAndLegacyTopics(t *testing.T) {
	t.Parallel()
	cfg := config.KafkaConfig{TopicPrefix: "yunshu-agent", K8sTopicPrefix: "yunshu-k8s"}
	now := time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC)
	existing := []string{
		"yunshu-k8s-4-p1-2026.10.29",
		"yunshu-agent-8", // 旧版按 server_id 命名，无日期
		"unrelated-topic",
	}

	got := desiredDailyTopics(existing, cfg, now)
	want := []string{
		"yunshu-k8s-4-p1-2026.10.28",
		"yunshu-k8s-4-p1-2026.10.30",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDesiredDailyTopics_AllPresent(t *testing.T) {
	t.Parallel()
	cfg := config.KafkaConfig{TopicPrefix: "yunshu-agent", K8sTopicPrefix: "yunshu-k8s"}
	now := time.Date(2026, 10, 29, 3, 0, 0, 0, time.UTC)
	existing := []string{
		"yunshu-agent-10-10-10-5-2026.10.28",
		"yunshu-agent-10-10-10-5-2026.10.29",
		"yunshu-agent-10-10-10-5-2026.10.30",
	}
	if got := desiredDailyTopics(existing, cfg, now); len(got) != 0 {
		t.Fatalf("expected nothing to create, got %v", got)
	}
}
