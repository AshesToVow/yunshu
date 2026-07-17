package logplatform

import "testing"

func TestAgentIndexSink(t *testing.T) {
	got := AgentIndexSink(7)
	want := "yunshu-agent-7-${+YYYY.MM.DD}"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolveSearchIndices(t *testing.T) {
	sid := uint(3)
	if got := ResolveSearchIndices(&sid, nil); got != "yunshu-agent-3-*" {
		t.Fatalf("single: %q", got)
	}
	got := ResolveSearchIndices(nil, []uint{1, 2})
	if got != "yunshu-agent-1-*,yunshu-agent-2-*" {
		t.Fatalf("multi: %q", got)
	}
	many := make([]uint, maxSearchIndexServers+1)
	for i := range many {
		many[i] = uint(i + 1)
	}
	if got := ResolveSearchIndices(nil, many); got != GlobalAgentIndexPattern() {
		t.Fatalf("overflow should fallback: %q", got)
	}
}
