package logplatform

import (
	"testing"

	"yunshu/internal/model"
)

func TestLogSourceToGlobPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  model.ServiceLogSource
		want string
	}{
		{
			name: "exact syslog file without .log suffix",
			src:  model.ServiceLogSource{Path: "/var/log/messages"},
			want: "/var/log/messages",
		},
		{
			name: "exact .log file",
			src:  model.ServiceLogSource{Path: "/var/log/myapp/app.log"},
			want: "/var/log/myapp/app.log",
		},
		{
			name: "user glob kept",
			src:  model.ServiceLogSource{Path: "/var/log/myapp/*.log"},
			want: "/var/log/myapp/*.log",
		},
		{
			name: "dir trailing slash defaults to all files",
			src:  model.ServiceLogSource{Path: "/var/log/myapp/"},
			want: "/var/log/myapp/*",
		},
		{
			name: "dir with include pattern",
			src:  model.ServiceLogSource{Path: "/var/log/kube-apiserver", IncludeRegex: strPtr("*.log")},
			want: "/var/log/kube-apiserver/*.log",
		},
		{
			name: "dir slash with custom include",
			src:  model.ServiceLogSource{Path: "/var/log/", IncludeRegex: strPtr("messages*")},
			want: "/var/log/messages*",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := logSourceToGlobPath(tc.src)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
