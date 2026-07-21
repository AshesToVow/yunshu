package logplatform

import (
	"testing"

	"yunshu/internal/model"
)

func TestResolveCleanupPatternsLegacy(t *testing.T) {
	t.Parallel()
	got := resolveCleanupPatterns(model.LogRetentionPolicy{IndexPattern: "yunshu-logs-*"})
	if len(got) < 2 {
		t.Fatalf("want agent + logs patterns, got %v", got)
	}
	hasAgent, hasLogs := false, false
	for _, p := range got {
		if p == GlobalAgentIndexPattern() {
			hasAgent = true
		}
		if p == "yunshu-logs-*" {
			hasLogs = true
		}
	}
	if !hasAgent || !hasLogs {
		t.Fatalf("got %v", got)
	}
}

func TestIsPlatformManagedIndex(t *testing.T) {
	t.Parallel()
	if !isPlatformManagedIndex("yunshu-agent-10-10-10-4-2026.07.20", "yunshu-logs-*") {
		t.Fatal("agent index should be manageable even with legacy pattern")
	}
	if isPlatformManagedIndex(".kibana", "yunshu-agent-*") {
		t.Fatal("system index must not be manageable")
	}
	if isPlatformManagedIndex("other-app-2026.07.20", "*") {
		t.Fatal("bare * must not allow arbitrary indices")
	}
}
