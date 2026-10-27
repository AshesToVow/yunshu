package system

import "testing"

func TestIsSingletonDictType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		dictType string
		want     bool
	}{
		{name: "cicd singleton", dictType: "cicd_harbor_url", want: true},
		{name: "trim and lower", dictType: "  CICD_HARBOR_TLS_VERIFY ", want: true},
		{name: "enum type not singleton", dictType: "cicd_pipeline_type", want: false},
		{name: "unknown type", dictType: "custom_dict", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isSingletonDictType(tc.dictType); got != tc.want {
				t.Fatalf("isSingletonDictType(%q) = %v, want %v", tc.dictType, got, tc.want)
			}
		})
	}
}

func TestIsBuiltinDictEntry(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		dictType string
		label    string
		want     bool
	}{
		{name: "singleton type by type only", dictType: "mail_host", label: "任意标签", want: true},
		{name: "seeded enum label", dictType: "cicd_pipeline_type", label: "前端服务", want: true},
		{name: "enum custom label", dictType: "cicd_pipeline_type", label: "自定义类型", want: false},
		{name: "unknown type", dictType: "custom_dict", label: "自定义类型", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isBuiltinDictEntry(tc.dictType, tc.label); got != tc.want {
				t.Fatalf("isBuiltinDictEntry(%q,%q) = %v, want %v", tc.dictType, tc.label, got, tc.want)
			}
		})
	}
}

func TestDictTypeLabelKeyCanonicalizesType(t *testing.T) {
	t.Parallel()
	a := dictTypeLabelKey("CICD_Pipeline_Type ", " 前端服务 ")
	b := dictTypeLabelKey("cicd_pipeline_type", "前端服务")
	if a != b {
		t.Fatalf("keys differ: %q vs %q", a, b)
	}
}
