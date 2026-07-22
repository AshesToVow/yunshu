package plugingate

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestResolveBackupAPIResource(t *testing.T) {
	t.Parallel()
	got := ResolveAPIResourcePlugin("/api/v1/projects/1/mysql-backup/instances")
	if got != "backup" {
		t.Fatalf("got %q, want backup", got)
	}
}

func TestFilterMenusPromotesBackupWhenDbmgmtDisabled(t *testing.T) {
	t.Parallel()
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
