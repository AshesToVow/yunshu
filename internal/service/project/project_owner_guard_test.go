package project

import (
	"errors"
	"testing"

	"yunshu/internal/model"
	bizerrors "yunshu/internal/pkg/errors"
)

// TestEnsureProjectKeepsOwner 项目必须保留至少一名负责人。
func TestEnsureProjectKeepsOwner(t *testing.T) {
	t.Parallel()

	rows := []model.ProjectMember{
		{ID: 1, UserID: 11, Role: "owner"},
		{ID: 2, UserID: 12, Role: "admin"},
		{ID: 3, UserID: 13, Role: ""},
	}
	tests := []struct {
		name     string
		members  []model.ProjectMember
		changing uint
		wantDeny bool
	}{
		{name: "another owner remains", members: []model.ProjectMember{{ID: 1, Role: "owner"}, {ID: 2, Role: "owner"}}, changing: 1},
		{name: "last owner removed", members: rows, changing: 1, wantDeny: true},
		{name: "other member keeps owner untouched", members: rows, changing: 2},
		{name: "no members left", members: []model.ProjectMember{{ID: 1, Role: "owner"}}, changing: 1, wantDeny: true},
		{name: "role with spaces and case", members: []model.ProjectMember{{ID: 1, Role: " Owner "}, {ID: 2, Role: "member"}}, changing: 2},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ensureProjectKeepsOwner(tc.members, tc.changing)
			if !tc.wantDeny {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("want error, got nil")
			}
			var biz *bizerrors.BizError
			if !errors.As(err, &biz) || biz.Code != 11020 {
				t.Fatalf("want BizError 11020, got %v", err)
			}
		})
	}
}
