package system

import (
	"reflect"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apiroute"
)

func TestBuildRoutePolicyDiff(t *testing.T) {
	t.Parallel()

	routes := []apiroute.Entry{
		{Method: "get", Path: "/api/v1/users"},
		{Method: "POST", Path: "/api/v1/users"},
		{Method: "GET", Path: "/api/v1/roles"},
		{Method: "GET", Path: "/api/v1/health"},
	}
	permissions := []model.Permission{
		{Resource: "/api/v1/users", Action: "GET"},
		{Resource: "/api/v1/users", Action: "POST"},
		{Resource: "/api/v1/roles", Action: "GET"},
		// 已下线接口的残留权限点
		{Resource: "/api/v1/legacy", Action: "GET"},
		// 伪权限不参与比对
		{Resource: "menu:dashboard", Action: "GET"},
	}
	policies := [][]string{
		{"ops", "/api/v1/users", "GET"},
		{"ops", "/api/v1/users", "GET"}, // 重复行只报一次
		{"ops", "/api/v1/legacy", "GET"},
		{"ops", "/api/v1/granted-but-no-perm", "GET"},
		{"ops", "menu:dashboard", "GET"}, // 非 "/" 资源跳过
		{"broken"},
	}

	got := BuildRoutePolicyDiff(routes, permissions, policies)

	if want := []string{"GET /api/v1/health"}; !reflect.DeepEqual(got.RouteWithoutPermission, want) {
		t.Fatalf("RouteWithoutPermission = %v, want %v", got.RouteWithoutPermission, want)
	}
	if want := []string{"GET /api/v1/legacy"}; !reflect.DeepEqual(got.PermissionWithoutRoute, want) {
		t.Fatalf("PermissionWithoutRoute = %v, want %v", got.PermissionWithoutRoute, want)
	}
	// /api/v1/legacy 仍有权限点（只是路由没了），归入 permission_without_route，不算孤儿策略。
	wantPolicies := []string{
		"ops -> GET /api/v1/granted-but-no-perm",
	}
	if !reflect.DeepEqual(got.PolicyWithoutPermission, wantPolicies) {
		t.Fatalf("PolicyWithoutPermission = %v, want %v", got.PolicyWithoutPermission, wantPolicies)
	}
	if got.IsEmpty() {
		t.Fatal("IsEmpty should be false when diff exists")
	}
}

func TestBuildRoutePolicyDiffClean(t *testing.T) {
	t.Parallel()

	routes := []apiroute.Entry{{Method: "GET", Path: "/api/v1/users"}}
	permissions := []model.Permission{{Resource: "/api/v1/users", Action: "GET"}}
	policies := [][]string{{"ops", "/api/v1/users", "GET"}}

	got := BuildRoutePolicyDiff(routes, permissions, policies)
	if !got.IsEmpty() {
		t.Fatalf("want clean diff, got %+v", got)
	}
	if samples := got.Samples(10); len(samples) != 3 {
		t.Fatalf("Samples should always return 3 categories, got %d", len(samples))
	}
}

func TestRoutePolicyDiffSamplesTruncate(t *testing.T) {
	t.Parallel()

	diff := RoutePolicyDiff{RouteWithoutPermission: []string{"A", "B", "C"}}
	if got := diff.Samples(2)["route_without_permission"]; !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("truncate = %v, want [A B]", got)
	}
	if got := diff.Samples(0)["route_without_permission"]; len(got) != 3 {
		t.Fatalf("n<=0 should keep all, got %v", got)
	}
}
