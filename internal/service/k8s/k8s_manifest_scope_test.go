package k8s

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8srole"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestManifestClusterScoped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		gvk  schema.GroupVersionKind
		want bool
	}{
		{schema.GroupVersionKind{Kind: "ClusterRoleBinding", Group: "rbac.authorization.k8s.io"}, true},
		{schema.GroupVersionKind{Kind: "ClusterRole", Group: "rbac.authorization.k8s.io"}, true},
		{schema.GroupVersionKind{Kind: "CustomResourceDefinition", Group: "apiextensions.k8s.io"}, true},
		{schema.GroupVersionKind{Kind: "PersistentVolume"}, true},
		{schema.GroupVersionKind{Kind: "Node"}, true},
		{schema.GroupVersionKind{Kind: "StorageClass", Group: "storage.k8s.io"}, true},
		{schema.GroupVersionKind{Kind: "ValidatingWebhookConfiguration", Group: "admissionregistration.k8s.io"}, true},
		// 自定义组同名 Kind 不得误判为集群级。
		{schema.GroupVersionKind{Kind: "ClusterRole", Group: "example.com"}, false},
		{schema.GroupVersionKind{Kind: "ConfigMap"}, false},
		{schema.GroupVersionKind{Kind: "Deployment", Group: "apps"}, false},
	}
	for _, tc := range cases {
		if got := manifestClusterScoped(tc.gvk); got != tc.want {
			t.Errorf("manifestClusterScoped(%s/%s)=%v want %v", tc.gvk.Group, tc.gvk.Kind, got, tc.want)
		}
	}
}

func TestEnsureManifestClusterScopesAllowed(t *testing.T) {
	t.Parallel()
	manifest := "apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding\nmetadata:\n  name: escalate\n"
	namespacedManifest := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm\n  namespace: app\n"
	namespaceManifest := "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: new-ns\n"

	restricted := k8sauth.WithRequestScope(context.Background(), k8sauth.RequestScope{
		ClusterID: 1,
		Grants: []model.K8sClusterAccessGrant{{
			Preset: k8srole.ClusterAdmin, ClusterID: 1, Namespaces: "app",
		}},
	})
	unrestricted := k8sauth.WithRequestScope(context.Background(), k8sauth.RequestScope{
		ClusterID: 1,
		Grants: []model.K8sClusterAccessGrant{{
			Preset: k8srole.ClusterAdmin, ClusterID: 1,
		}},
	})
	bypass := k8sauth.WithRequestScope(context.Background(), k8sauth.RequestScope{
		ClusterID: 1, BypassAuth: true,
	})
	skip := k8sauth.WithSkipNamespacePolicy(restricted)

	svc := &DynamicResourceService{}
	cases := []struct {
		name     string
		ctx      context.Context
		manifest string
		wantErr  string
	}{
		{name: "restricted_admin_denied_clusterrolebinding", ctx: restricted, manifest: manifest, wantErr: "集群级资源"},
		{name: "unrestricted_admin_allowed", ctx: unrestricted, manifest: manifest},
		{name: "bypass_allowed", ctx: bypass, manifest: manifest},
		{name: "skip_namespace_policy_allowed", ctx: skip, manifest: manifest},
		{name: "no_scope_allowed_internal", ctx: context.Background(), manifest: manifest},
		{name: "restricted_admin_allowed_namespaced", ctx: restricted, manifest: namespacedManifest},
		{name: "restricted_admin_allowed_namespace_creation", ctx: restricted, manifest: namespaceManifest},
		{name: "multi_doc_one_blocked", ctx: restricted, manifest: namespacedManifest + "---\n" + manifest, wantErr: "ClusterRoleBinding"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := svc.ensureManifestClusterScopesAllowed(tc.ctx, nil, tc.manifest)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v want contains %q", err, tc.wantErr)
			}
		})
	}
}
