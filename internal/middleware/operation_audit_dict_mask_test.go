package middleware

import (
	"strings"
	"testing"
)

func TestMaskSensitivePayloadInPath_DictEntry(t *testing.T) {
	t.Parallel()
	payload := `{"dict_type":"alert_webhook_token","label":"hook","value":"plaintext-secret"}`

	got := maskSensitivePayloadInPath(payload, "/api/v1/dict/entries/12")
	if strings.Contains(got, "plaintext-secret") {
		t.Fatalf("dict entry secret leaked: %s", got)
	}
	if !strings.Contains(got, `"label":"hook"`) {
		t.Fatalf("non-sensitive field should be preserved: %s", got)
	}

	// 非字典路径下 value 不是通用敏感键，原样保留（避免误伤普通业务字段）。
	got = maskSensitivePayloadInPath(payload, "/api/v1/alerts/rules")
	if !strings.Contains(got, "plaintext-secret") {
		t.Fatalf("generic path should not mask value: %s", got)
	}

	// 非敏感字典类型不脱敏 value。
	got = maskSensitivePayloadInPath(`{"dict_type":"alert_severity","value":"critical"}`, "/api/v1/dict/entries")
	if !strings.Contains(got, "critical") {
		t.Fatalf("non-sensitive dict value should be preserved: %s", got)
	}
}

func TestIsDictEntryAuditPath(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"/api/v1/dict/entries":     true,
		"/api/v1/dict/entries/12":  true,
		"/api/v1/dict-entries":     true,
		"/api/v1/dict/options/foo": false,
		"/api/v1/users":            false,
	}
	for path, want := range cases {
		if got := isDictEntryAuditPath(path); got != want {
			t.Errorf("isDictEntryAuditPath(%q)=%v want %v", path, got, want)
		}
	}
}
