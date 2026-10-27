package eventforward

import (
	"encoding/json"
	"strings"

	"yunshu/internal/model"
)

type RuleFilter struct {
	Namespaces []string
	Names      []string
	Reasons    []string
	Reverse    bool
}

func ParseRuleFilter(rule model.K8sEventForwardRule) RuleFilter {
	var rf RuleFilter
	_ = json.Unmarshal([]byte(rule.RuleNamespaces), &rf.Namespaces)
	_ = json.Unmarshal([]byte(rule.RuleNames), &rf.Names)
	_ = json.Unmarshal([]byte(rule.RuleReasons), &rf.Reasons)
	// 前端多行输入常见首尾空白与空项；空项会让「正选」永远不命中。
	rf.Namespaces = normalizeFilterValues(rf.Namespaces)
	rf.Names = normalizeFilterValues(rf.Names)
	rf.Reasons = normalizeFilterValues(rf.Reasons)
	rf.Reverse = rule.RuleReverse
	return rf
}

func normalizeFilterValues(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func ParseClusterIDSet(clusterIDs string) map[string]struct{} {
	out := make(map[string]struct{})
	for p := range strings.SplitSeq(clusterIDs, ",") {
		c := strings.TrimSpace(p)
		if c != "" {
			out[c] = struct{}{}
		}
	}
	return out
}

func (f RuleFilter) Match(ev *model.K8sForwardedEvent) bool {
	matched := f.matchPositive(ev)
	if f.Reverse {
		return !matched
	}
	return matched
}

func (f RuleFilter) matchPositive(ev *model.K8sForwardedEvent) bool {
	if len(f.Namespaces) > 0 {
		nsEvent := strings.TrimSpace(ev.Namespace)
		ok := false
		for _, ns := range f.Namespaces {
			if ns == nsEvent {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Names) > 0 {
		ok := false
		for _, n := range f.Names {
			if n != "" && strings.Contains(ev.Name, n) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Reasons) > 0 {
		ok := false
		for _, r := range f.Reasons {
			if r == "" {
				continue
			}
			if strings.Contains(ev.Reason, r) || strings.Contains(ev.Message, r) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}
