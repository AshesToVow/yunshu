package logplatform

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestBuildMultiPipelineBundle_PerLogSourceFields(t *testing.T) {
	sources := sourcesFromLogSources(1, 7, []model.ServiceLogSource{
		{ID: 11, ServiceID: 3, LogType: "file", Path: "/var/log/messages"},
		{ID: 12, ServiceID: 4, LogType: "file", Path: "/var/log/kube-apiserver", IncludeRegex: strPtr("*.log")},
	})
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	bundle := BuildMultiPipelineBundle(1, 7, sources, 9196, config.ElasticsearchConfig{
		Addresses:    []string{"http://10.10.10.103:9200"},
		IndexPattern: "yunshu-agent-*",
	}, config.KafkaConfig{}, "token", "http://127.0.0.1:8080", "/export/loggie")

	yaml := bundle.PipelinesOnlyYAML
	if bundle.PipelineCount != 2 {
		t.Fatalf("expected 2 pipelines, got %d", bundle.PipelineCount)
	}
	if !strings.Contains(yaml, "log_source_id: \"11\"") || !strings.Contains(yaml, "service_id: \"3\"") {
		t.Fatal("missing fields for first log source")
	}
	if !strings.Contains(yaml, "log_source_id: \"12\"") || !strings.Contains(yaml, "service_id: \"4\"") {
		t.Fatal("missing fields for second log source")
	}
	if !strings.Contains(yaml, `index: "yunshu-agent-7-${+YYYY.MM.DD}"`) {
		t.Fatalf("expected per-agent index, got:\n%s", yaml)
	}
	if !strings.Contains(yaml, "name: yunshu-p1-s7-ls11") {
		t.Fatal("expected per-source pipeline name")
	}
	if !strings.Contains(bundle.EnvFile, "LOGGIE_DEPLOY_DIR=/export/loggie") {
		t.Fatal("expected deploy dir in env file")
	}
}

func strPtr(s string) *string { return &s }
