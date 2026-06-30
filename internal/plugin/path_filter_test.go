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

func TestIsAPIResourceAllowedCicdRequiresProject(t *testing.T) {
	t.Parallel()
	cfg := &config.PluginsConfig{Enabled: []string{"cicd"}}
	if IsAPIResourceAllowed("/api/v1/projects/1/cicd/services", cfg) {
		t.Fatal("cicd API should require project plugin")
	}
	cfg.Enabled = []string{"cicd", "project"}
	if !IsAPIResourceAllowed("/api/v1/projects/1/cicd/services", cfg) {
		t.Fatal("expected allowed when both enabled")
	}
}
