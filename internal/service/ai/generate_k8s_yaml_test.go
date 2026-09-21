package ai

import (
	"strings"
	"testing"
)

func TestExtractYAMLDocument(t *testing.T) {
	t.Parallel()
	raw := "说明如下：\n```yaml\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\n```\n"
	got := extractYAMLDocument(raw)
	if got == "" || !strings.Contains(got, "apiVersion: v1") || !strings.Contains(got, "kind: ConfigMap") {
		t.Fatalf("unexpected: %q", got)
	}
	plain := "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: x"
	if extractYAMLDocument(plain) != plain {
		t.Fatalf("plain changed")
	}
}
