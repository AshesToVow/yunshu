package model

import "testing"

func TestExtractEnabledRoleCodes(t *testing.T) {
	t.Parallel()
	roles := []Role{
		{Code: "ops", Status: StatusEnabled},
		{Code: "super-admin", Status: StatusDisabled},
		{Code: "  ", Status: StatusEnabled},
		{Code: "viewer", Status: StatusEnabled},
	}
	got := ExtractEnabledRoleCodes(roles)
	if len(got) != 2 || got[0] != "ops" || got[1] != "viewer" {
		t.Fatalf("got %#v", got)
	}
}
