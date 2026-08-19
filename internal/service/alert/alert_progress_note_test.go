package alert

import (
	"strings"
	"testing"
)

func TestNormalizeAlertNoteContent(t *testing.T) {
	t.Parallel()
	if _, err := normalizeAlertNoteContent("  "); err == nil {
		t.Fatal("empty should fail")
	}
	got, err := normalizeAlertNoteContent("  已扩容磁盘  ")
	if err != nil || got != "已扩容磁盘" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := normalizeAlertNoteContent(strings.Repeat("a", maxAlertNoteRunes+1)); err == nil {
		t.Fatal("too long should fail")
	}
}
