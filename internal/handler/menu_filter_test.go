package handler

import (
	"testing"

	"yunshu/internal/model"
)

func sampleMenuTree() []model.Menu {
	child := func(id uint, path string, status int) model.Menu {
		return model.Menu{ID: id, Path: path, Name: path, Status: status}
	}
	return []model.Menu{
		{ID: 1, Path: "/k8s", Name: "K8s", Status: model.StatusEnabled, Children: []model.Menu{
			child(11, "/k8s/clusters", model.StatusEnabled),
			child(12, "/k8s/offline", model.StatusDisabled),
		}},
		{ID: 2, Path: "/legacy", Name: "旧菜单", Status: model.StatusDisabled, Children: []model.Menu{
			child(21, "/legacy/page", model.StatusEnabled),
		}},
		{ID: 3, Path: "/admin", Name: "管理", Status: model.StatusEnabled, AdminOnly: true},
	}
}

// TestFilterDisabledMenus 守门：停用节点必须连同子树一起消失，启用父节点保留。
func TestFilterDisabledMenus(t *testing.T) {
	t.Parallel()

	got := filterDisabledMenus(sampleMenuTree())

	if len(got) != 2 {
		t.Fatalf("top level count = %d, want 2", len(got))
	}
	if got[0].ID != 1 || got[1].ID != 3 {
		t.Fatalf("top level ids = %d,%d, want 1,3", got[0].ID, got[1].ID)
	}
	if len(got[0].Children) != 1 || got[0].Children[0].ID != 11 {
		t.Fatalf("children of /k8s = %+v, want only menu 11", got[0].Children)
	}
}

// TestFilterAdminOnlyMenus 守门：AdminOnly 节点连同子树移除，普通菜单不受影响。
func TestFilterAdminOnlyMenus(t *testing.T) {
	t.Parallel()

	got := filterAdminOnlyMenus(sampleMenuTree())

	for _, item := range got {
		if item.ID == 3 {
			t.Fatal("admin-only menu 3 should be filtered")
		}
	}
	if len(got) != 2 {
		t.Fatalf("top level count = %d, want 2", len(got))
	}
}

// TestFilterMenusDoesNotMutateInput 守门：过滤基于值拷贝，不得改动原始树（调用方会复用）。
func TestFilterMenusDoesNotMutateInput(t *testing.T) {
	t.Parallel()

	tree := sampleMenuTree()
	_ = filterDisabledMenus(tree)

	if len(tree) != 3 || len(tree[0].Children) != 2 {
		t.Fatalf("input tree mutated: top=%d children=%d", len(tree), len(tree[0].Children))
	}
}
