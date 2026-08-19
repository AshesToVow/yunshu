package eventforward

import (
	"testing"

	"yunshu/internal/model"
)

func TestRuleFilter_Match(t *testing.T) {
	t.Parallel()
	ev := &model.K8sForwardedEvent{
		Namespace: "yunshu-logging",
		Name:      "yunshu-loggie-p1-dd9g2",
		Reason:    "Failed",
		Message:   `Error: ErrImagePull`,
	}
	cases := []struct {
		name string
		rule model.K8sEventForwardRule
		want bool
	}{
		{name: "empty_matches_all", rule: model.K8sEventForwardRule{}, want: true},
		{
			name: "namespace_exact",
			rule: model.K8sEventForwardRule{RuleNamespaces: `["yunshu-logging"]`},
			want: true,
		},
		{
			name: "namespace_miss",
			rule: model.K8sEventForwardRule{RuleNamespaces: `["springboot"]`},
			want: false,
		},
		{
			name: "name_substring",
			rule: model.K8sEventForwardRule{RuleNames: `["yunshu-loggie"]`},
			want: true,
		},
		{
			name: "reason_or_message",
			rule: model.K8sEventForwardRule{RuleReasons: `["ErrImagePull"]`},
			want: true,
		},
		{
			name: "reverse_excludes_match",
			rule: model.K8sEventForwardRule{RuleNamespaces: `["yunshu-logging"]`, RuleReverse: true},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ParseRuleFilter(tc.rule).Match(ev); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
