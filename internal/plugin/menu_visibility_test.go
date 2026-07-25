package plugin

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/menu"
)

func TestDesiredMenuStatusBidirectional(t *testing.T) {
	t.Parallel()
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
