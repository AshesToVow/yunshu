package plugingate

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func ensurePathFilterStubs() {
	plugin.Register(&stubPlugin{
		name: "core",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/users"},
			APIPrefixes:       []string{"/api/v1/overview"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "project",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/projects"},
			APIPrefixes:       []string{"/api/v1/projects"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "cicd",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/cicd"},
			APIPrefixes: []string{
				"/api/v1/overview/project-launches",
				"/api/v1/overview/release-by-person",
				"/api/v1/cicd/jenkins/callback",
				"/api/v1/registries",
				"/api/v1/pipeline-templates",
			},
			DependsOn: []string{"project"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "inspect",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/project-inspect"},
			DependsOn:        []string{"project"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "dbmgmt",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/dbmgmt"},
			DependsOn:        []string{"project"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "backup",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/mysql-backup"},
			DependsOn:        []string{"project"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "alert",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/alert-"},
			APIPrefixes:      []string{"/api/v1/alerts"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "k8s",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/clusters", "/k8s-services"},
			APIPrefixes:      []string{"/api/v1/clusters", "/api/v1/k8s-services", "/api/v1/pods"},
		},
	})
	plugin.Register(&stubPlugin{
		name: "cmdb",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/project-servers"},
			APIPrefixes:      []string{"/api/v1/cloud-accounts", "/api/v1/server-groups"},
			DependsOn:        []string{"project"},
		},
	})
}

func TestResolveCicdAPIResource(t *testing.T) {
	ensurePathFilterStubs()
	cases := []struct {
		resource string
		want     string
	}{
		{"/api/v1/overview/project-launches", "cicd"},
		{"/api/v1/overview/release-by-person", "cicd"},
		{"/api/v1/cicd/jenkins/callback", "cicd"},
		{"/api/v1/registries", "cicd"},
		{"/api/v1/registries/1/ping", "cicd"},
		{"/api/v1/pipeline-templates", "cicd"},
		{"/api/v1/projects/1/registry-binding", "cicd"},
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
	ensurePathFilterStubs()
	if got := ResolveMenuPathPlugin("/cicd/services"); got != "cicd" {
		t.Fatalf("expected cicd, got %q", got)
	}
}

func TestPathMatchesHyphenPrefix(t *testing.T) {
	t.Parallel()
	if !pathMatchesPrefix("/alert-channels", "/alert-") {
		t.Fatal("expected /alert- to match /alert-channels")
	}
	if pathMatchesPrefix("/alerts", "/alert-") {
		t.Fatal("/alert- should not match /alerts")
	}
}

func TestResolveInspectAPIAndMenu(t *testing.T) {
	ensurePathFilterStubs()
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

func TestResolveBackupAPIResource(t *testing.T) {
	t.Parallel()
	got := ResolveAPIResourcePlugin("/api/v1/projects/1/mysql-backup/instances")
	if got != "backup" {
		t.Fatalf("got %q, want backup", got)
	}
}

func TestFilterMenusPromotesBackupWhenDbmgmtDisabled(t *testing.T) {
	ensurePathFilterStubs()
	cfg := &config.PluginsConfig{Enabled: []string{"core", "project", "backup"}}
	menus := []model.Menu{
		{
			Path: "/dbmgmt",
			Name: "数据库管理",
			Children: []model.Menu{
				{Path: "/dbmgmt/instances", Name: "实例管理"},
				{Path: "/mysql-backup", Name: "MySQL 备份"},
			},
		},
	}
	out := FilterMenusByPlugins(menus, cfg)
	if len(out) != 1 || out[0].Path != "/mysql-backup" {
		t.Fatalf("want promoted /mysql-backup, got %+v", out)
	}
}
