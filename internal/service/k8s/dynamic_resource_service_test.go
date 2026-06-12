package k8s

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGvkClusterScoped(t *testing.T) {
	tests := []struct {
		kind     string
		cluster  bool
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
