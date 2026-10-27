package k8s

import (
	"sort"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8scaps"
	"yunshu/internal/pkg/k8srole"
)

func grantRow(id uint, preset, namespaces, blacklist, caps string) model.K8sClusterAccessGrant {
	return model.K8sClusterAccessGrant{
		ID:                  id,
		PrincipalKind:       model.K8sPrincipalRole,
		PrincipalRef:        "viewer",
		ClusterID:           0,
		Preset:              preset,
		Namespaces:          namespaces,
		BlacklistNamespaces: blacklist,
		Capabilities:        caps,
	}
}

func TestPlanK8sGrantMigrationMergesSameTargetRole(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(169, "readonly", "", "", ""),
		grantRow(170, k8srole.ClusterReadonly, "ns-a", "denied", k8scaps.Marshal([]string{k8scaps.Read})),
	}
	plan := planK8sGrantMigration(grants, nil, nil)

	if len(plan.Survivors) != 1 {
		t.Fatalf("同主体同目标角色应合并为 1 行，实际 %d", len(plan.Survivors))
	}
	got := plan.Survivors[0]
	if got.ID != 169 {
		t.Fatalf("应保留 id 最小的行，实际 %d", got.ID)
	}
	if got.Preset != k8srole.ClusterReadonly {
		t.Fatalf("归一后应为 cluster_readonly，实际 %q", got.Preset)
	}
	if got.Namespaces != "" {
		t.Fatalf("任一行为空即不限制，实际 %q", got.Namespaces)
	}
	if got.BlacklistNamespaces != "denied" {
		t.Fatalf("黑名单应取并集，实际 %q", got.BlacklistNamespaces)
	}
	if len(plan.DeleteIDs) != 1 || plan.DeleteIDs[0] != 170 {
		t.Fatalf("应删除重复行 170，实际 %v", plan.DeleteIDs)
	}
	if len(plan.ExecRows) != 0 {
		t.Fatalf("只读角色不应补 Exec 行，实际 %v", plan.ExecRows)
	}
}

func TestPlanK8sGrantMigrationKeepsDistinctRoles(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(1, k8srole.ClusterReadonly, "ns-a", "", ""),
		grantRow(2, k8srole.ClusterPodExec, "ns-b", "", ""),
	}
	plan := planK8sGrantMigration(grants, nil, nil)

	if len(plan.Survivors) != 2 {
		t.Fatalf("不同角色不应合并，实际 %d 行", len(plan.Survivors))
	}
	if len(plan.DeleteIDs) != 0 || len(plan.ExecRows) != 0 {
		t.Fatalf("不应删除或补行：delete=%v exec=%v", plan.DeleteIDs, plan.ExecRows)
	}
}

func TestPlanK8sGrantMigrationExpandsReadonlyExec(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(1, k8scaps.PresetReadonlyExec, "ns-a", "", k8scaps.Marshal([]string{k8scaps.Read, k8scaps.Exec})),
	}
	plan := planK8sGrantMigration(grants, nil, nil)

	if len(plan.Survivors) != 1 || plan.Survivors[0].Preset != k8srole.ClusterReadonly {
		t.Fatalf("只读+Exec 主行应落 cluster_readonly，实际 %+v", plan.Survivors)
	}
	if len(plan.ExecRows) != 1 {
		t.Fatalf("应补 1 行 Exec，实际 %d", len(plan.ExecRows))
	}
	exec := plan.ExecRows[0]
	if exec.Preset != k8srole.ClusterPodExec || exec.Namespaces != "ns-a" {
		t.Fatalf("Exec 行应继承主行命名空间，实际 %+v", exec)
	}
}

func TestPlanK8sGrantMigrationMergesLegacyAdminAndCustom(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(1, "admin", "", "", ""),
		grantRow(2, k8scaps.PresetCustom, "", "", k8scaps.Marshal([]string{k8scaps.Read, k8scaps.Restart})),
	}
	plan := planK8sGrantMigration(grants, nil, nil)

	if len(plan.Survivors) != 1 || plan.Survivors[0].Preset != k8srole.ClusterAdmin {
		t.Fatalf("admin 与自定义高危能力应合并为 cluster_admin，实际 %+v", plan.Survivors)
	}
	if len(plan.DeleteIDs) != 1 {
		t.Fatalf("应删除重复行，实际 %v", plan.DeleteIDs)
	}
	if plan.Survivors[0].Capabilities != k8scaps.Marshal(k8srole.CapsForRole(k8srole.ClusterAdmin)) {
		t.Fatalf("能力包应按 admin 重写，实际 %q", plan.Survivors[0].Capabilities)
	}
}

func TestPlanK8sGrantMigrationFillsNamespacesFromRules(t *testing.T) {
	k := nsKey{kind: model.K8sPrincipalRole, ref: "viewer", cluster: 0}
	allowBy := map[nsKey][]string{k: {"ns-b", "ns-a", "ns-a"}}
	denyBy := map[nsKey][]string{k: {"kube-system"}}
	grants := []model.K8sClusterAccessGrant{grantRow(1, k8srole.ClusterReadonly, "", "", "")}

	plan := planK8sGrantMigration(grants, allowBy, denyBy)
	got := plan.Survivors[0]
	if got.Namespaces != "ns-b,ns-a" {
		t.Fatalf("白名单应从 allow 规则回填并去重保序，实际 %q", got.Namespaces)
	}
	if got.BlacklistNamespaces != "kube-system" {
		t.Fatalf("黑名单应从 deny 规则回填，实际 %q", got.BlacklistNamespaces)
	}
}

func TestPlanK8sGrantMigrationIdempotent(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(1, k8srole.ClusterReadonly, "ns-a", "", ""),
		grantRow(2, k8srole.ClusterPodExec, "ns-a", "", ""),
	}
	first := planK8sGrantMigration(grants, nil, nil)
	if len(first.DeleteIDs) != 0 || len(first.ExecRows) != 0 {
		t.Fatalf("已归一的数据不应产生删除/补行：%v %v", first.DeleteIDs, first.ExecRows)
	}

	// 二次执行应得到完全一致的落库内容（迁移可重复运行）。
	normalized := append([]model.K8sClusterAccessGrant(nil), first.Survivors...)
	second := planK8sGrantMigration(normalized, nil, nil)
	for i := range first.Survivors {
		if first.Survivors[i].Preset != second.Survivors[i].Preset ||
			first.Survivors[i].Namespaces != second.Survivors[i].Namespaces ||
			first.Survivors[i].Capabilities != second.Survivors[i].Capabilities {
			t.Fatalf("二次迁移不应改变数据：%+v vs %+v", first.Survivors[i], second.Survivors[i])
		}
	}
}

func TestPlanK8sGrantMigrationUnionsNamespaces(t *testing.T) {
	grants := []model.K8sClusterAccessGrant{
		grantRow(1, k8srole.ClusterReadonly, "ns-a", "", ""),
		grantRow(2, "readonly", "ns-b,ns-a", "", ""),
		grantRow(3, k8srole.ClusterReadonly, "ns-c", "", ""),
	}
	plan := planK8sGrantMigration(grants, nil, nil)
	if len(plan.Survivors) != 1 {
		t.Fatalf("三行应合并为 1 行，实际 %d", len(plan.Survivors))
	}
	parts := k8srole.SplitNamespaces(plan.Survivors[0].Namespaces)
	sort.Strings(parts)
	if len(parts) != 3 || parts[0] != "ns-a" || parts[1] != "ns-b" || parts[2] != "ns-c" {
		t.Fatalf("白名单应取并集，实际 %q", plan.Survivors[0].Namespaces)
	}
}
