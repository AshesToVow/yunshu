package logplatform

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestDetectParseProfile_Elasticsearch(t *testing.T) {
	p := detectParseProfile("/export/elasticsearch-7.14.2/logs/yunshu.log", nil)
	if p.name != "elasticsearch" {
		t.Fatalf("expected elasticsearch profile, got %s", p.name)
	}
	if !strings.Contains(p.regexPattern, "(?P<level>") {
		t.Fatal("expected level capture group")
	}
}

func TestExtractLevelFromMessage_Elasticsearch(t *testing.T) {
	msg := "[2026-07-13T23:42:49,235][WARN ][o.e.t.ThreadPool ] [yunshuNode] timer thread slept"
	if lv := extractLevelFromMessage(msg); lv != "WARN" {
		t.Fatalf("expected WARN, got %q", lv)
	}
}

func TestExtractLevelFromMessage_Spring(t *testing.T) {
	msg := "2026-07-13 23:42:49 WARN  com.example.App started"
	if lv := extractLevelFromMessage(msg); lv != "WARN" {
		t.Fatalf("expected WARN, got %q", lv)
	}
}

func TestLevelFilterIncludesMessagePatterns(t *testing.T) {
	q := levelFilter("WARN")
	boolQ, ok := q["bool"].(map[string]any)
	if !ok {
		t.Fatalf("expected bool query, got %#v", q)
	}
	should, ok := boolQ["should"].([]map[string]any)
	if !ok || len(should) < 3 {
		t.Fatalf("expected multiple should clauses, got %#v", boolQ)
	}
	foundWildcard := false
	for _, clause := range should {
		if wc, ok := clause["wildcard"].(map[string]any); ok {
			if msg, ok := wc["message"].(string); ok && strings.Contains(msg, "WARN") {
				foundWildcard = true
			}
		}
	}
	if !foundWildcard {
		t.Fatal("expected message wildcard for WARN")
	}
}

func TestBuildMultiPipelineBundle_ElasticsearchLevelRegex(t *testing.T) {
	sources := sourcesFromLogSources(1, 7, []model.ServiceLogSource{
		{ID: 21, ServiceID: 5, LogType: "file", Path: "/export/elasticsearch-7.14.2/logs/yunshu.log"},
	})
	bundle := BuildMultiPipelineBundle(1, 7, sources, 9196, config.ElasticsearchConfig{
		Addresses:    []string{"http://127.0.0.1:9200"},
		IndexPattern: "yunshu-logs-*",
	}, "token", "", "/export/loggie")
	yaml := bundle.PipelinesOnlyYAML
	if !strings.Contains(yaml, "(?P<level>") {
		t.Fatal("expected level capture in elasticsearch pipeline")
	}
	if !strings.Contains(yaml, "action: timestamp(ts)") {
		t.Fatal("expected Loggie native timestamp action")
	}
	if !strings.Contains(yaml, `move(ts, @timestamp)`) {
		t.Fatal("expected move ts to @timestamp")
	}
	if !strings.Contains(yaml, "multiline:") {
		t.Fatal("expected multiline config")
	}
}
