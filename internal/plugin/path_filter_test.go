package plugin

import (
	"testing"

	"yunshu/internal/config"
)

func TestResolveCicdAPIResource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		resource string
		want     string
	}{
		{"/api/v1/overview/project-launches", "cicd"},
		{"/api/v1/overview/release-by-person", "cicd"},
		{"/api/v1/projects/1/cicd/services", "cicd"},
		{"/api/v1/projects/1/members", "project"},
		{"/api/v1/overview", "core"},
	}
	for _, tc := range cases {
		got := ResolveAPIResourcePlugin(tc.resource)
		if got != tc.want {
			t.Fatalf("ResolveAPIResourcePlugin(%q) = %q, want %q", tc.resource, got, tc.want)
		}
	}
}

func TestResolveMenuPathPluginCicd(t *testing.T) {
	t.Parallel()
	if got := ResolveMenuPathPlugin("/cicd/services"); got != "cicd" {
		t.Fatalf("expected cicd, got %q", got)
	}
}

func TestResolveInspectAPIAndMenu(t *testing.T) {
	t.Parallel()
	if got := ResolveAPIResourcePlugin("/api/v1/projects/1/inspect/plan"); got != "inspect" {
		t.Fatalf("expected inspect, got %q", got)
	}
	if got := ResolveMenuPathPlugin("/project-inspect"); got != "inspect" {
		t.Fatalf("expected inspect menu, got %q", got)
	}
	cfg := &config.PluginsConfig{Enabled: []string{"inspect"}}
	if IsMenuPathAllowed("/project-inspect", cfg) {
		t.Fatal("inspect menu should require project plugin")
	}
	cfg.Enabled = []string{"inspect", "project"}
	if !IsMenuPathAllowed("/project-inspect", cfg) {
		t.Fatal("expected inspect menu allowed")
	}
}

