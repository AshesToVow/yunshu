package ai

import (
	"strings"
	"testing"
)

func TestScrubNonBMPForMySQL(t *testing.T) {
	t.Parallel()
	in := "构建失败 \U0001F5C3 日志"
	got := scrubNonBMPForMySQL(in)
	for _, r := range got {
		if r > 0xFFFF {
			t.Fatalf("still has non-BMP: %q", got)
		}
	}
	if !strings.Contains(got, "构建失败") || !strings.Contains(got, "日志") {
		t.Fatalf("lost chinese text: %q", got)
	}
	plain := "hello 中文"
	if scrubNonBMPForMySQL(plain) != plain {
		t.Fatalf("plain changed")
	}
}
