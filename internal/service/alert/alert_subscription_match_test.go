package alert

import (
	"testing"
)

func TestMatchNodeContinueWithDisabledChildrenDoesNotUseParentRGs(t *testing.T) {
	svc := &AlertSubscriptionService{}
	parent := &CachedSubscriptionNode{
		ID:               1,
		Name:             "root",
		Enabled:          true,
		Continue:         true,
		HasChildrenInDB:  true,
		ReceiverGroupIDs: []uint{99}, // 若误用父节点接收组会走到企微/邮箱
		MatchSeverity:    "",
		MatchLabels:      map[string]string{},
		Children:         nil, // 子节点均停用后缓存中为空
	}
	_, ok := svc.matchNodeRecursiveDetailed(parent, map[string]string{"severity": "warning"}, "warning", "firing")
	if ok {
		t.Fatalf("continue 父节点在子节点均未命中/均停用时不应再使用自身接收组")
	}
}

func TestMatchNodeContinueLeafWithOwnRGsStillWorks(t *testing.T) {
	svc := &AlertSubscriptionService{}
	leaf := &CachedSubscriptionNode{
		ID:               4,
		Name:             "warn-wecom",
		Enabled:          true,
		Continue:         true, // 现网部分叶子也开了 continue
		HasChildrenInDB:  false,
		ReceiverGroupIDs: []uint{7},
		MatchSeverity:    "warning",
		MatchLabels:      map[string]string{"severity": "warning"},
	}
	res, ok := svc.matchNodeRecursiveDetailed(leaf, map[string]string{"severity": "warning"}, "warning", "firing")
	if !ok {
		t.Fatalf("无子节点的 continue 叶子应仍可使用自身接收组")
	}
	if len(res.ReceiverGroupIDs) != 1 || res.ReceiverGroupIDs[0] != 7 {
		t.Fatalf("unexpected receiver groups: %#v", res.ReceiverGroupIDs)
	}
}

func TestMatchNodeDisabledNeverMatches(t *testing.T) {
	svc := &AlertSubscriptionService{}
	node := &CachedSubscriptionNode{
		ID:               4,
		Name:             "warn-wecom",
		Enabled:          false,
		Continue:         false,
		ReceiverGroupIDs: []uint{7},
		MatchSeverity:    "warning",
	}
	_, ok := svc.matchNodeRecursiveDetailed(node, map[string]string{"severity": "warning"}, "warning", "firing")
	if ok {
		t.Fatalf("停用节点不得命中")
	}
}
