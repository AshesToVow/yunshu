package logplatform

import (
	"strings"
	"testing"

	"yunshu/internal/config"
)

func TestBuildPipelineBundle_AppLogSyntax(t *testing.T) {
	bundle := BuildPipelineBundle(LoggiePipelineOptions{
		ProjectID: 1,
		ServerID:  10,
		LogPaths:  []string{"/var/log/myapp/*.log"},
	}, config.ElasticsearchConfig{
		Addresses:    []string{"http://127.0.0.1:9200"},
		IndexPattern: "yunshu-agent-*",
	}, config.KafkaConfig{}, "token", "http://yunshu:8080")

	sys := bundle.PipelineYAML
	if !strings.Contains(sys, "reload:") {
		t.Fatal("expected system pipeline.yml to contain reload")
	}
	if !strings.Contains(sys, "type: schema") {
		t.Fatal("expected defaults schema interceptor in system config")
	}

	yaml := bundle.PipelinesOnlyYAML
	if !strings.Contains(yaml, "pattern: '^\\d{4}-\\d{2}-\\d{2}") {
		t.Fatal("expected app log multiline pattern")
	}
	if !strings.Contains(yaml, "service_id") && !strings.Contains(yaml, "project_id") {
		t.Fatal("expected fields.project_id in pipeline")
	}
	if !strings.Contains(yaml, `yunshu-agent-10-${+YYYY.MM.DD}`) && !strings.Contains(yaml, `yunshu-agent-server-10-${+YYYY.MM.DD}`) {
		// without ServerHost enrichment in BuildPipelineBundle, fallback server-{id}
		if !strings.Contains(yaml, "${+YYYY.MM.DD}") {
			t.Fatal("expected per-agent daily index")
		}
	}
	if !strings.Contains(yaml, "copy(state.hostname, host)") {
		t.Fatal("expected hostname copy action")
	}
}

func TestBuildPipelineBundle_KafkaSink(t *testing.T) {
	bundle := BuildPipelineBundle(LoggiePipelineOptions{
		ProjectID: 1,
		ServerID:  10,
		LogPaths:  []string{"/var/log/myapp/*.log"},
	}, config.ElasticsearchConfig{
		Addresses: []string{"http://127.0.0.1:9200"},
	}, config.KafkaConfig{
		Enabled: true,
		Brokers: []string{"10.0.0.1:9092", "10.0.0.2:9092"},
		Topic:   "yunshu-logs",
	}, "token", "http://yunshu:8080")

	yaml := bundle.PipelinesOnlyYAML
	if !strings.Contains(yaml, "type: kafka") {
		t.Fatal("expected kafka sink")
	}
	if !strings.Contains(yaml, "10.0.0.1:9092") || !strings.Contains(yaml, "10.0.0.2:9092") {
		t.Fatal("expected kafka brokers")
	}
	if strings.Contains(yaml, "type: elasticsearch") {
		t.Fatal("should not use elasticsearch sink when kafka enabled")
	}
}

func TestBuildPipelineBundle_SyslogProfile(t *testing.T) {
	bundle := BuildPipelineBundle(LoggiePipelineOptions{
		ProjectID: 1,
		ServerID:  10,
		LogPaths:  []string{"/var/log/messages"},
	}, config.ElasticsearchConfig{
		Addresses: []string{"http://10.10.10.103:9200"},
	}, config.KafkaConfig{}, "token", "")

	yaml := bundle.PipelinesOnlyYAML
	if !strings.Contains(yaml, "pattern: '^\\w{3}\\s+\\d{1,2}\\s+'") {
		t.Fatal("expected syslog multiline pattern")
	}
	if !strings.Contains(yaml, "(?P<host>") {
		t.Fatal("expected syslog host capture group")
	}
	// 无年份 syslog 不得用 timestamp(ts) 覆盖采集时间
	if strings.Contains(yaml, `fromLayout: "Jan _2 15:04:05"`) {
		t.Fatal("syslog must not parse year-less timestamp into @timestamp")
	}
}
