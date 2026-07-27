package promapi_test

import (
	"testing"

	"yunshu/internal/pkg/promapi"
)

func TestDeriveAlertmanagerURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"http://10.0.0.1:9090", "http://10.0.0.1:9093"},
		{"http://10.0.0.1:9090/", "http://10.0.0.1:9093"},
		{"http://10.0.0.1:9093", "http://10.0.0.1:9093"},
		{"http://10.0.0.1:8080", ""},
		{"", ""},
	}
	for _, tc := range cases {
		if got := promapi.DeriveAlertmanagerURL(tc.in); got != tc.want {
			t.Fatalf("DeriveAlertmanagerURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
