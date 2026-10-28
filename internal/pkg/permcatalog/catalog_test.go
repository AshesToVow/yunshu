package permcatalog

import (
	"strings"
	"testing"
)

func TestCatalogNoDuplicateRouteKeys(t *testing.T) {
	t.Parallel()
	seen := make(map[string]string)
	for _, e := range Entries() {
		key := Key(e.Method, e.Path)
		if group, ok := seen[key]; ok {
			t.Fatalf("duplicate catalog key %q in groups %q and %q", key, group, e.Group)
		}
		seen[key] = e.Group
	}
}

func TestCatalogEntriesWellFormed(t *testing.T) {
	t.Parallel()
	for _, e := range Entries() {
		if strings.TrimSpace(e.Name) == "" {
			t.Fatalf("catalog entry %s %s has empty name", e.Method, e.Path)
		}
		if !strings.HasPrefix(e.Path, "/api/v1/") {
			t.Fatalf("catalog entry %s %s must start with /api/v1/", e.Method, e.Path)
		}
		if strings.Contains(e.Path, "*") {
			t.Fatalf("catalog entry %s %s must not contain wildcard", e.Method, e.Path)
		}
		switch e.Method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			t.Fatalf("catalog entry %s %s has invalid method", e.Method, e.Path)
		}
	}
}

func TestAllPermissionsMatchGroups(t *testing.T) {
	t.Parallel()
	groups := Groups()
	want := 0
	for _, g := range groups {
		want += len(g.Entries)
	}
	if got := len(AllPermissions()); got != want {
		t.Fatalf("AllPermissions size = %d, want %d", got, want)
	}
	for _, p := range AllPermissions() {
		if _, ok := Lookup(p.Action, p.Resource); !ok {
			t.Fatalf("permission %s %s not resolvable via Lookup", p.Action, p.Resource)
		}
	}
}

func TestPseudoPermissionsResolveFromCatalog(t *testing.T) {
	t.Parallel()
	pseudo := PseudoPermissions()
	if len(pseudo) != 2 {
		t.Fatalf("want 2 pseudo permissions, got %d", len(pseudo))
	}
	for _, p := range pseudo {
		entry, ok := Lookup(p.Action, p.Resource)
		if !ok {
			t.Fatalf("pseudo permission %s %s missing from catalog", p.Action, p.Resource)
		}
		if p.Name != entry.Name {
			t.Fatalf("pseudo permission name = %q, want catalog name %q", p.Name, entry.Name)
		}
		if !IsPseudoResource(p.Resource) {
			t.Fatalf("IsPseudoResource(%q) = false", p.Resource)
		}
	}
	if IsPseudoResource("/api/v1/users") {
		t.Fatal("route resource must not be treated as pseudo")
	}
}

func TestSnapshotIncludesPublicEntries(t *testing.T) {
	t.Parallel()
	doc := Snapshot()
	if doc.Total != len(Entries()) {
		t.Fatalf("snapshot total = %d, want %d", doc.Total, len(Entries()))
	}
	var found bool
	for _, g := range doc.Groups {
		for _, e := range g.Entries {
			if e.Path == "/api/v1/health" && !e.Auth {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("snapshot must include public health route with auth=false")
	}
}
