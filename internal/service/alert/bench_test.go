package alert

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/pkg/alertnotify"
)

func BenchmarkChannelMatchesAlert(b *testing.B) {
	settings := map[string]any{
		"matchLabels": map[string]any{"cluster": "prod"},
		"matchRegex":  map[string]any{"namespace": "^kube-"},
	}
	labels := map[string]string{"env": "prod", "team": "sre"}
	dims := alertnotify.Dims{Cluster: "prod", Namespace: "kube-system"}
	b.ResetTimer()
	for b.Loop() {
		channelMatchesAlert(settings, labels, dims)
	}
}

func BenchmarkComputeGroupKey(b *testing.B) {
	s := &AlertService{cfg: config.AlertConfig{GroupBy: []string{"alertname", "cluster", "namespace", "severity", "receiver"}}}
	labels := map[string]string{
		"alertname": "HighCPU",
		"cluster":   "prod",
		"namespace": "app",
		"severity":  "warning",
	}
	dims := alertnotify.Dims{Cluster: "prod", Namespace: "app"}
	b.ResetTimer()
	for b.Loop() {
		s.computeGroupKey("webhook", "firing", "warning", "HighCPU", labels, dims)
	}
}

func BenchmarkShrinkLargestNotifyStrings(b *testing.B) {
	long := strings.Repeat("x", 4000)
	b.ResetTimer()
	for b.Loop() {
		body := map[string]any{
			"markdown": map[string]any{"text": long},
		}
		shrinkLargestNotifyStrings(body)
	}
}

func BenchmarkParseUintCSV(b *testing.B) {
	raw := "1,2,3,4,5,6,7,8,9,10"
	b.ResetTimer()
	for b.Loop() {
		parseUintCSV(raw)
	}
}
