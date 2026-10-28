package system

import (
	"errors"
	"testing"

	"yunshu/internal/pkg/auth"
	bizerrors "yunshu/internal/pkg/errors"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

func currentUser(t *testing.T, id uint, roleCodes ...string) *auth.CurrentUser {
	t.Helper()
	return &auth.CurrentUser{ID: id, Username: "tester", RoleCodes: roleCodes}
}

// TestAuthorizeRoleTarget 守门：非超管不得改动自己所属角色（防自我提权），超管不受限。
func TestAuthorizeRoleTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		actor    *auth.CurrentUser
		target   string
		wantDeny bool
		wantCode int
	}{
		{name: "nil actor", actor: nil, target: "ops", wantDeny: true, wantCode: 10002},
		{name: "super admin any role", actor: currentUser(t, 1, auth.SuperAdminRoleCode), target: auth.SuperAdminRoleCode, wantDeny: false},
		{name: "own role denied", actor: currentUser(t, 2, "ops", "dev"), target: "ops", wantDeny: true, wantCode: 11022},
		{name: "own role with spaces", actor: currentUser(t, 2, " ops "), target: "ops", wantDeny: true, wantCode: 11022},
		{name: "other role allowed", actor: currentUser(t, 2, "ops"), target: "dev", wantDeny: false},
		{name: "empty target", actor: currentUser(t, 2, "ops"), target: "  ", wantDeny: true, wantCode: 11020},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := authorizeRoleTarget(tc.actor, tc.target)
			if !tc.wantDeny {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, got nil")
			}
			assertBizCode(t, err, tc.wantCode)
		})
	}
}

// TestAuthorizeRolePolicyChange 守门：非超管只能授予/回收自己已持有的权限。
func TestAuthorizeRolePolicyChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		actor    *auth.CurrentUser
		target   string
		holds    bool
		wantDeny bool
		wantCode int
	}{
		{name: "nil actor", actor: nil, target: "ops", holds: true, wantDeny: true, wantCode: 10002},
		{name: "super admin bypass holds", actor: currentUser(t, 1, auth.SuperAdminRoleCode), target: "ops", holds: false, wantDeny: false},
		{name: "own role denied even if holds", actor: currentUser(t, 2, "ops"), target: "ops", holds: true, wantDeny: true, wantCode: 11022},
		{name: "other role without permission denied", actor: currentUser(t, 2, "ops"), target: "dev", holds: false, wantDeny: true, wantCode: 11022},
		{name: "other role with permission allowed", actor: currentUser(t, 2, "ops"), target: "dev", holds: true, wantDeny: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := authorizeRolePolicyChange(tc.actor, tc.target, tc.holds)
			if !tc.wantDeny {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, got nil")
			}
			assertBizCode(t, err, tc.wantCode)
		})
	}
}

// TestAuthorizePermissionCatalogChange 守门：权限点增删仅内置超管。
func TestAuthorizePermissionCatalogChange(t *testing.T) {
	t.Parallel()

	if err := authorizePermissionCatalogChange(nil, "新增"); err == nil {
		t.Fatal("nil actor should be rejected")
	}
	if err := authorizePermissionCatalogChange(currentUser(t, 2, "perm-admin"), "删除"); err == nil {
		t.Fatal("non super admin should be rejected")
	} else {
		assertBizCode(t, err, 11022)
	}
	if err := authorizePermissionCatalogChange(currentUser(t, 1, auth.SuperAdminRoleCode), "删除"); err != nil {
		t.Fatalf("super admin should pass, got %v", err)
	}
}

// TestActorHoldsPermission 校验 Enforce 主体构造：用户 → 角色 → 策略链路。
func TestActorHoldsPermission(t *testing.T) {
	t.Parallel()

	m, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
`)
	if err != nil {
		t.Fatalf("build model: %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("new enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("ops", "/api/v1/users", "GET"); err != nil {
		t.Fatalf("add policy: %v", err)
	}
	if _, err := enforcer.AddRoleForUser(UserSubject(7), "ops"); err != nil {
		t.Fatalf("add role: %v", err)
	}

	op := currentUser(t, 7, "ops")
	if !actorHoldsPermission(enforcer, op, " /api/v1/users ", "get") {
		t.Fatal("ops should hold GET /api/v1/users (trim + upper normalization)")
	}
	if actorHoldsPermission(enforcer, op, "/api/v1/users", "POST") {
		t.Fatal("ops should not hold POST /api/v1/users")
	}
	if actorHoldsPermission(enforcer, currentUser(t, 8, "dev"), "/api/v1/users", "GET") {
		t.Fatal("user without role should not hold permission")
	}
	if actorHoldsPermission(nil, op, "/api/v1/users", "GET") {
		t.Fatal("nil enforcer should deny")
	}
}

func assertBizCode(t *testing.T, err error, want int) {
	t.Helper()
	var biz *bizerrors.BizError
	if !errors.As(err, &biz) {
		t.Fatalf("want BizError with code %d, got %T %v", want, err, err)
	}
	if biz.Code != want {
		t.Fatalf("code = %d, want %d", biz.Code, want)
	}
}
