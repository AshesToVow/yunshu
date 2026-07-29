package plugingate

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/plugin"
)

type stubPlugin struct {
	plugin.Base
	name string
	mf   plugin.Manifest
}

func (s *stubPlugin) Name() string             { return s.name }
func (s *stubPlugin) Manifest() plugin.Manifest { return s.mf }

func TestDesiredMenuStatusBidirectional(t *testing.T) {
	plugin.Register(&stubPlugin{
		name: "core",
		mf:   plugin.Manifest{MenuPathPrefixes: []string{"/users"}},
	})
	plugin.Register(&stubPlugin{
		name: "project",
		mf:   plugin.Manifest{MenuPathPrefixes: []string{"/projects"}},
	})
	plugin.Register(&stubPlugin{
		name: "inspect",
		mf: plugin.Manifest{
			MenuPathPrefixes: []string{"/project-inspect"},
			DependsOn:        []string{"project"},
		},
	})

	catalog := menu.PathStatusMap()

	disabled := &config.PluginsConfig{Enabled: []string{"core", "project"}}
	want, managed := DesiredMenuStatus("/project-inspect", 1, disabled, catalog)
	if !managed || want != 0 {
		t.Fatalf("inspect off: want managed status=0, got managed=%v status=%d", managed, want)
	}

	enabled := &config.PluginsConfig{Enabled: []string{"core", "project", "inspect"}}
	want, managed = DesiredMenuStatus("/project-inspect", 0, enabled, catalog)
	if !managed || want != catalog["/project-inspect"] {
		t.Fatalf("inspect on: want catalog status %d, got managed=%v status=%d", catalog["/project-inspect"], managed, want)
	}

	want, managed = DesiredMenuStatus("/users", 1, disabled, catalog)
	if !managed || want != catalog["/users"] {
		t.Fatalf("core menu: want catalog %d, got managed=%v status=%d", catalog["/users"], managed, want)
	}

	want, managed = DesiredMenuStatus("/custom-unmanaged", 1, enabled, catalog)
	if managed {
		t.Fatalf("unmanaged path should not be managed, got want=%d", want)
	}
}

func TestPathStatusMapIncludesInspect(t *testing.T) {
	t.Parallel()
	m := menu.PathStatusMap()
	if m["/project-inspect"] != 1 {
		t.Fatalf("expected /project-inspect status 1, got %d", m["/project-inspect"])
	}
	if m["/dbmgmt/grants"] != 0 {
		t.Fatalf("hidden /dbmgmt/grants should be 0, got %d", m["/dbmgmt/grants"])
	}
}
