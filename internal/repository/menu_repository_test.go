package repository

import (
	"testing"

	"yunshu/internal/model"
)

func uintPtr(v uint) *uint { return &v }

func TestBuildMenuTree_threeLevels(t *testing.T) {
	menus := []model.Menu{
		{ID: 1, Path: "/a", Name: "A", Sort: 1},
		{ID: 2, ParentID: uintPtr(1), Path: "/a/b", Name: "B", Sort: 1},
		{ID: 3, ParentID: uintPtr(2), Path: "/a/b/c", Name: "C", Sort: 1},
		{ID: 4, Path: "/x", Name: "X", Sort: 2},
	}

	tree := buildMenuTree(menus)
	if len(tree) != 2 {
		t.Fatalf("root count = %d, want 2", len(tree))
	}
	if tree[0].Path != "/a" || len(tree[0].Children) != 1 {
		t.Fatalf("unexpected root A: %+v", tree[0])
	}
	if tree[0].Children[0].Path != "/a/b" || len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("unexpected child B: %+v", tree[0].Children[0])
	}
	if tree[0].Children[0].Children[0].Path != "/a/b/c" {
		t.Fatalf("unexpected grandchild C: %+v", tree[0].Children[0].Children[0])
	}
	if tree[1].Path != "/x" || len(tree[1].Children) != 0 {
		t.Fatalf("unexpected root X: %+v", tree[1])
	}
}
