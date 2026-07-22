package dbmgmt

import "testing"

func TestNormalizeDbStageKey(t *testing.T) {
	t.Parallel()
	key, err := normalizeDbStageKey("DBA_Lead")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if key != "dba_lead" {
		t.Fatalf("got %q", key)
	}
	if _, err := normalizeDbStageKey("-bad"); err == nil {
		t.Fatal("expected error for invalid key")
	}
	gen, err := normalizeDbStageKey("")
	if err != nil {
		t.Fatalf("empty key should generate: %v", err)
	}
	if len(gen) < 8 || gen[:7] != "custom_" {
		t.Fatalf("unexpected generated key %q", gen)
	}
}
