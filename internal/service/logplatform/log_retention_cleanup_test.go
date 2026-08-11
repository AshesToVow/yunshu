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
	hasAgent, hasLogs, hasK8s := false, false, false
	for _, p := range got {
		if p == GlobalAgentIndexPattern() {
			hasAgent = true
		}
		if p == "yunshu-logs-*" {
			hasLogs = true
		}
		if p == GlobalK8sIndexPattern() {
			hasK8s = true
		}
	}
	if !hasAgent || !hasLogs {
		t.Fatalf("got %v", got)
	}
	if !hasK8s {
		t.Fatalf("legacy default scope should also cover k8s, got %v", got)
	}
}

func TestResolveCleanupPatternsDefaultIncludesK8s(t *testing.T) {
	t.Parallel()
	got := resolveCleanupPatterns(model.LogRetentionPolicy{})
	hasAgent, hasK8s := false, false
	for _, p := range got {
		if p == GlobalAgentIndexPattern() {
			hasAgent = true
		}
		if p == GlobalK8sIndexPattern() {
			hasK8s = true
		}
	}
	if !hasAgent || !hasK8s {
		t.Fatalf("empty pattern should cover agent+k8s, got %v", got)
	}
}

func TestResolveCleanupPatternsServerScoped(t *testing.T) {
	t.Parallel()
	got := resolveCleanupPatterns(model.LogRetentionPolicy{ServerID: 7})
	if len(got) != 1 || got[0] != AgentIndexPatternByServerID(7) {
		t.Fatalf("server scope should only target agent index, got %v", got)
	}
}

func TestIsPlatformManagedIndex(t *testing.T) {
	t.Parallel()
	if !isPlatformManagedIndex("yunshu-agent-10-10-10-4-2026.07.20", "yunshu-logs-*") {
		t.Fatal("agent index should be manageable even with legacy pattern")
	}
	if !isPlatformManagedIndex("yunshu-k8s-3-2026.08.11", "yunshu-agent-*") {
		t.Fatal("k8s index should be manageable")
	}
	if isPlatformManagedIndex(".kibana", "yunshu-agent-*") {
		t.Fatal("system index must not be manageable")
	}
	if isPlatformManagedIndex("other-app-2026.07.20", "*") {
		t.Fatal("bare * must not allow arbitrary indices")
	}
}
