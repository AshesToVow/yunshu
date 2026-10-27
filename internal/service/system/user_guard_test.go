package system

import (
	"strings"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

func TestEnsureAssignableRoles(t *testing.T) {
	t.Parallel()
	ops := model.Role{ID: 1, Code: " ops "}
	admin := model.Role{ID: 2, Code: "admin"}
	cases := []struct {
		name    string
		actor   *auth.CurrentUser
		roles   []model.Role
		wantErr string
	}{
		{
			name:  "super_admin_assigns_anything",
			actor: &auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}},
			roles: []model.Role{admin},
		},
		{
			name:  "non_super_assigns_own_role",
			actor: &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}},
			roles: []model.Role{ops},
		},
		{
			name:    "non_super_cannot_escalate",
			actor:   &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}},
			roles:   []model.Role{admin},
			wantErr: "超出自身权限范围",
		},
		{
			name:    "nil_actor",
			actor:   nil,
			roles:   nil,
			wantErr: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ensureAssignableRoles(tc.actor, tc.roles)
			if tc.actor == nil {
				if err == nil {
					t.Fatal("nil actor must be rejected")
				}
				return
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v want contains %q", err, tc.wantErr)
			}
		})
	}
}

func TestEnsureTargetManageable(t *testing.T) {
	t.Parallel()
	superTarget := &model.User{ID: 1, Roles: []model.Role{{Code: "super-admin"}}}
	normalTarget := &model.User{ID: 2, Roles: []model.Role{{Code: "ops"}}}
	cases := []struct {
		name    string
		actor   *auth.CurrentUser
		target  *model.User
		wantErr string
	}{
		{
			name:   "super_admin_manages_super_admin",
			actor:  &auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}},
			target: superTarget,
		},
		{
			name:    "non_super_cannot_touch_super_admin",
			actor:   &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}},
			target:  superTarget,
			wantErr: "仅内置超级管理员",
		},
		{
			name:   "non_super_manages_normal_user",
			actor:  &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}},
			target: normalTarget,
		},
		{
			name:    "nil_target",
			actor:   &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}},
			target:  nil,
			wantErr: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := (&UserService{}).ensureTargetManageable(tc.actor, tc.target)
			if tc.target == nil {
				if err == nil {
					t.Fatal("nil target must be rejected")
				}
				return
			}
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v want contains %q", err, tc.wantErr)
			}
		})
	}
}
