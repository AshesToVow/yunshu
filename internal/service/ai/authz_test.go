package ai

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/repository"
	"yunshu/internal/service/k8s"
)

type fakeClusterAccess struct {
	rankByCluster map[uint]int
}

func (f *fakeClusterAccess) ListByPrincipal(context.Context, string, string) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeClusterAccess) ListByRoleCode(context.Context, string) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeClusterAccess) ListGrantsApplyingToCluster(context.Context, uint) ([]model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeClusterAccess) GetByID(context.Context, uint) (*model.K8sClusterAccessGrant, error) {
	return nil, nil
}
func (f *fakeClusterAccess) Upsert(context.Context, *model.K8sClusterAccessGrant) error { return nil }
func (f *fakeClusterAccess) DeleteByID(context.Context, uint) error                      { return nil }
func (f *fakeClusterAccess) EffectiveTier(_ context.Context, _ k8sauth.PrincipalPack, clusterID uint) int {
	if f == nil || f.rankByCluster == nil {
		return 0
	}
	return f.rankByCluster[clusterID]
}
func (f *fakeClusterAccess) BuildEffectiveTierIndex(context.Context, k8sauth.PrincipalPack) (repository.EffectiveTierIndex, error) {
	return repository.EffectiveTierIndex{}, nil
}
func (f *fakeClusterAccess) HasAnyK8sGrant(context.Context, k8sauth.PrincipalPack) bool { return false }

func TestScriptToolRequiresApproval(t *testing.T) {
	t.Parallel()
	cases := []struct {
		risk, perm string
		want       bool
	}{
		{"LOW", "READ_ONLY", false},
		{"MEDIUM", "READ_ONLY", false},
		{"HIGH", "READ_ONLY", true},
		{"CRITICAL", "READ_ONLY", true},
		{"LOW", "WRITE", true},
		{"low", "write", true},
	}
	for _, tc := range cases {
		if got := scriptToolRequiresApproval(tc.risk, tc.perm); got != tc.want {
			t.Fatalf("risk=%s perm=%s got %v want %v", tc.risk, tc.perm, got, tc.want)
		}
	}
}

func TestAssertK8sClusterAccess_DenyWithoutGrant(t *testing.T) {
	t.Parallel()
	s := &Service{accessRepo: &fakeClusterAccess{rankByCluster: map[uint]int{3: 0}}}
	actor := &auth.CurrentUser{ID: 7, RoleCodes: []string{"developer"}}
	err := s.assertK8sClusterAccess(context.Background(), actor, 3, "", k8s.K8sAccessRankReadonly)
	if err == nil {
		t.Fatal("expected forbidden")
	}
	if !strings.Contains(err.Error(), "权限不足") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAssertK8sClusterAccess_AllowWithReadonly(t *testing.T) {
	t.Parallel()
	s := &Service{accessRepo: &fakeClusterAccess{rankByCluster: map[uint]int{
		3: k8s.K8sAccessRankReadonly,
	}}}
	actor := &auth.CurrentUser{ID: 7, RoleCodes: []string{"developer"}}
	if err := s.assertK8sClusterAccess(context.Background(), actor, 3, "", k8s.K8sAccessRankReadonly); err != nil {
		t.Fatal(err)
	}
}

func TestAssertK8sClusterAccess_SuperAdminSkipsTier(t *testing.T) {
	t.Parallel()
	s := &Service{accessRepo: &fakeClusterAccess{rankByCluster: map[uint]int{}}}
	actor := &auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}}
	if err := s.assertK8sClusterAccess(context.Background(), actor, 9, "", k8s.K8sAccessRankAdmin); err != nil {
		t.Fatal(err)
	}
}

func TestAssertK8sClusterAccess_NilRepoFailClosed(t *testing.T) {
	t.Parallel()
	s := &Service{}
	actor := &auth.CurrentUser{ID: 2, RoleCodes: []string{"ops"}}
	if err := s.assertK8sClusterAccess(context.Background(), actor, 1, "", k8s.K8sAccessRankReadonly); err == nil {
		t.Fatal("expected error when accessRepo nil")
	}
}
