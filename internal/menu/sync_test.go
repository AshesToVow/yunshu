package menu

import (
	"testing"

	"yunshu/internal/model"
)

// TestApplySpecToExisting 内置目录同步只覆盖结构字段，管理员维护的展示配置不被重启还原。
func TestApplySpecToExisting(t *testing.T) {
	t.Parallel()

	existing := model.Menu{
		Name:      "自定义名称",
		Icon:      "custom-icon",
		Sort:      99,
		Hidden:    true,
		Status:    0,
		Component: "old-page",
		Redirect:  "/old",
		AdminOnly: false,
	}
	spec := Spec{
		Path:      "/users",
		Name:      "用户管理",
		Icon:      "TeamOutlined",
		Sort:      1,
		Hidden:    false,
		Status:    1,
		Component: "users-page",
		Redirect:  "/users/list",
		AdminOnly: true,
	}

	applySpecToExisting(&existing, spec)

	if existing.Name != "自定义名称" {
		t.Fatalf("name overwritten: %q", existing.Name)
	}
	if existing.Icon != "custom-icon" {
		t.Fatalf("icon overwritten: %q", existing.Icon)
	}
	if existing.Sort != 99 {
		t.Fatalf("sort overwritten: %d", existing.Sort)
	}
	if !existing.Hidden {
		t.Fatal("hidden overwritten")
	}
	if existing.Status != 0 {
		t.Fatalf("status overwritten: %d", existing.Status)
	}
	if existing.Component != "users-page" {
		t.Fatalf("component not synced: %q", existing.Component)
	}
	if existing.Redirect != "/users/list" {
		t.Fatalf("redirect not synced: %q", existing.Redirect)
	}
	if !existing.AdminOnly {
		t.Fatal("admin_only not synced")
	}
}
