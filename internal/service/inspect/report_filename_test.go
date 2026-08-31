package inspect

import (
	"strings"
	"testing"
)

func TestReportDownloadFilename(t *testing.T) {
	t.Parallel()
	got := reportDownloadFilename("蓝海项目", 12, "pdf")
	if got != "蓝海项目-巡检-12.pdf" {
		t.Fatalf("got %q", got)
	}
	got = reportDownloadFilename("a/b:c?", 3, "xlsx")
	if !strings.Contains(got, "巡检-3.xlsx") {
		t.Fatalf("got %q", got)
	}
	got = reportDownloadFilename("  ", 9, "pdf")
	if got != "inspect-run-9.pdf" {
		t.Fatalf("empty name fallback got %q", got)
	}
}
