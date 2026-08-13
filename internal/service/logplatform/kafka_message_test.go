package logplatform

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCoalesceLogMessage_AvoidsNilLiteral(t *testing.T) {
	t.Parallel()
	doc := map[string]any{
		"message": nil,
		"body":    "10.244.1.1 - - [12/Aug/2026:10:41:09 +0000] \"GET / HTTP/1.1\" 200",
	}
	got := coalesceLogMessage(doc)
	if !strings.Contains(got, "GET / HTTP/1.1") {
		t.Fatalf("expected access log body, got %q", got)
	}
	if got == "<nil>" {
		t.Fatal("must not return <nil> literal")
	}
}

func TestParseKafkaLogMessage_NilBodyNotNilString(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]any{
		"body":       nil,
		"message":    nil,
		"@timestamp": "2026-08-12T10:41:09Z",
		"fields": map[string]any{
			"collector_mode": "k8s",
			"namespace":      "dodo-test",
			"pod":            "dodo-web-x",
		},
	})
	doc, _, err := parseKafkaLogMessage(raw, "yunshu-k8s-4-p1-2026.08.12", "yunshu-agent")
	if err != nil {
		t.Fatal(err)
	}
	if msg, _ := doc["message"].(string); msg == "<nil>" {
		t.Fatalf("message should not be <nil>, doc=%v", doc)
	}
	if _, ok := doc["message"]; ok {
		if s, _ := doc["message"].(string); strings.TrimSpace(s) == "" {
			t.Fatal("empty message should be deleted")
		}
	}
}

func TestParseKafkaLogMessage_UsesBodyString(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]any{
		"body": "hello from nginx",
		"fields": map[string]any{
			"collector_mode": "k8s",
		},
	})
	doc, index, err := parseKafkaLogMessage(raw, "yunshu-k8s-4-p1-2026.08.12", "yunshu-agent")
	if err != nil {
		t.Fatal(err)
	}
	if doc["message"] != "hello from nginx" {
		t.Fatalf("message=%v", doc["message"])
	}
	if !strings.HasPrefix(index, "yunshu-k8s-4-p1-") {
		t.Fatalf("index=%s", index)
	}
}
