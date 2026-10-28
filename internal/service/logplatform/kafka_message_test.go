package logplatform

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
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
	doc, _, err := parseKafkaLogMessage(raw, "yunshu-k8s-4-p1-2026.08.12", "yunshu-agent", "yunshu-k8s", "yunshu-k8s")
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
	doc, index, err := parseKafkaLogMessage(raw, "yunshu-k8s-4-p1-2026.08.12", "yunshu-agent", "yunshu-k8s", "yunshu-k8s")
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

// 回归：Loggie 文件源会给事件带 "@timestamp": null，原样入库后文档没有时间，
// 检索/统计的时间窗过滤永远查不到（Nginx 访问统计恒为空就卡在这一步）。
func TestParseKafkaLogMessage_NullTimestampReplaced(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "null_at_timestamp_uses_ts",
			raw: map[string]any{
				"body":       `1.2.3.4 - - [28/Oct/2026:21:08:21 +0800] "GET / HTTP/1.1" 200 368 "-" "curl" "-"`,
				"@timestamp": nil,
				"ts":         "2026-10-28T13:08:21.000Z",
			},
			want: "2026-10-28T13:08:21Z",
		},
		{
			name: "null_at_timestamp_uses_raw_nginx_time",
			raw: map[string]any{
				"body":       `1.2.3.4 - - [28/Oct/2026:21:08:21 +0800] "GET / HTTP/1.1" 200 368`,
				"@timestamp": nil,
				"ts":         "28/Oct/2026:21:08:21 +0800",
			},
			want: "2026-10-28T21:08:21+08:00",
		},
		{
			name: "valid_at_timestamp_kept",
			raw: map[string]any{
				"body":       "hello",
				"@timestamp": "2026-10-28T05:00:00Z",
			},
			want: "2026-10-28T05:00:00Z",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			raw, _ := json.Marshal(tc.raw)
			doc, _, err := parseKafkaLogMessage(raw, "yunshu-agent-10-10-10-5-2026.10.28", "yunshu-agent", "yunshu-k8s", "yunshu-k8s")
			if err != nil {
				t.Fatal(err)
			}
			got, _ := doc["@timestamp"].(string)
			ts, err := time.Parse(time.RFC3339Nano, got)
			if err != nil {
				t.Fatalf("@timestamp must be RFC3339, got %q", got)
			}
			if !ts.Equal(mustParse(t, tc.want)) {
				t.Fatalf("@timestamp got %q want %q", got, tc.want)
			}
		})
	}
}

// 全部时间线索都缺失时也要落一个可用时间，不能留 null。
func TestParseKafkaLogMessage_MissingTimestampFilled(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]any{"body": "no time here"})
	doc, _, err := parseKafkaLogMessage(raw, "yunshu-agent-10-10-10-5-2026.10.28", "yunshu-agent", "yunshu-k8s", "yunshu-k8s")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := doc["@timestamp"].(string)
	if _, err := time.Parse(time.RFC3339Nano, got); err != nil {
		t.Fatalf("@timestamp must be parseable, got %q", got)
	}
}

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad expected time %q: %v", s, err)
	}
	return ts
}
