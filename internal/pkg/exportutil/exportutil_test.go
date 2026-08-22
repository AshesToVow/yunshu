package exportutil

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	t.Parallel()
	if got := SanitizeFilename("../../etc/passwd"); got != "passwd" {
		t.Fatalf("got %q", got)
	}
	if got := SanitizeFilename(""); got != "download.bin" {
		t.Fatalf("got %q", got)
	}
}

func TestContentDispositionAttachment(t *testing.T) {
	t.Parallel()
	h := ContentDispositionAttachment("login_logs.xlsx")
	if !strings.Contains(h, "filename=") || !strings.Contains(h, "UTF-8") {
		t.Fatalf("bad header: %s", h)
	}
}
