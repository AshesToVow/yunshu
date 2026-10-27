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

func TestParseRuleFilterTrimsAndDropsEmpty(t *testing.T) {
	rule := model.K8sEventForwardRule{
		RuleNamespaces: `[" kube-system ","", "default"]`,
		RuleNames:      `[" nginx ", ""]`,
		RuleReasons:    `["BackOff", " "]`,
	}
	f := ParseRuleFilter(rule)
	if len(f.Namespaces) != 2 || f.Namespaces[0] != "kube-system" || f.Namespaces[1] != "default" {
		t.Fatalf("unexpected namespaces: %#v", f.Namespaces)
	}
	if len(f.Names) != 1 || f.Names[0] != "nginx" {
		t.Fatalf("unexpected names: %#v", f.Names)
	}
	if len(f.Reasons) != 1 || f.Reasons[0] != "BackOff" {
		t.Fatalf("unexpected reasons: %#v", f.Reasons)
	}
}

func TestRuleFilterMatchAfterNormalize(t *testing.T) {
	f := ParseRuleFilter(model.K8sEventForwardRule{
		RuleNamespaces: `["default", ""]`,
		RuleReasons:    `["BackOff"]`,
	})
	ev := &model.K8sForwardedEvent{Namespace: "default", Name: "nginx-0", Reason: "BackOff", Message: "restarting failed"}
	if !f.Match(ev) {
		t.Fatal("expected matched event")
	}
	if f.Match(&model.K8sForwardedEvent{Namespace: "kube-system", Reason: "BackOff"}) {
		t.Fatal("expected namespace mismatch")
	}
}
