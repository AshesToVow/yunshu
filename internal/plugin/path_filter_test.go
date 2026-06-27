package plugin

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestIsMenuPathAllowedByPlugin(t *testing.T) {
	t.Parallel()
	cfg := &config.PluginsConfig{Enabled: []string{"core"}}
	if IsMenuPathAllowed("/alert-channels", cfg) {
		t.Fatal("alert menu should be hidden when alert plugin disabled")
	}
	if !IsMenuPathAllowed("/users", cfg) {
		t.Fatal("core menu should remain")
	}
}

func TestFilterMenusByPlugins(t *testing.T) {
	t.Parallel()
	cfg := &config.PluginsConfig{Enabled: []string{"core", "alert"}}
	tree := []model.Menu{
		{
			Path: "/alert-notify", Name: "告警",
			Children: []model.Menu{
				{Path: "/alert-channels", Name: "通道"},
			},
		},
		{Path: "/users", Name: "用户"},
	}
	out := FilterMenusByPlugins(tree, cfg)
	if len(out) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(out))
	}
	cfg2 := &config.PluginsConfig{Enabled: []string{"core"}}
	out2 := FilterMenusByPlugins(tree, cfg2)
	if len(out2) != 1 || out2[0].Path != "/users" {
		t.Fatalf("alert branch should be removed, got %+v", out2)
	}
}

func TestResolveAPIResourcePlugin(t *testing.T) {
	t.Parallel()
	if ResolveAPIResourcePlugin("/api/v1/alerts/channels") != "alert" {
		t.Fatal("expected alert plugin")
	}
	if ResolveAPIResourcePlugin("/api/v1/k8s-policies/grant") != "k8s" {
		t.Fatal("k8s-policies belongs to k8s plugin")
	}
}
