package cmdb

import "testing"

func TestValidateServerGroupCategory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in    string
		wantOK bool
	}{
		{"self_hosted", true},
		{"cloud", true},
		{"idc", true},
		{"edge_node", true},
		{"a", true},
		{"", false},
		{"1idc", false},
		{"IDC", false},
		{"idc-1", false},
		{"idc.1", false},
	}
	for _, tc := range cases {
		err := validateServerGroupCategory(tc.in)
		if tc.wantOK && err != nil {
			t.Fatalf("%q: unexpected err %v", tc.in, err)
		}
		if !tc.wantOK && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
	}
}
