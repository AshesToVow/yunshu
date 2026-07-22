package k8s

import (
	"testing"

	"yunshu/internal/model"
)

func modelPermission(path, method, desc string, enabled bool) model.Permission {
	return model.Permission{
		Resource:        path,
		Action:          method,
		Description:     desc,
		K8sScopeEnabled: enabled,
	}
}

func TestExtractManifestNamespaces(t *testing.T) {
	t.Parallel()
	manifest := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
  namespace: ns-a
---
apiVersion: v1
kind: Pod
metadata:
  name: b
---
apiVersion: v1
kind: Namespace
metadata:
  name: skip-me
`
	got := extractManifestNamespaces(manifest)
	want := map[string]bool{"ns-a": true, "default": true}
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	for _, ns := range got {
		if !want[ns] {
			t.Fatalf("unexpected ns %q in %v", ns, got)
		}
	}
}

func TestIsScopedK8sPermissionOffTag(t *testing.T) {
	t.Parallel()
	p := modelPermission("/api/v1/pods", "GET", "[k8s-scope=off]", false)
	if isScopedK8sPermission(p) {
		t.Fatal("off tag must exclude from scope catalog")
	}
	p2 := modelPermission("/api/v1/pods", "GET", "", false)
	if !isScopedK8sPermission(p2) {
		t.Fatal("default k8s path should be scoped")
	}
}
