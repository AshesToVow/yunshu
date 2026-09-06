package ai

import (
	"testing"

	"yunshu/internal/service/logplatform"
)

func TestNormalizeLogSignature(t *testing.T) {
	t.Parallel()
	a := logplatform.NormalizeLogSignature("ERROR conn 10.0.0.1 failed request-id=aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee timeout after 30s")
	b := logplatform.NormalizeLogSignature("ERROR conn 192.168.1.9 failed request-id=11111111-2222-3333-4444-555555555555 timeout after 90s")
	if a == "" || a != b {
		t.Fatalf("expected same signature, got %q vs %q", a, b)
	}
	msg := "something ERROR happened"
	summary := logplatform.SummarizeLogHits(nil, 8)
	if summary == nil {
		t.Fatal("summary nil")
	}
	_ = msg
}
