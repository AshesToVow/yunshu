package cmd

import (
	"strings"
	"testing"
)

func TestDefaultPermissionsDisjointFromStale(t *testing.T) {
	t.Parallel()
	stale := make(map[string]struct{})
	for _, p := range stalePermissions() {
		stale[p.Resource+"\x00"+p.Action] = struct{}{}
	}
	for _, p := range defaultPermissions() {
		key := p.Resource + "\x00" + p.Action
		if _, ok := stale[key]; ok {
			t.Fatalf("seed permission still in stale list: %s %s", p.Action, p.Resource)
		}
	}
}

func TestDefaultPermissionsNoWildcardOrForcedK8sScope(t *testing.T) {
	t.Parallel()
	for _, p := range defaultPermissions() {
		if strings.Contains(p.Resource, "*") {
			t.Fatalf("seed permission contains wildcard: %s %s", p.Action, p.Resource)
		}
		if p.K8sScopeEnabled {
			t.Fatalf("seed must not force k8s_scope_enabled: %s %s", p.Action, p.Resource)
		}
		if strings.Contains(strings.ToLower(p.Description), "k8s-scope=") {
			t.Fatalf("seed description must not embed k8s-scope tag: %s %s %q", p.Action, p.Resource, p.Description)
		}
	}
}

func TestStalePermissionsNoDuplicates(t *testing.T) {
	t.Parallel()
	seen := make(map[string]struct{})
	for _, p := range stalePermissions() {
		key := p.Resource + "\x00" + p.Action
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate stale permission: %s %s", p.Action, p.Resource)
		}
		seen[key] = struct{}{}
	}
}
