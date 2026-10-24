package ai

import "testing"

func TestCheckWriteToolPolicy(t *testing.T) {
	t.Parallel()
	if err := checkWriteToolPolicy("restart_deployment", `{}`, "", "x"); err == nil {
		t.Fatal("expected reject empty namespace for restart")
	}
	if err := checkWriteToolPolicy("scale_deployment", `{"replicas":10}`, "", "x"); err == nil {
		t.Fatal("expected reject empty namespace for scale")
	}
	if err := checkWriteToolPolicy("delete_pod", `{}`, "kube-system", "cleanup"); err == nil {
		t.Fatal("expected reject delete in kube-system")
	}
	if err := checkWriteToolPolicy("delete_pod", `{}`, "yunshu-logging", "cleanup"); err == nil {
		t.Fatal("expected reject delete in yunshu-logging")
	}
	if err := checkWriteToolPolicy("scale_deployment", `{"replicas":100}`, "default", "need more"); err == nil {
		t.Fatal("expected reject scale >50 without emergency")
	}
	if err := checkWriteToolPolicy("scale_deployment", `{"replicas":100}`, "default", "emergency capacity"); err != nil {
		t.Fatalf("expected allow with emergency: %v", err)
	}
	if err := checkWriteToolPolicy("delete_pod", `{}`, "app", "restart stuck pod"); err != nil {
		t.Fatalf("expected allow delete in app ns: %v", err)
	}
}

func TestPackUnpackEmbedding(t *testing.T) {
	t.Parallel()
	in := []float32{0.1, -0.5, 2.25}
	b := packEmbedding(in)
	out := unpackEmbedding(b)
	if len(out) != len(in) {
		t.Fatalf("len %d != %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Fatalf("idx %d: %v != %v", i, out[i], in[i])
		}
	}
}
