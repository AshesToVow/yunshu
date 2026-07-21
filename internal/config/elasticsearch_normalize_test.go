package config

import "testing"

func TestElasticsearchNormalizedMigratesLegacyPattern(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", "yunshu-agent-*"},
		{"yunshu-logs-*", "yunshu-agent-*"},
		{"yunshu-logs", "yunshu-agent-*"},
		{"*", "yunshu-agent-*"},
		{"yunshu-agent-*", "yunshu-agent-*"},
		{"custom-*", "custom-*"},
	}
	for _, tc := range cases {
		got := ElasticsearchConfig{IndexPattern: tc.in}.Normalized().IndexPattern
		if got != tc.want {
			t.Fatalf("IndexPattern %q => %q, want %q", tc.in, got, tc.want)
		}
	}
}
