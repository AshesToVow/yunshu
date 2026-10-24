package k8s

import (
	"testing"

	"yunshu/internal/pkg/k8scaps"
	"yunshu/internal/pkg/k8srole"
)

func TestResolveGrantCapabilities(t *testing.T) {
	t.Parallel()
	caps, preset, err := resolveGrantCapabilities([]string{"exec", "scale"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if preset != k8srole.ClusterAdmin {
		t.Fatalf("write caps map to admin preset=%s", preset)
	}
	if !k8scaps.Has(caps, k8scaps.Apply) {
		t.Fatalf("caps=%v", caps)
	}

	caps, preset, err = resolveGrantCapabilities(nil, "readonly_exec")
	if err != nil {
		t.Fatal(err)
	}
	if preset != "readonly_exec" || !k8scaps.Has(caps, k8scaps.Exec) {
		t.Fatalf("preset=%s caps=%v", preset, caps)
	}

	if _, _, err := resolveGrantCapabilities(nil, ""); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := resolveGrantCapabilities(nil, "custom"); err == nil {
		t.Fatal("expected custom without caps to fail")
	}
}
