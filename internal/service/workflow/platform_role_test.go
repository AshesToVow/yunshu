package workflow

import (
	"testing"

	"yunshu/internal/pkg/auth"
)

func TestCanPlatformRoleReview(t *testing.T) {
	t.Parallel()
	if CanPlatformRoleReview(nil) {
		t.Fatal("nil actor")
	}
	if CanPlatformRoleReview(&auth.CurrentUser{ID: 1, RoleCodes: []string{"developer"}}) {
		t.Fatal("developer should not review")
	}
	if !CanPlatformRoleReview(&auth.CurrentUser{ID: 1, RoleCodes: []string{"ai-approver"}}) {
		t.Fatal("ai-approver should review")
	}
	if !CanPlatformRoleReview(&auth.CurrentUser{ID: 1, RoleCodes: []string{"super-admin"}}) {
		t.Fatal("super-admin should review")
	}
}
