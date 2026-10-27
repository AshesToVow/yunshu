package logplatform

import (
	"strings"
	"testing"
	"time"
)

func TestValidateSearchTimeRange(t *testing.T) {
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) string { return base.Add(d).Format(time.RFC3339) }
	cases := []struct {
		name    string
		from    string
		to      string
		wantErr string
	}{
		{name: "两端为空不校验", from: "", to: ""},
		{name: "单端为空不校验", from: at(0), to: ""},
		{name: "正常一小时", from: at(-time.Hour), to: at(0)},
		{name: "恰好 31 天", from: at(-31 * 24 * time.Hour), to: at(0)},
		{name: "超过 31 天", from: at(-31*24*time.Hour - time.Second), to: at(0), wantErr: "31 天"},
		{name: "to 早于 from", from: at(time.Hour), to: at(0), wantErr: "to 早于 from"},
		{name: "from 非 RFC3339", from: "2026-10-01", to: at(0), wantErr: "RFC3339"},
		{name: "to 非 RFC3339", from: at(-time.Hour), to: "not-a-time", wantErr: "RFC3339"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSearchTimeRange(tc.from, tc.to)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validateSearchTimeRange(%q,%q) = %v, want nil", tc.from, tc.to, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateSearchTimeRange(%q,%q) = nil, want error containing %q", tc.from, tc.to, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestEscapeWildcardPattern(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"/var/log/app.log", "/var/log/app.log"},
		{"*", `\*`},
		{"app-?.log", `app-\?.log`},
		{`a\b`, `a\\b`},
		{"a*b?c", `a\*b\?c`},
	}
	for _, tc := range cases {
		if got := escapeWildcardPattern(tc.in); got != tc.want {
			t.Fatalf("escapeWildcardPattern(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestEnsureRetentionIndexPatternInScope(t *testing.T) {
	cases := []struct {
		name       string
		pattern    string
		configured string
		wantErr    bool
	}{
		{name: "空模式", pattern: ""},
		{name: "纯通配归一为平台", pattern: "*"},
		{name: "平台前缀", pattern: "yunshu-agent-*"},
		{name: "平台前缀大写", pattern: "Yunshu-K8s-*"},
		{name: "与配置一致的自定义模式", pattern: "app-logs-*", configured: "app-logs-*"},
		{name: "业务索引拒绝", pattern: "logs-*", wantErr: true},
		{name: "非平台精确名拒绝", pattern: "app-logs", configured: "other-*", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureRetentionIndexPatternInScope(tc.pattern, tc.configured)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
