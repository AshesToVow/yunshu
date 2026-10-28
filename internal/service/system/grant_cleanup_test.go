package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

type fakeGrantCleaner struct {
	calls []string
	err   error
}

func (f *fakeGrantCleaner) CleanupPrincipalGrants(_ context.Context, kind, ref string) error {
	f.calls = append(f.calls, kind+"/"+ref)
	return f.err
}

func superCtx(ctx context.Context, id uint) context.Context {
	return auth.WithRequestUser(ctx, &auth.CurrentUser{ID: id, Username: "root", RoleCodes: []string{auth.SuperAdminRoleCode}})
}

// --- 用户组删除级联 ---

type fakeUserGroupRepo struct {
	interfaces.UserGroupRepository
	group          *model.UserGroup
	cleanedMembers uint
	saved          bool
	deletedID      uint
}

func (f *fakeUserGroupRepo) GetByID(ctx context.Context, id uint) (*model.UserGroup, error) {
	if f.group == nil || f.group.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return f.group, nil
}

func (f *fakeUserGroupRepo) ReplaceMemberUserIDs(ctx context.Context, id uint, ids []uint) error {
	f.cleanedMembers = id
	return nil
}

func (f *fakeUserGroupRepo) Save(ctx context.Context, g *model.UserGroup) error {
	f.saved = true
	return nil
}

func (f *fakeUserGroupRepo) CountMembers(ctx context.Context, id uint) (int64, error) {
	return 0, nil
}

func (f *fakeUserGroupRepo) Delete(ctx context.Context, g *model.UserGroup) error {
	f.deletedID = g.ID
	return nil
}

func newCascadeTestService(repo *fakeUserGroupRepo, cleaner PrincipalGrantCleaner) *UserGroupService {
	return NewUserGroupService(repo, nil, nil, nil, cleaner)
}

func TestUserGroupDeleteCleansGrantsBeforeDelete(t *testing.T) {
	t.Parallel()

	scope := uint(5)
	repo := &fakeUserGroupRepo{group: &model.UserGroup{ID: 9, Code: "ops-grp", ScopeProjectID: &scope}}
	cleaner := &fakeGrantCleaner{}
	svc := newCascadeTestService(repo, cleaner)

	if err := svc.Delete(superCtx(context.Background(), 1), 9); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(cleaner.calls) != 1 || cleaner.calls[0] != model.K8sPrincipalGroup+"/ops-grp" {
		t.Fatalf("cleaner calls = %v, want [group/ops-grp]", cleaner.calls)
	}
	if repo.cleanedMembers != 9 || repo.deletedID != 9 {
		t.Fatalf("members=%d deleted=%d, want both 9", repo.cleanedMembers, repo.deletedID)
	}
}

func TestUserGroupDeleteAbortsWhenGrantCleanupFails(t *testing.T) {
	t.Parallel()

	repo := &fakeUserGroupRepo{group: &model.UserGroup{ID: 9, Code: "ops-grp"}}
	cleaner := &fakeGrantCleaner{err: errors.New("k8s down")}
	svc := newCascadeTestService(repo, cleaner)

	if err := svc.Delete(superCtx(context.Background(), 1), 9); err == nil {
		t.Fatal("delete should fail when grant cleanup fails")
	}
	if repo.deletedID != 0 || repo.cleanedMembers != 0 {
		t.Fatalf("group must not be deleted on cleanup failure: deleted=%d cleaned=%d", repo.deletedID, repo.cleanedMembers)
	}
}

// --- 项目组提升为全局组 ---

type fakeProjectMemberRepo struct {
	interfaces.ProjectMemberRepository
}

func (f *fakeProjectMemberRepo) GetByProjectAndUser(ctx context.Context, projectID, userID uint) (*model.ProjectMember, error) {
	return &model.ProjectMember{ProjectID: projectID, UserID: userID, Role: "admin"}, nil
}

func TestUserGroupPromotionToGlobalRequiresSuperAdmin(t *testing.T) {
	t.Parallel()

	scope := uint(5)
	zero := uint(0)

	t.Run("super admin passes", func(t *testing.T) {
		t.Parallel()
		repo := &fakeUserGroupRepo{group: &model.UserGroup{ID: 9, Code: "ops-grp", ScopeProjectID: &scope}}
		svc := newCascadeTestService(repo, &fakeGrantCleaner{})
		if _, err := svc.Update(superCtx(context.Background(), 1), 9, UserGroupUpdateRequest{ScopeProjectID: &zero}); err != nil {
			t.Fatalf("super admin promotion should pass: %v", err)
		}
		if !repo.saved || repo.group.ScopeProjectID != nil {
			t.Fatalf("saved=%v scope=%v, want saved and nil scope", repo.saved, repo.group.ScopeProjectID)
		}
	})

	t.Run("project admin rejected", func(t *testing.T) {
		t.Parallel()
		repo := &fakeUserGroupRepo{group: &model.UserGroup{ID: 9, Code: "ops-grp", ScopeProjectID: &scope}}
		svc := NewUserGroupService(repo, nil, &fakeProjectMemberRepo{}, nil, &fakeGrantCleaner{})
		ctx := auth.WithRequestUser(context.Background(), &auth.CurrentUser{ID: 2, Username: "pm", RoleCodes: []string{"pm"}})
		if _, err := svc.Update(ctx, 9, UserGroupUpdateRequest{ScopeProjectID: &zero}); err == nil {
			t.Fatal("non super admin promotion should be rejected")
		} else if !strings.Contains(err.Error(), "仅超级管理员") {
			t.Fatalf("want super-admin-only message, got %v", err)
		}
		if repo.saved {
			t.Fatal("group must not be saved on rejection")
		}
	})

	t.Run("global group demotion unaffected", func(t *testing.T) {
		t.Parallel()
		repo := &fakeUserGroupRepo{group: &model.UserGroup{ID: 9, Code: "ops-grp"}}
		svc := NewUserGroupService(repo, nil, &fakeProjectMemberRepo{}, nil, &fakeGrantCleaner{})
		ctx := auth.WithRequestUser(context.Background(), &auth.CurrentUser{ID: 2, Username: "pm", RoleCodes: []string{"pm"}})
		if _, err := svc.Update(ctx, 9, UserGroupUpdateRequest{ScopeProjectID: &zero}); err != nil {
			t.Fatalf("setting 0 on already-global group should pass: %v", err)
		}
	})
}

// --- 角色删除级联 ---

type fakeRoleRepo struct {
	interfaces.RoleRepository
	role      *model.Role
	deletedID uint
}

func (f *fakeRoleRepo) GetByID(ctx context.Context, id uint) (*model.Role, error) {
	if f.role == nil || f.role.ID != id {
		return nil, gorm.ErrRecordNotFound
	}
	return f.role, nil
}

func (f *fakeRoleRepo) Delete(ctx context.Context, role *model.Role) error {
	f.deletedID = role.ID
	return nil
}

func newRoleDeleteTestService(t *testing.T, repo *fakeRoleRepo, cleaner PrincipalGrantCleaner) *RoleService {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(`
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
	return NewRoleService(repo, nil, enforcer, cleaner)
}

func TestRoleDeleteCleansGrantsBeforeDelete(t *testing.T) {
	t.Parallel()

	repo := &fakeRoleRepo{role: &model.Role{ID: 3, Code: "ops"}}
	cleaner := &fakeGrantCleaner{}
	svc := newRoleDeleteTestService(t, repo, cleaner)

	if err := svc.Delete(superCtx(context.Background(), 1), 3); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(cleaner.calls) != 1 || cleaner.calls[0] != model.K8sPrincipalRole+"/ops" {
		t.Fatalf("cleaner calls = %v, want [role/ops]", cleaner.calls)
	}
	if repo.deletedID != 3 {
		t.Fatalf("role not deleted: %d", repo.deletedID)
	}
}

func TestRoleDeleteAbortsWhenGrantCleanupFails(t *testing.T) {
	t.Parallel()

	repo := &fakeRoleRepo{role: &model.Role{ID: 3, Code: "ops"}}
	cleaner := &fakeGrantCleaner{err: errors.New("dbmgmt down")}
	svc := newRoleDeleteTestService(t, repo, cleaner)

	if err := svc.Delete(superCtx(context.Background(), 1), 3); err == nil {
		t.Fatal("delete should fail when grant cleanup fails")
	}
	if repo.deletedID != 0 {
		t.Fatalf("role must not be deleted on cleanup failure: %d", repo.deletedID)
	}
}

// --- 聚合清理器行为 ---

type stubK8sAccessRepo struct {
	interfaces.K8sClusterAccessRepository
	calls []string
	err   error
}

func (s *stubK8sAccessRepo) DeleteByPrincipal(ctx context.Context, kind, ref string) error {
	s.calls = append(s.calls, kind+"/"+ref)
	return s.err
}

type stubNsRepo struct {
	interfaces.K8sNamespaceAllowRepository
	calls []string
}

func (s *stubNsRepo) DeleteByPrincipal(ctx context.Context, kind, ref string) error {
	s.calls = append(s.calls, kind+"/"+ref)
	return nil
}

type stubNsDenyRepo struct {
	interfaces.K8sNamespaceDenyRepository
	calls []string
}

func (s *stubNsDenyRepo) DeleteByPrincipal(ctx context.Context, kind, ref string) error {
	s.calls = append(s.calls, kind+"/"+ref)
	return nil
}

type stubDbmgmtRepo struct {
	interfaces.DbmgmtRepository
	calls []string
}

func (s *stubDbmgmtRepo) DeleteGrantsByPrincipal(ctx context.Context, kind, ref string) error {
	s.calls = append(s.calls, kind+"/"+ref)
	return nil
}

func TestAggregateCleanerCoversAllDomains(t *testing.T) {
	t.Parallel()

	access := &stubK8sAccessRepo{}
	allow := &stubNsRepo{}
	deny := &stubNsDenyRepo{}
	dbmgmt := &stubDbmgmtRepo{}
	cleaner := NewPrincipalGrantCleaner(access, allow, deny, dbmgmt)

	if err := cleaner.CleanupPrincipalGrants(context.Background(), "group", "ops-grp"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	for name, calls := range map[string][]string{
		"k8s-access": access.calls,
		"ns-allow":   allow.calls,
		"ns-deny":    deny.calls,
		"dbmgmt":     dbmgmt.calls,
	} {
		if len(calls) != 1 || !strings.HasPrefix(calls[0], "group/ops-grp") {
			t.Fatalf("%s calls = %v, want [group/ops-grp]", name, calls)
		}
	}
}

func TestAggregateCleanerAbortsOnFirstError(t *testing.T) {
	t.Parallel()

	access := &stubK8sAccessRepo{err: errors.New("boom")}
	allow := &stubNsRepo{}
	deny := &stubNsDenyRepo{}
	dbmgmt := &stubDbmgmtRepo{}
	cleaner := NewPrincipalGrantCleaner(access, allow, deny, dbmgmt)

	if err := cleaner.CleanupPrincipalGrants(context.Background(), "group", "ops-grp"); err == nil {
		t.Fatal("cleanup should propagate first error")
	}
	if len(allow.calls) != 0 {
		t.Fatalf("downstream domains must not run after failure: %v", allow.calls)
	}
}
