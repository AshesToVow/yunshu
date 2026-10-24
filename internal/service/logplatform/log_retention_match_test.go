package logplatform

import "testing"

func TestMatchIndexPattern(t *testing.T) {
	cases := []struct {
		name, pattern string
		want          bool
	}{
		{"yunshu-agent-7-2026.07.13", "yunshu-agent-*", true},
		{"yunshu-agent-7-2026.07.13", "yunshu-logs-*", false},
		{"app-logs", "*", true},
		{"exact", "exact", true},
		{"exact", "other", false},
		{"a-b-c", "a-*-c", true},
		{"a-b-d", "a-*-c", false},
		// 大小写不敏感：历史存量模式可能带大写
		{"yunshu-agent-7-2026.07.13", "Yunshu-Agent-*", true},
		{"YUNSHU-AGENT-7", "yunshu-agent-*", true},
	}
	for _, tc := range cases {
		if got := matchIndexPattern(tc.name, tc.pattern); got != tc.want {
			t.Fatalf("matchIndexPattern(%q,%q)=%v want %v", tc.name, tc.pattern, got, tc.want)
		}
	}
}

func TestValidateRetentionIndexPattern(t *testing.T) {
	valid := []string{"", "yunshu-agent-*", "*", "app-logs", "a.b-c_d-*"}
	for _, p := range valid {
		if err := validateRetentionIndexPattern(p); err != nil {
			t.Fatalf("validateRetentionIndexPattern(%q) = %v, want nil", p, err)
		}
	}
	invalid := []string{
		"a-*,b-*",  // 逗号多索引
		"a-*+",     // 索引运算
		"-a-*",     // 前导 - 排除
		"a b*",     // 空白
		"A-*",      // 大写
		"a-?",      // 未支持通配
		"a-[0-9]*", // 字符类
		`a-*"`,     // 引号
	}
	for _, p := range invalid {
		if err := validateRetentionIndexPattern(p); err == nil {
			t.Fatalf("validateRetentionIndexPattern(%q) = nil, want error", p)
		}
	}
}
