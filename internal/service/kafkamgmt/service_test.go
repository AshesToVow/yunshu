package kafkamgmt

import "testing"

func TestNormalizeSASL(t *testing.T) {
	if got := normalizeSASL("", "user"); got != "plain" {
		t.Fatalf("expected plain, got %q", got)
	}
	if got := normalizeSASL("none", ""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := normalizeSASL("SCRAM-SHA-256", "u"); got != "scram-sha-256" {
		t.Fatalf("expected scram-sha-256, got %q", got)
	}
}

func TestSplitBrokers(t *testing.T) {
	got := splitBrokers("a:9092, b:9092;\nc:9092")
	if len(got) != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
}
