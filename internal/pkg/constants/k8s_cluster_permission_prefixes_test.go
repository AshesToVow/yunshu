package constants

import (
	"strings"
	"testing"
)

func TestK8sClusterPermissionPathPrefixes_NoStalePaths(t *testing.T) {
	t.Parallel()
	stale := []string{
		"/api/v1/k8s-event-forward",
		"/api/v1/component-status",
		"/api/v1/services",
		"/api/v1/ingress-classes",
		"/api/v1/helm/charts",
		"/api/v1/helm/harbor",
		"/api/v1/k8s/event-forward",
		"/api/v1/k8s-policies",
	}
	for _, prefix := range K8sClusterPermissionPathPrefixes {
		for _, s := range stale {
			if prefix == s || strings.HasPrefix(prefix, s+"/") {
				t.Fatalf("cluster scope prefix %q matches stale path %q", prefix, s)
			}
		}
	}
}

func TestIsK8sClusterPermissionResource_EventForwardNotClusterScoped(t *testing.T) {
	t.Parallel()
	if IsK8sClusterPermissionResource("/api/v1/k8s/event-forward/settings") {
		t.Fatal("event-forward is platform config, must not be cluster-scoped")
	}
	if !IsK8sClusterPermissionResource("/api/v1/pods") {
		t.Fatal("pods should be cluster-scoped")
	}
}
