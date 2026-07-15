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
	}, "token", "http://yunshu:8080")

	yaml := bundle.PipelineYAML
	if !strings.Contains(yaml, "pattern: '^\\d{4}-\\d{2}-\\d{2}") {
		t.Fatal("expected app log multiline pattern")
	}
	if !strings.Contains(yaml, "service_id") && !strings.Contains(yaml, "project_id") {
		t.Fatal("expected fields.project_id in pipeline")
	}
	if !strings.Contains(yaml, `yunshu-agent-10-${+YYYY.MM.DD}`) {
		t.Fatal("expected per-agent daily index")
	}
	if !strings.Contains(yaml, "type: schema") {
		t.Fatal("expected defaults schema interceptor")
	}
	if !strings.Contains(yaml, "copy(state.hostname, host)") {
		t.Fatal("expected hostname copy action")
	}
}

func TestBuildPipelineBundle_SyslogProfile(t *testing.T) {
	bundle := BuildPipelineBundle(LoggiePipelineOptions{
		ProjectID: 1,
		ServerID:  10,
		LogPaths:  []string{"/var/log/messages"},
	}, config.ElasticsearchConfig{
		Addresses: []string{"http://10.10.10.103:9200"},
	}, "token", "")

	yaml := bundle.PipelineYAML
	if !strings.Contains(yaml, "pattern: '^\\w{3}\\s+\\d{1,2}\\s+'") {
		t.Fatal("expected syslog multiline pattern")
	}
	if !strings.Contains(yaml, "(?P<host>") {
		t.Fatal("expected syslog host capture group")
	}
}
