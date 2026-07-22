package k8s

import (
	"testing"

	"yunshu/internal/model"
	cryptox "yunshu/internal/pkg/crypto"
)

func TestSealOpenClusterSecret_RoundTrip(t *testing.T) {
	aead, err := cryptox.NewAESGCMFromKeyString("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	plain := "apiVersion: v1\nkind: Config"
	sealed, err := sealClusterSecret(aead, plain)
	if err != nil {
		t.Fatal(err)
	}
	if sealed == plain {
		t.Fatal("expected ciphertext different from plaintext")
	}
	opened, err := openClusterSecret(aead, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if opened != plain {
		t.Fatalf("got %q want %q", opened, plain)
	}
}

func TestOpenClusterSecret_PlaintextCompat(t *testing.T) {
	aead, err := cryptox.NewAESGCMFromKeyString("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	plain := "apiVersion: v1\nkind: Config\nclusters: []"
	opened, err := openClusterSecret(aead, plain)
	if err != nil {
		t.Fatal(err)
	}
	if opened != plain {
		t.Fatalf("plaintext compat failed: %q", opened)
	}
}

func TestIsScopedK8sPermission_IncludesGET(t *testing.T) {
	if !isScopedK8sPermission(model.Permission{
		Resource: "/api/v1/pods",
		Action:   "GET",
	}) {
		t.Fatal("GET /pods should be scoped by default")
	}
	if !isScopedK8sPermission(model.Permission{
		Resource: "/api/v1/deployments",
		Action:   "DELETE",
	}) {
		t.Fatal("DELETE should be scoped")
	}
}

func TestSyncDenyNamespaces_WildClusterWritesZero(t *testing.T) {
	// smoke: empty principal returns zeros
	added, skipped, err := syncDenyNamespaces(t.Context(), nil, "", "", []uint{0}, []string{"kube-system"})
	if err != nil || added != 0 || skipped != 0 {
		t.Fatalf("empty principal: added=%d skipped=%d err=%v", added, skipped, err)
	}
}
