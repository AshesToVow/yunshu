package middleware

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMaskSensitivePayloadJSON(t *testing.T) {
	nestedYAML := "pipelines:\n  - name: es-sink\n    type: elasticsearch\n    password: es-secret\n    username: elastic\n"
	payload, err := json.Marshal(map[string]any{
		"username":      "alice",
		"password":      "p@ssw0rd",
		"access_token":  "token-123",
		"pipelines_yml": nestedYAML,
		"nested": map[string]any{
			"secret": "nested-secret",
			"code":   403,
		},
		"code": "BadRequest",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := maskSensitivePayload(string(payload))

	for _, leaked := range []string{"p@ssw0rd", "token-123", "es-secret", "nested-secret"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("output leaked %q: %s", leaked, got)
		}
	}
	if !strings.Contains(got, `"code":403`) {
		t.Fatalf("numeric code should be preserved: %s", got)
	}
	if !strings.Contains(got, `"username":"alice"`) {
		t.Fatalf("non-sensitive field should be preserved: %s", got)
	}
	if !strings.Contains(got, "password: ***") {
		t.Fatalf("nested YAML password should be masked: %s", got)
	}
}

func TestMaskSensitivePayloadText(t *testing.T) {
	env := "export LOGGIE_TOKEN=super-secret\nNORMAL_FLAG=1\nYUNSHU_DB_PASSWORD=hunter2\n"
	got := maskSensitivePayload(env)
	if strings.Contains(got, "super-secret") || strings.Contains(got, "hunter2") {
		t.Fatalf("env secrets leaked: %s", got)
	}
	if !strings.Contains(got, "NORMAL_FLAG=1") {
		t.Fatalf("non-sensitive env line should be preserved: %s", got)
	}
}

func TestMaskSensitivePayloadPlainJSONSecretKey(t *testing.T) {
	got := maskSensitivePayload(`{"api_key":"abc","private_key_data":"-----BEGIN PRIVATE KEY-----","empty":""}`)
	if strings.Contains(got, "abc") || strings.Contains(got, "BEGIN PRIVATE KEY") {
		t.Fatalf("json secrets leaked: %s", got)
	}
	if !strings.Contains(got, `"empty":""`) {
		t.Fatalf("empty non-sensitive field should be preserved: %s", got)
	}
}

func TestMaskSensitivePayloadEmptyAndInvalid(t *testing.T) {
	if got := maskSensitivePayload("   "); got != "" {
		t.Fatalf("blank payload = %q, want empty", got)
	}
	// 非 JSON 文本走 YAML/env 行规则，不 panic 且保留普通内容。
	got := maskSensitivePayload("hello world")
	if got != "hello world" {
		t.Fatalf("plain text changed: %q", got)
	}
}
