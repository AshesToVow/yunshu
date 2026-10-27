package k8s

import (
	"context"
	"strings"
	"testing"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type fakeSnapshotRepo struct {
	repository.K8sWorkloadSnapshotRepo
	calls int
	got   repository.K8sWorkloadSnapshotListParams
}

func (f *fakeSnapshotRepo) List(_ context.Context, p repository.K8sWorkloadSnapshotListParams) (*pagination.Result[model.K8sWorkloadSnapshot], error) {
	f.calls++
	f.got = p
	return &pagination.Result[model.K8sWorkloadSnapshot]{}, nil
}

type fakeProjectMemberRepo struct {
	interfaces.ProjectMemberRepository
	member bool
	ids    []uint
	err    error
}

func (f *fakeProjectMemberRepo) GetByProjectAndUser(_ context.Context, projectID uint, userID uint) (*model.ProjectMember, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.member {
		return &model.ProjectMember{ProjectID: projectID, UserID: userID}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeProjectMemberRepo) ListProjectIDsByUser(_ context.Context, _ uint) ([]uint, error) {
	return f.ids, f.err
}

func ctxWithUser(u *auth.CurrentUser) context.Context {
	return auth.WithRequestUser(context.Background(), u)
}

func TestListSnapshots_ProjectScoping(t *testing.T) {
	t.Parallel()
	ops := &auth.CurrentUser{ID: 9, RoleCodes: []string{"ops"}}
	super := &auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}}
	cases := []struct {
		name        string
		ctx         context.Context
		query       SnapshotListQuery
		member      *fakeProjectMemberRepo
		wantErr     string
		wantScope   bool
		wantIDs     []uint
		wantProject uint
		wantCalls   int
	}{
		{
			name:      "no_user_rejected",
			ctx:       context.Background(),
			query:     SnapshotListQuery{ClusterID: 1},
			member:    &fakeProjectMemberRepo{},
			wantErr:   "登录已失效",
			wantCalls: 0,
		},
		{
			name:      "super_admin_unfiltered",
			ctx:       ctxWithUser(super),
			query:     SnapshotListQuery{ClusterID: 1},
			member:    &fakeProjectMemberRepo{},
			wantCalls: 1,
		},
		{
			name:      "non_super_cluster_only_scoped_to_memberships",
			ctx:       ctxWithUser(ops),
			query:     SnapshotListQuery{ClusterID: 1},
			member:    &fakeProjectMemberRepo{ids: []uint{2, 3}},
			wantScope: true,
			wantIDs:   []uint{2, 3},
			wantCalls: 1,
		},
		{
			name:      "non_super_without_membership_matches_nothing",
			ctx:       ctxWithUser(ops),
			query:     SnapshotListQuery{ClusterID: 1},
			member:    &fakeProjectMemberRepo{},
			wantScope: true,
			wantCalls: 1,
		},
		{
			name:    "non_super_project_requires_membership",
			ctx:     ctxWithUser(ops),
			query:   SnapshotListQuery{ProjectID: 5},
			member:  &fakeProjectMemberRepo{},
			wantErr: "非项目成员",
		},
		{
			name:        "non_super_project_member_allowed",
			ctx:         ctxWithUser(ops),
			query:       SnapshotListQuery{ProjectID: 5},
			member:      &fakeProjectMemberRepo{member: true},
			wantProject: 5,
			wantCalls:   1,
		},
		{
			name:        "super_admin_project_unfiltered",
			ctx:         ctxWithUser(super),
			query:       SnapshotListQuery{ProjectID: 5},
			member:      &fakeProjectMemberRepo{},
			wantProject: 5,
			wantCalls:   1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repo := &fakeSnapshotRepo{}
			svc := &K8sWorkloadService{snapRepo: repo, memberRepo: tc.member}
			_, err := svc.ListSnapshots(tc.ctx, tc.query)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want contains %q", err, tc.wantErr)
				}
				if repo.calls != 0 {
					t.Fatalf("repo must not be queried when denied, calls=%d", repo.calls)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if repo.calls != tc.wantCalls {
				t.Fatalf("repo calls=%d want %d", repo.calls, tc.wantCalls)
			}
			if repo.got.ProjectScope != tc.wantScope {
				t.Fatalf("project scope=%v want %v", repo.got.ProjectScope, tc.wantScope)
			}
			if len(tc.wantIDs) != len(repo.got.ProjectIDs) {
				t.Fatalf("project ids=%v want %v", repo.got.ProjectIDs, tc.wantIDs)
			}
			for i := range tc.wantIDs {
				if repo.got.ProjectIDs[i] != tc.wantIDs[i] {
					t.Fatalf("project ids=%v want %v", repo.got.ProjectIDs, tc.wantIDs)
				}
			}
			if repo.got.ProjectID != tc.wantProject {
				t.Fatalf("project id=%d want %d", repo.got.ProjectID, tc.wantProject)
			}
		})
	}
}

func TestAssertSnapshotProjectMember(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		svc       *K8sWorkloadService
		projectID uint
		wantErr   string
	}{
		{name: "member", svc: &K8sWorkloadService{memberRepo: &fakeProjectMemberRepo{member: true}}, projectID: 5},
		{name: "non_member", svc: &K8sWorkloadService{memberRepo: &fakeProjectMemberRepo{}}, projectID: 5, wantErr: "非项目成员"},
		{name: "legacy_global_snapshot", svc: &K8sWorkloadService{memberRepo: &fakeProjectMemberRepo{member: true}}, projectID: 0, wantErr: "全局快照"},
		{name: "nil_repo", svc: &K8sWorkloadService{}, projectID: 5, wantErr: "平台服务异常"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.svc.assertSnapshotProjectMember(context.Background(), tc.projectID, 9)
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
