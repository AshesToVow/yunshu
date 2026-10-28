package system

import (
	"context"
	"errors"
	"testing"

	"yunshu/internal/model"
)

// TestMenuPathConflict 菜单路径唯一性：trim 后比对，忽略自身与空路径。
func TestMenuPathConflict(t *testing.T) {
	t.Parallel()

	all := []model.Menu{
		{ID: 1, Path: "/users"},
		{ID: 2, Path: " /reports "},
		{ID: 3, Path: ""},
	}
	tests := []struct {
		name      string
		path      string
		excludeID uint
		want      bool
	}{
		{name: "exact hit", path: "/users", want: true},
		{name: "hit after trim", path: "/reports", want: true},
		{name: "input with spaces", path: "  /users  ", want: true},
		{name: "self excluded", path: "/users", excludeID: 1, want: false},
		{name: "empty path never conflicts", path: "   ", want: false},
		{name: "new path", path: "/audit", want: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := menuPathConflict(all, tc.path, tc.excludeID); got != tc.want {
				t.Fatalf("menuPathConflict(%q, %d) = %v, want %v", tc.path, tc.excludeID, got, tc.want)
			}
		})
	}
}

// TestBuildMenuBindings 绑定校验：动作白名单、mode、重复项、权限点必须真实存在。
func TestBuildMenuBindings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	allowAll := func(context.Context, string, string) (bool, error) { return true, nil }
	denyAll := func(context.Context, string, string) (bool, error) { return false, nil }

	t.Run("normalized binding", func(t *testing.T) {
		t.Parallel()
		out, err := buildMenuBindings(ctx, []MenuPermissionBindingItem{
			{Resource: " /api/v1/users ", Action: " get ", Mode: "ALL"},
		}, allowAll)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out) != 1 || out[0].Resource != "/api/v1/users" || out[0].Action != "GET" || out[0].Mode != "all" {
			t.Fatalf("unexpected binding: %+v", out)
		}
	})

	t.Run("default mode any", func(t *testing.T) {
		t.Parallel()
		out, err := buildMenuBindings(ctx, []MenuPermissionBindingItem{
			{Resource: "/api/v1/users", Action: "GET"},
		}, allowAll)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out[0].Mode != "any" {
			t.Fatalf("mode = %q, want any", out[0].Mode)
		}
	})

	t.Run("nil lookup skips existence check", func(t *testing.T) {
		t.Parallel()
		if _, err := buildMenuBindings(ctx, []MenuPermissionBindingItem{
			{Resource: "/api/v1/users", Action: "GET"},
		}, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	errCases := []struct {
		name  string
		items []MenuPermissionBindingItem
		look  menuPermissionExists
	}{
		{name: "empty resource", items: []MenuPermissionBindingItem{{Resource: "  ", Action: "GET"}}, look: allowAll},
		{name: "wildcard resource", items: []MenuPermissionBindingItem{{Resource: "/api/*", Action: "GET"}}, look: allowAll},
		{name: "resource without slash", items: []MenuPermissionBindingItem{{Resource: "api/v1/users", Action: "GET"}}, look: allowAll},
		{name: "invalid action", items: []MenuPermissionBindingItem{{Resource: "/api/v1/users", Action: "FETCH"}}, look: allowAll},
		{name: "invalid mode", items: []MenuPermissionBindingItem{{Resource: "/api/v1/users", Action: "GET", Mode: "some"}}, look: allowAll},
		{name: "duplicate", items: []MenuPermissionBindingItem{
			{Resource: "/api/v1/users", Action: "GET"},
			{Resource: " /api/v1/users ", Action: "get "},
		}, look: allowAll},
		{name: "missing permission", items: []MenuPermissionBindingItem{{Resource: "/api/v1/users", Action: "GET"}}, look: denyAll},
	}
	for _, tc := range errCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildMenuBindings(ctx, tc.items, tc.look)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			assertBizCode(t, err, 11020)
		})
	}

	t.Run("lookup failure propagates", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("db down")
		_, err := buildMenuBindings(ctx, []MenuPermissionBindingItem{
			{Resource: "/api/v1/users", Action: "GET"},
		}, func(context.Context, string, string) (bool, error) { return false, boom })
		if !errors.Is(err, boom) {
			t.Fatalf("want wrapped lookup error, got %v", err)
		}
	})
}
