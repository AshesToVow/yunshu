package k8s

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8scaps"
	"yunshu/internal/pkg/k8srole"
	"yunshu/internal/repository"
)

type fakeGrantAccessRepo struct {
	grants []model.K8sClusterAccessGrant
	calls  int
}

func (f *fakeGrantAccessRepo) Upsert(context.Context, *model.K8sClusterAccessGrant) error {
	return nil
}
func (f *fakeGrantAccessRepo) ListGrantsApplyingToCluster(context.Context, uint) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeGrantAccessRepo) ListByPrincipal(context.Context, string, string) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeGrantAccessRepo) ListByRoleCode(context.Context, string) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeGrantAccessRepo) ListForPack(context.Context, k8sauth.PrincipalPack, uint) ([]model.K8sClusterAccessGrant, error) {
	f.calls++
	return f.grants, nil
}
func (f *fakeGrantAccessRepo) DeleteByPrincipalCluster(context.Context, string, string, uint) error {
	return nil
}
func (f *fakeGrantAccessRepo) GetByID(context.Context, uint) (*model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeGrantAccessRepo) DeleteByID(context.Context, uint) error { return nil }
func (f *fakeGrantAccessRepo) EffectiveTier(context.Context, k8sauth.PrincipalPack, uint) int {
	return 0
}
func (f *fakeGrantAccessRepo) EffectiveCapabilities(context.Context, k8sauth.PrincipalPack, uint) []string {
	return nil
}
func (f *fakeGrantAccessRepo) BuildEffectiveTierIndex(context.Context, k8sauth.PrincipalPack) (repository.EffectiveTierIndex, error) {
	return repository.EffectiveTierIndex{}, nil
}
func (f *fakeGrantAccessRepo) HasAnyK8sGrant(context.Context, k8sauth.PrincipalPack) bool {
	return false
}

func grantActorCtx(userID uint, roles, groups []string) context.Context {
	return auth.WithRequestUser(context.Background(), &auth.CurrentUser{
		ID:         userID,
		Username:   "actor",
		RoleCodes:  roles,
		GroupCodes: groups,
	})
}

func newGrantActorService(grants []model.K8sClusterAccessGrant) (*K8sScopedPolicyService, *fakeGrantAccessRepo) {
	repo := &fakeGrantAccessRepo{grants: grants}
	return &K8sScopedPolicyService{accessRepo: repo}, repo
}

func TestEnsureGrantActorAllowedSelfTargets(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		kind string
		ref  string
	}{
		{"self user", model.K8sPrincipalUser, "7"},
		{"own role", model.K8sPrincipalRole, "dev"},
		{"own group", model.K8sPrincipalGroup, "ops"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc, repo := newGrantActorService(nil)
			ctx := grantActorCtx(7, []string{"dev"}, []string{"ops"})
			err := svc.ensureGrantActorAllowed(ctx, tc.kind, tc.ref, []uint{3}, k8scaps.RankReadonly)
			if err == nil || !strings.Contains(err.Error(), "不能给自己") {
				t.Fatalf("expected self-grant rejection, got %v", err)
			}
			if repo.calls != 0 {
				t.Fatalf("rank check should not run before self checks, calls=%d", repo.calls)
			}
		})
	}
}

func TestEnsureGrantActorAllowedAllClustersDenied(t *testing.T) {
	t.Parallel()
	svc, _ := newGrantActorService(nil)
	ctx := grantActorCtx(7, []string{"dev"}, nil)
	err := svc.ensureGrantActorAllowed(ctx, model.K8sPrincipalUser, "9", []uint{0}, k8scaps.RankReadonly)
	if err == nil || !strings.Contains(err.Error(), "全部集群") {
		t.Fatalf("expected all-cluster rejection, got %v", err)
	}
}

func TestEnsureGrantActorAllowedRankCeiling(t *testing.T) {
	t.Parallel()
	readonlyGrants := []model.K8sClusterAccessGrant{{Preset: k8srole.ClusterReadonly}}
	svc, _ := newGrantActorService(readonlyGrants)
	ctx := grantActorCtx(7, []string{"dev"}, nil)

	err := svc.ensureGrantActorAllowed(ctx, model.K8sPrincipalUser, "9", []uint{3}, k8scaps.RankAdmin)
	if err == nil || !strings.Contains(err.Error(), "超过自身权限档位") {
		t.Fatalf("expected rank ceiling rejection, got %v", err)
	}

	if err := svc.ensureGrantActorAllowed(ctx, model.K8sPrincipalUser, "9", []uint{3}, k8scaps.RankReadonly); err != nil {
		t.Fatalf("same-rank grant should pass, got %v", err)
	}
}

func TestEnsureGrantActorAllowedSuperAdminBypass(t *testing.T) {
	t.Parallel()
	svc, repo := newGrantActorService(nil)
	ctx := grantActorCtx(7, []string{auth.SuperAdminRoleCode}, nil)
	err := svc.ensureGrantActorAllowed(ctx, model.K8sPrincipalUser, "7", []uint{0}, k8scaps.RankAdmin)
	if err != nil {
		t.Fatalf("super admin should bypass actor checks, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("super admin bypass should skip rank lookups, calls=%d", repo.calls)
	}
}

func TestEnsureGrantActorAllowedNoActorSkips(t *testing.T) {
	t.Parallel()
	svc, _ := newGrantActorService(nil)
	if err := svc.ensureGrantActorAllowed(context.Background(), model.K8sPrincipalUser, "9", []uint{0}, k8scaps.RankAdmin); err != nil {
		t.Fatalf("background context should skip actor checks, got %v", err)
	}
}

func TestGrantPresetRejectsActorAboveOwnRank(t *testing.T) {
	t.Parallel()
	svc, repo := newGrantActorService([]model.K8sClusterAccessGrant{{Preset: k8srole.ClusterReadonly}})
	ctx := grantActorCtx(7, []string{"dev"}, nil)
	_, err := svc.GrantPreset(ctx, K8sScopedPolicyGrantPresetRequest{
		PrincipalKind: model.K8sPrincipalUser,
		UserID:        9,
		ClusterIDs:    []uint{3},
		Preset:        k8scaps.PresetAdmin,
	})
	if err == nil || !strings.Contains(err.Error(), "超过自身权限档位") {
		t.Fatalf("expected GrantPreset rejection, got %v", err)
	}
	if repo.calls == 0 {
		t.Fatal("expected actor rank lookup")
	}
}
