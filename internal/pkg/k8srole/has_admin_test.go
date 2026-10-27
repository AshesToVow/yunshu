package k8srole

import (
	"testing"

	"yunshu/internal/model"
)

func TestHasUnrestrictedAdmin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		grants []model.K8sClusterAccessGrant
		want   bool
	}{
		{name: "empty", grants: nil, want: false},
		{
			name:   "namespace_scoped_admin",
			grants: []model.K8sClusterAccessGrant{{Preset: ClusterAdmin, Namespaces: "app"}},
			want:   false,
		},
		{
			name:   "blacklisted_admin",
			grants: []model.K8sClusterAccessGrant{{Preset: ClusterAdmin, BlacklistNamespaces: "kube-system"}},
			want:   false,
		},
		{
			name:   "readonly_unrestricted_is_not_admin",
			grants: []model.K8sClusterAccessGrant{{Preset: ClusterReadonly}},
			want:   false,
		},
		{
			name:   "unrestricted_admin",
			grants: []model.K8sClusterAccessGrant{{Preset: ClusterAdmin}},
			want:   true,
		},
		{
			name: "one_restricted_one_unrestricted",
			grants: []model.K8sClusterAccessGrant{
				{Preset: ClusterAdmin, Namespaces: "app"},
				{Preset: ClusterAdmin},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasUnrestrictedAdmin(tc.grants); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
