package k8s

import (
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{name: "plain", input: "ls -la /tmp", want: []string{"ls", "-la", "/tmp"}},
		{name: "double quote keeps spaces", input: `sh -c "echo hello world"`, want: []string{"sh", "-c", "echo hello world"}},
		{name: "single quote literal backslash", input: `printf '%s\n' "a b"`, want: []string{"printf", `%s\n`, "a b"}},
		{name: "escaped space outside quotes", input: `echo a\ b`, want: []string{"echo", "a b"}},
		{name: "empty quoted arg", input: `cmd "" x`, want: []string{"cmd", "", "x"}},
		{name: "adjacent quotes merge", input: `cmd a"b"c`, want: []string{"cmd", "abc"}},
		{name: "empty input", input: "", want: nil},
		{name: "unterminated quote", input: `sh -c "echo`, wantErr: true},
		{name: "dangling backslash", input: `echo a\`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := splitCommandLine(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
