package handler

import (
	"testing"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

func newDictTestEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()
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
	return enforcer
}

// TestResolveSensitiveDictAccess 守门：敏感字典读写必须与普通字典分离授权，且默认拒绝（fail closed）。
func TestResolveSensitiveDictAccess(t *testing.T) {
	t.Parallel()

	enforcer := newDictTestEnforcer(t)
	grant := func(role, action string) {
		if _, err := enforcer.AddPolicy(role, sensitiveDictPermissionResource, action); err != nil {
			t.Fatalf("add policy: %v", err)
		}
	}
	grant("dict-reader", "GET")
	grant("dict-editor", "GET")
	grant("dict-editor", "PUT")
	grant("dict-writer", "PUT")
	for _, tc := range []struct {
		userID uint
		role   string
	}{
		{userID: 101, role: "dict-reader"},
		{userID: 102, role: "dict-editor"},
		{userID: 103, role: "plain-user"},
		{userID: 104, role: "dict-writer"},
	} {
		if _, err := enforcer.AddRoleForUser(service.UserSubject(tc.userID), tc.role); err != nil {
			t.Fatalf("add role: %v", err)
		}
	}

	tests := []struct {
		name      string
		enforcer  *casbin.SyncedEnforcer
		actor     *auth.CurrentUser
		hasActor  bool
		wantRead  bool
		wantWrite bool
	}{
		{name: "no actor", enforcer: enforcer, actor: nil, hasActor: false},
		{name: "super admin bypass", enforcer: enforcer, actor: &auth.CurrentUser{ID: 1, RoleCodes: []string{auth.SuperAdminRoleCode}}, hasActor: true, wantRead: true, wantWrite: true},
		{name: "nil enforcer fails closed", enforcer: nil, actor: &auth.CurrentUser{ID: 102, RoleCodes: []string{"dict-editor"}}, hasActor: true},
		{name: "reader can read only", enforcer: enforcer, actor: &auth.CurrentUser{ID: 101, RoleCodes: []string{"dict-reader"}}, hasActor: true, wantRead: true},
		{name: "editor can read and write", enforcer: enforcer, actor: &auth.CurrentUser{ID: 102, RoleCodes: []string{"dict-editor"}}, hasActor: true, wantRead: true, wantWrite: true},
		{name: "writer implies read", enforcer: enforcer, actor: &auth.CurrentUser{ID: 104, RoleCodes: []string{"dict-writer"}}, hasActor: true, wantRead: true, wantWrite: true},
		{name: "plain user denied", enforcer: enforcer, actor: &auth.CurrentUser{ID: 103, RoleCodes: []string{"plain-user"}}, hasActor: true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveSensitiveDictAccess(tc.enforcer, tc.actor, tc.hasActor)
			if got.read != tc.wantRead || got.write != tc.wantWrite {
				t.Fatalf("access = %+v, want read=%v write=%v", got, tc.wantRead, tc.wantWrite)
			}
		})
	}
}
