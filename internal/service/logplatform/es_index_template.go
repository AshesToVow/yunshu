package logplatform

import (
	"context"
	"log/slog"
	"sync"

	"yunshu/internal/pkg/esclient"
)

const logIndexTemplateName = "yunshu-logs"

var ensureTemplateOnce sync.Once

// ensureLogIndexTemplateWithClient 用已创建的客户端写模板，禁止再调 ElasticsearchProvider.Client（会重入锁死锁）。
func ensureLogIndexTemplateWithClient(ctx context.Context, cli *esclient.Client) {
	if cli == nil {
		return
	}
	ensureTemplateOnce.Do(func() {
		body := map[string]any{
			"index_patterns": []string{"yunshu-agent-*", "yunshu-k8s-*"},
			"order":          100,
			"settings": map[string]any{
				"index": map[string]any{
					"number_of_shards":   1,
					"number_of_replicas": 0,
				},
			},
			"mappings": map[string]any{
				"properties": map[string]any{
					"@timestamp":     map[string]string{"type": "date"},
					"timestamp":      map[string]string{"type": "date"},
					"message":        map[string]any{"type": "text", "fields": map[string]any{"keyword": map[string]any{"type": "keyword", "ignore_above": 8192}}},
					"level":          map[string]any{"type": "keyword", "ignore_above": 32},
					"trace_id":       map[string]any{"type": "keyword", "ignore_above": 128},
					"span_id":        map[string]any{"type": "keyword", "ignore_above": 64},
					"service_name":   map[string]any{"type": "keyword", "ignore_above": 256},
					"project_id":     map[string]any{"type": "keyword", "ignore_above": 32},
					"service_id":     map[string]any{"type": "keyword", "ignore_above": 32},
					"server_id":      map[string]any{"type": "keyword", "ignore_above": 32},
					"log_source_id":  map[string]any{"type": "keyword", "ignore_above": 32},
					"collector_mode": map[string]any{"type": "keyword", "ignore_above": 16},
					"cluster_id":     map[string]any{"type": "keyword", "ignore_above": 32},
					"namespace":      map[string]any{"type": "keyword", "ignore_above": 128},
					"podname":        map[string]any{"type": "keyword", "ignore_above": 256},
					"containername":  map[string]any{"type": "keyword", "ignore_above": 256},
					"file_path":      map[string]any{"type": "keyword", "ignore_above": 512},
					"host":           map[string]any{"type": "keyword", "ignore_above": 256},
					"fields": map[string]any{
						"properties": map[string]any{
							"level":          map[string]any{"type": "keyword", "ignore_above": 32},
							"trace_id":       map[string]any{"type": "keyword", "ignore_above": 128},
							"span_id":        map[string]any{"type": "keyword", "ignore_above": 64},
							"project_id":     map[string]any{"type": "keyword", "ignore_above": 32},
							"service_id":     map[string]any{"type": "keyword", "ignore_above": 32},
							"collector_mode": map[string]any{"type": "keyword", "ignore_above": 16},
						},
					},
				},
			},
		}
		if err := cli.PutLegacyIndexTemplate(ctx, logIndexTemplateName, body); err != nil {
			slog.Default().With("component", "logplatform").Warn("Ensure log index template failed", "error", err)
			return
		}
		slog.Default().With("component", "logplatform").Info("Log index template ensured", "name", logIndexTemplateName)
	})
}
