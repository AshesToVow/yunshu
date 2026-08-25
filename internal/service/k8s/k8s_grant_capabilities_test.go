package k8s

import (
	"testing"

	"yunshu/internal/pkg/k8scaps"
)

func TestResolveGrantCapabilities(t *testing.T) {
	t.Parallel()
	caps, preset, err := resolveGrantCapabilities([]string{"exec", "scale"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if preset != k8scaps.PresetCustom {
		t.Fatalf("preset=%s", preset)
	}
	if !k8scaps.Has(caps, k8scaps.Read) || !k8scaps.Has(caps, k8scaps.Exec) || !k8scaps.Has(caps, k8scaps.Scale) {
		t.Fatalf("caps=%v", caps)
	}

	caps, preset, err = resolveGrantCapabilities(nil, "readonly_exec")
	if err != nil {
		t.Fatal(err)
	}
	if preset != "readonly_exec" || len(caps) != 2 {
		t.Fatalf("preset=%s caps=%v", preset, caps)
	}

	if _, _, err := resolveGrantCapabilities(nil, ""); err == nil {
		t.Fatal("expected error")
	}
	if _, _, err := resolveGrantCapabilities(nil, "custom"); err == nil {
		t.Fatal("expected custom without caps to fail")
	}
}
