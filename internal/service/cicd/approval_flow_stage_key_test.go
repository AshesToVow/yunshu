package cicd

import "testing"

func TestNormalizeStageKey(t *testing.T) {
	t.Parallel()
	key, err := normalizeStageKey("Test_Lead")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if key != "test_lead" {
		t.Fatalf("got %q", key)
	}
	if _, err := normalizeStageKey("1bad"); err == nil {
		t.Fatal("expected error for key starting with digit")
	}
	gen, err := normalizeStageKey("")
	if err != nil {
		t.Fatalf("empty key should generate: %v", err)
	}
	if len(gen) < 8 || gen[:7] != "custom_" {
		t.Fatalf("unexpected generated key %q", gen)
	}
}
