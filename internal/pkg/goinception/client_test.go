package goinception

import "testing"

func TestEscapeInceptionValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "abc123", want: "abc123"},
		{name: "single quote", in: `a'b`, want: `a\'b`},
		{name: "trailing backslash", in: `pass\`, want: `pass\\`},
		// 反斜杠必须先于单引号转义，否则 \' 会突破 header 的引号闭合。
		{name: "backslash quote", in: `a\'b`, want: `a\\\'b`},
		{name: "empty", in: "", want: ""},
	}
	for _, c := range cases {
		if got := escapeInceptionValue(c.in); got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestIsDDLStatement(t *testing.T) {
	t.Parallel()
	if !isDDLStatement("  alter table t add c int") {
		t.Fatal("ALTER must be DDL")
	}
	if isDDLStatement("UPDATE t SET a=1") {
		t.Fatal("UPDATE is not DDL")
	}
}
