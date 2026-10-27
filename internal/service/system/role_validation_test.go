package system

import (
	"strings"
	"testing"
)

func TestNormalizeRoleCode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "valid", raw: "ops-admin", want: "ops-admin"},
		{name: "trimmed", raw: "  dev_01  ", want: "dev_01"},
		{name: "empty", raw: "   ", wantErr: "不能为空"},
		{name: "reserved_exact", raw: "super-admin", wantErr: "内置保留"},
		{name: "reserved_case", raw: "Super-Admin", wantErr: "内置保留"},
		{name: "reserved_with_space", raw: " super-admin ", wantErr: "内置保留"},
		{name: "uppercase", raw: "OpsAdmin", wantErr: "小写字母"},
		{name: "leading_dash", raw: "-ops", wantErr: "小写字母"},
		{name: "space_inside", raw: "ops admin", wantErr: "小写字母"},
		{name: "too_long", raw: strings.Repeat("a", 65), wantErr: "小写字母"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeRoleCode(tc.raw)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want contains %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("code=%q want %q", got, tc.want)
			}
		})
	}
}

func TestIsBuiltinSuperAdminRole(t *testing.T) {
	t.Parallel()
	if !isBuiltinSuperAdminRole(" super-admin ") {
		t.Fatal("trimmed super-admin should be treated as builtin")
	}
	if isBuiltinSuperAdminRole("Super-Admin") {
		t.Fatal("builtin check is exact-match; normalization happens at write time")
	}
	if isBuiltinSuperAdminRole("ops-admin") {
		t.Fatal("ordinary role must not be treated as builtin")
	}
}
