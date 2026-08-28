package platformtpl

import (
	"testing"
)

func TestBuiltinContent(t *testing.T) {
	body, format, ok := BuiltinContent("cicd.apollo.backend-launch")
	if !ok || body == "" || format != "shell" {
		t.Fatalf("expected cicd apollo seed, got ok=%v format=%q len=%d", ok, format, len(body))
	}
	_, _, ok = BuiltinContent("missing.key")
	if ok {
		t.Fatal("expected missing key")
	}
}

func TestNormalizeCategory(t *testing.T) {
	if normalizeCategory("cicd") != "cicd_snippet" {
		t.Fatal("cicd alias")
	}
	if normalizeCategory("bad") != "" {
		t.Fatal("bad category")
	}
}
