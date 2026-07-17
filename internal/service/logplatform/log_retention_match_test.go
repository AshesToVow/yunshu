package logplatform

import "testing"

func TestMatchIndexPattern(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"yunshu-agent-7-2026.07.13", "yunshu-agent-*", true},
		{"yunshu-agent-7-2026.07.13", "yunshu-logs-*", false},
		{"app-logs", "*", true},
		{"exact", "exact", true},
		{"exact", "other", false},
		{"a-b-c", "a-*-c", true},
		{"a-b-d", "a-*-c", false},
	}
	for _, tc := range cases {
		if got := matchIndexPattern(tc.name, tc.pattern); got != tc.want {
			t.Fatalf("matchIndexPattern(%q,%q)=%v want %v", tc.name, tc.pattern, got, tc.want)
		}
	}
}
