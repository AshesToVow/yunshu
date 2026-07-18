package logplatform

import (
	"testing"
	"time"
)

func TestSanitizeHostForName(t *testing.T) {
	if got := SanitizeHostForName("10.10.10.5"); got != "10-10-10-5" {
		t.Fatalf("got %s", got)
	}
}

func TestAgentIndexSink(t *testing.T) {
	got := AgentIndexSink("10.10.10.5")
	want := "yunshu-agent-10-10-10-5-${+YYYY.MM.DD}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAgentKafkaTopicForDay(t *testing.T) {
	day := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	got := AgentKafkaTopicForDay("10.10.10.5", "yunshu-agent", day)
	want := "yunshu-agent-10-10-10-5-2026.07.17"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsAgentKafkaTopic(t *testing.T) {
	if !IsAgentKafkaTopic("yunshu-agent-8", "yunshu-agent") {
		t.Fatal("legacy id topic")
	}
	if !IsAgentKafkaTopic("yunshu-agent-10-10-10-5-2026.07.17", "yunshu-agent") {
		t.Fatal("ip+date topic")
	}
	if IsAgentKafkaTopic("other-topic", "yunshu-agent") {
		t.Fatal("should reject")
	}
}

func TestResolveSearchIndices(t *testing.T) {
	sid := uint(3)
	if got := ResolveSearchIndices(&sid, nil); got != "yunshu-agent-3-*" {
		t.Fatalf("got %s", got)
	}
	got := ResolveSearchIndices(nil, []uint{1, 2})
	if got != "yunshu-agent-1-*,yunshu-agent-2-*" {
		t.Fatalf("got %s", got)
	}
	many := make([]uint, maxSearchIndexServers+1)
	for i := range many {
		many[i] = uint(i + 1)
	}
	if got := ResolveSearchIndices(nil, many); got != GlobalAgentIndexPattern() {
		t.Fatalf("expected global, got %s", got)
	}
}

func TestResolveSearchIndicesByHosts(t *testing.T) {
	got := ResolveSearchIndicesByHosts([]string{"10.10.10.5"}, []uint{8})
	if !containsAll(got, "yunshu-agent-10-10-10-5-*", "yunshu-agent-8-*") {
		t.Fatalf("got %s", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsSub(s, p) {
			return false
		}
	}
	return true
}

func containsSub(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
