package k8s

import (
	"context"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8srole"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGvkClusterScoped(t *testing.T) {
	tests := []struct {
		kind    string
		cluster bool
	}{
		{"Ingress", false},
		{"Pod", false},
		{"IngressClass", true},
		{"NodeMetrics", true},
		{"CustomResourceDefinition", true},
		{"Deployment", false},
	}
	for _, tc := range tests {
		got := gvkClusterScoped(schema.GroupVersionKind{Kind: tc.kind})
		if got != tc.cluster {
			t.Fatalf("gvkClusterScoped(%q) = %v, want %v", tc.kind, got, tc.cluster)
		}
	}
}

func TestEnsureClusterScopedAllowed(t *testing.T) {
	t.Parallel()
	userCtx := auth.WithRequestUser(context.Background(), &auth.CurrentUser{ID: 7, Username: "dev"})

	noScope := userCtx
	if err := ensureClusterScopedAllowed(noScope, "CustomResourceDefinition"); err != nil {
		t.Fatalf("no scope should skip: %v", err)
	}

	restricted := k8sauth.WithRequestScope(userCtx, k8sauth.RequestScope{
		ClusterID: 1,
		Grants: []model.K8sClusterAccessGrant{{
			Preset: k8srole.ClusterAdmin, Namespaces: "app-dev",
		}},
	})
	if err := ensureClusterScopedAllowed(restricted, "CustomResourceDefinition"); err == nil {
		t.Fatal("namespace-restricted admin must not delete cluster-scoped resources")
	}

	unrestricted := k8sauth.WithRequestScope(userCtx, k8sauth.RequestScope{
		ClusterID: 1,
		Grants:    []model.K8sClusterAccessGrant{{Preset: k8srole.ClusterAdmin}},
	})
	if err := ensureClusterScopedAllowed(unrestricted, "CustomResourceDefinition"); err != nil {
		t.Fatalf("unrestricted admin should pass: %v", err)
	}

	bypass := k8sauth.WithRequestScope(userCtx, k8sauth.RequestScope{ClusterID: 1, BypassAuth: true})
	if err := ensureClusterScopedAllowed(bypass, "Namespace"); err != nil {
		t.Fatalf("super admin bypass should pass: %v", err)
	}
}
