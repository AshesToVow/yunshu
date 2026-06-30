package cmd

import (
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
