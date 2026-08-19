package k8s

import "testing"

func TestIsK8sClusterGrantReadBypassPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "pods_list", path: "/api/v1/pods", want: true},
		{name: "namespaces", path: "/api/v1/namespaces", want: true},
		{name: "secrets_excluded", path: "/api/v1/secrets", want: false},
		{name: "secret_detail_excluded", path: "/api/v1/secrets/detail", want: false},
		{name: "pod_exec_excluded", path: "/api/v1/pods/exec", want: false},
		{name: "pod_exec_ws_excluded", path: "/api/v1/pods/exec/ws", want: false},
		{name: "pod_file_excluded", path: "/api/v1/pods/file", want: false},
		{name: "policies_matrix_excluded", path: "/api/v1/k8s-policies/cluster-auth-matrix", want: false},
		{name: "policies_user_cluster_excluded", path: "/api/v1/k8s-policies/user-cluster-auth", want: false},
		{name: "policies_list_excluded", path: "/api/v1/k8s-policies", want: false},
		{name: "helm_harbor_excluded", path: "/api/v1/helm/harbor/charts", want: false},
		{name: "empty", path: "", want: false},
		{name: "non_k8s", path: "/api/v1/users", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsK8sClusterGrantReadBypassPath(tc.path); got != tc.want {
				t.Fatalf("path=%q got %v want %v", tc.path, got, tc.want)
			}
		})
	}
}
