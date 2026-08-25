package k8s

import (
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8scaps"
)

func TestCapabilitiesForPresetAndInfer(t *testing.T) {
	t.Parallel()
	ro := k8scaps.ForPreset("readonly")
	if !k8scaps.Has(ro, CapRead) || k8scaps.Has(ro, CapExec) {
		t.Fatalf("readonly caps=%v", ro)
	}
	re := k8scaps.ForPreset("readonly_exec")
	if !k8scaps.Has(re, CapExec) {
		t.Fatalf("readonly_exec caps=%v", re)
	}
	if k8scaps.InferPreset(ro) != "readonly" {
		t.Fatalf("infer readonly")
	}
	if k8scaps.InferPreset(re) != "readonly_exec" {
		t.Fatalf("infer readonly_exec")
	}
	if k8scaps.InferPreset(k8scaps.All()) != "admin" {
		t.Fatalf("infer admin")
	}
	if k8scaps.InferPreset([]string{CapRead, CapRestart}) != "custom" {
		t.Fatalf("infer custom")
	}
}

func TestGrantCapabilitySetFallsBackPreset(t *testing.T) {
	t.Parallel()
	g := model.K8sClusterAccessGrant{Preset: "readonly_exec", Capabilities: ""}
	caps := k8scaps.FromGrant(g)
	if !k8scaps.Has(caps, CapExec) {
		t.Fatalf("got %v", caps)
	}
	g2 := model.K8sClusterAccessGrant{Preset: "readonly", Capabilities: `["read","scale"]`}
	caps2 := k8scaps.FromGrant(g2)
	if !k8scaps.Has(caps2, CapScale) || k8scaps.Has(caps2, CapExec) {
		t.Fatalf("got %v", caps2)
	}
}

func TestRequiredK8sCapability(t *testing.T) {
	t.Parallel()
	if RequiredK8sCapability(nil, "/api/v1/secrets/reveal", "GET", "") != CapSecretReveal {
		t.Fatal("secret reveal")
	}
	if RequiredK8sCapability(nil, "/api/v1/pods/exec", "POST", "") != CapExec {
		t.Fatal("exec")
	}
	if RequiredK8sCapability(nil, "/api/v1/nodes/drain", "POST", "") != CapDestructive {
		t.Fatal("drain")
	}
	if RequiredK8sCapability(nil, "/api/v1/deployments", "GET", "list") != CapRead {
		t.Fatal("read get")
	}
}

func TestRankFromCapabilities(t *testing.T) {
	t.Parallel()
	if k8scaps.Rank([]string{CapRead}) != K8sAccessRankReadonly {
		t.Fatal("read rank")
	}
	if k8scaps.Rank([]string{CapRead, CapExec}) != K8sAccessRankReadonlyExec {
		t.Fatal("exec rank")
	}
	if k8scaps.Rank([]string{CapRead, CapApply}) != K8sAccessRankAdmin {
		t.Fatal("apply rank")
	}
}
