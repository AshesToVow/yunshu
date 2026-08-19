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
