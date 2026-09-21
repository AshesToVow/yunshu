package middleware

import "testing"

func TestK8sScopeForceTierCheck(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "event_forward_settings", method: "PUT", path: "/api/v1/k8s/event-forward/settings", want: false},
		{name: "event_forward_create_rule", method: "POST", path: "/api/v1/k8s/event-forward/rules", want: false},
		{name: "event_forward_delete_rule", method: "DELETE", path: "/api/v1/k8s/event-forward/rules/:id", want: false},
		{name: "pod_exec", method: "POST", path: "/api/v1/pods/exec", want: true},
		{name: "pod_debug", method: "POST", path: "/api/v1/pods/debug", want: true},
		{name: "pod_file_upload", method: "POST", path: "/api/v1/pods/file/upload", want: true},
		{name: "pod_file_delete", method: "POST", path: "/api/v1/pods/file/delete", want: true},
		{name: "ingress_nginx_restart", method: "POST", path: "/api/v1/ingresses/nginx/restart", want: true},
		{name: "pod_exec_ws", method: "GET", path: "/api/v1/pods/exec/ws", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := k8sScopeForceTierCheck(tc.path, tc.method)
			if got != tc.want {
				t.Fatalf("%s %s: got %v want %v", tc.method, tc.path, got, tc.want)
			}
		})
	}
}

func TestK8sRouteIsExecLike(t *testing.T) {
	t.Parallel()
	if !k8sRouteIsExecLike("/api/v1/pods/exec") {
		t.Fatal("exec should match")
	}
	if !k8sRouteIsExecLike("/api/v1/pods/debug") {
		t.Fatal("debug should match")
	}
	if !k8sRouteIsExecLike("/api/v1/pods/file/upload") {
		t.Fatal("file upload should match")
	}
	if k8sRouteIsExecLike("/api/v1/deployments/apply") {
		t.Fatal("apply should not match")
	}
}
