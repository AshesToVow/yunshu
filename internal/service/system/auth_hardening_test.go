package system

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

func TestLoginLockSubject(t *testing.T) {
	t.Parallel()
	if got := loginLockSubject(0); got != "" {
		t.Fatalf("zero user id subject = %q, want empty", got)
	}
	if got := loginLockSubject(42); got != "uid:42" {
		t.Fatalf("subject = %q, want uid:42", got)
	}
}

func TestLoginLogListParamsScoping(t *testing.T) {
	t.Parallel()
	query := LoginLogListQuery{Username: "someone-else", Status: intPtr(1), Page: 3, PageSize: 20}

	if _, err := loginLogListParams(nil, query, 1, 10); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("nil actor err = %v, want unauthorized", err)
	}

	superParams, err := loginLogListParams(&auth.CurrentUser{ID: 1, Username: "root", RoleCodes: []string{"super-admin"}}, query, 1, 10)
	if err != nil {
		t.Fatalf("super params err = %v", err)
	}
	if superParams.Username != "someone-else" || superParams.UsernameExact != "" {
		t.Fatalf("super params = %+v, want query username and no exact filter", superParams)
	}

	normalParams, err := loginLogListParams(&auth.CurrentUser{ID: 2, Username: "alice", RoleCodes: []string{"ops"}}, query, 1, 10)
	if err != nil {
		t.Fatalf("normal params err = %v", err)
	}
	if normalParams.UsernameExact != "alice" || normalParams.Username != "" {
		t.Fatalf("normal params = %+v, want exact self filter", normalParams)
	}
}

func TestOperationLogListParamsScoping(t *testing.T) {
	t.Parallel()
	query := OperationLogListQuery{Path: "/api/v1/users", Page: 2, PageSize: 50}

	if _, err := operationLogListParams(nil, query, 1, 10); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("nil actor err = %v, want unauthorized", err)
	}

	superParams, err := operationLogListParams(&auth.CurrentUser{ID: 1, Username: "root", RoleCodes: []string{"super-admin"}}, query, 1, 10)
	if err != nil {
		t.Fatalf("super params err = %v", err)
	}
	if superParams.UsernameExact != "" {
		t.Fatalf("super params = %+v, want no exact filter", superParams)
	}

	normalParams, err := operationLogListParams(&auth.CurrentUser{ID: 2, Username: "bob", RoleCodes: []string{"ops"}}, query, 1, 10)
	if err != nil {
		t.Fatalf("normal params err = %v", err)
	}
	if normalParams.UsernameExact != "bob" {
		t.Fatalf("normal params = %+v, want exact self filter", normalParams)
	}
}

func TestRequireSuperAdminForLogMutation(t *testing.T) {
	t.Parallel()
	if err := requireSuperAdminForLogMutation(nil, "删除日志"); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("nil actor err = %v, want unauthorized", err)
	}
	err := requireSuperAdminForLogMutation(&auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}}, "删除日志")
	if err == nil || !strings.Contains(err.Error(), "内置超级管理员") {
		t.Fatalf("non-super err = %v, want forbidden message", err)
	}
	if err := requireSuperAdminForLogMutation(&auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}}, "删除日志"); err != nil {
		t.Fatalf("super err = %v, want nil", err)
	}
}

type stubUserCountRepo struct {
	interfaces.UserRepository
	count int64
	err   error
}

func (s stubUserCountRepo) CountEnabledUsersByRoleCode(context.Context, string) (int64, error) {
	return s.count, s.err
}

func TestEnsureSuperAdminRemains(t *testing.T) {
	t.Parallel()
	target := &model.User{ID: 7, Status: model.StatusEnabled, Roles: []model.Role{{Code: auth.SuperAdminRoleCode, Status: model.StatusEnabled}}}

	if err := (&UserService{userRepo: stubUserCountRepo{count: 1}}).ensureSuperAdminRemains(context.Background(), target); err == nil || !strings.Contains(err.Error(), "至少需要保留") {
		t.Fatalf("last super err = %v, want forbidden", err)
	}
	if err := (&UserService{userRepo: stubUserCountRepo{count: 2}}).ensureSuperAdminRemains(context.Background(), target); err != nil {
		t.Fatalf("with backup super err = %v, want nil", err)
	}

	disabled := *target
	disabled.Status = model.StatusDisabled
	if err := (&UserService{userRepo: stubUserCountRepo{count: 1}}).ensureSuperAdminRemains(context.Background(), &disabled); err != nil {
		t.Fatalf("disabled target err = %v, want nil", err)
	}

	normal := &model.User{ID: 8, Status: model.StatusEnabled, Roles: []model.Role{{Code: "ops", Status: model.StatusEnabled}}}
	if err := (&UserService{userRepo: stubUserCountRepo{count: 1}}).ensureSuperAdminRemains(context.Background(), normal); err != nil {
		t.Fatalf("non-super target err = %v, want nil", err)
	}
}

func TestUserGroupScopedAccessFailsClosed(t *testing.T) {
	t.Parallel()
	projectID := uint(9)
	group := &model.UserGroup{ID: 1, ScopeProjectID: &projectID}

	if err := (&UserGroupService{}).ensureVisibleScopedGroup(context.Background(), group); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("visible without user err = %v, want unauthorized", err)
	}
	if err := (&UserGroupService{}).ensureGroupManageable(context.Background(), group); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("manage without user err = %v, want unauthorized", err)
	}
}

type stubPermissionRepo struct {
	interfaces.PermissionRepository
	permission *model.Permission
}

func (s stubPermissionRepo) GetByID(context.Context, uint) (*model.Permission, error) {
	return s.permission, nil
}

func TestPermissionUpdateByActorRestrictsIdentityChanges(t *testing.T) {
	t.Parallel()
	permission := &model.Permission{ID: 5, Resource: "/api/v1/users", Action: "GET", K8sScopeEnabled: false}
	svc := &PermissionService{permissionRepo: stubPermissionRepo{permission: permission}}
	actor := &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}}

	if _, err := svc.UpdateByActor(context.Background(), nil, 5, PermissionUpdateRequest{}); !bizerrors.Is(err, constants.ErrUnauthorized) {
		t.Fatalf("nil actor err = %v, want unauthorized", err)
	}

	resource := "/api/v1/roles"
	if _, err := svc.UpdateByActor(context.Background(), actor, 5, PermissionUpdateRequest{Resource: &resource}); err == nil || !strings.Contains(err.Error(), "权限资源与方法") {
		t.Fatalf("resource rewrite err = %v, want forbidden", err)
	}

	scope := true
	if _, err := svc.UpdateByActor(context.Background(), actor, 5, PermissionUpdateRequest{K8sScopeEnabled: &scope}); err == nil || !strings.Contains(err.Error(), "K8s 范围校验") {
		t.Fatalf("k8s scope toggle err = %v, want forbidden", err)
	}
}

func intPtr(v int) *int { return &v }
