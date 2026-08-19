package constants

import "testing"

func TestHasPermissionResourceWildcard(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		{in: "/api/v1/pods", want: false},
		{in: "/api/v1/pods/:id", want: false},
		{in: "/api/v1/*", want: true},
		{in: "/api/v1/pods/*", want: true},
		{in: "/*", want: true},
		{in: "*", want: true},
		{in: "", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := HasPermissionResourceWildcard(tc.in); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
