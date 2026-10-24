package k8srole

import (
	"testing"

	"yunshu/internal/model"
)

func TestCheck_AdminRestartAndReadonlyDenied(t *testing.T) {
	t.Parallel()
	admin := []model.K8sClusterAccessGrant{{
		Preset: ClusterAdmin, ClusterID: 1, Namespaces: "app-dev",
	}}
	if err := Check(admin, "1", "dev", "patch", []string{"app-dev"}); err != nil {
		t.Fatalf("admin restart: %v", err)
	}
	ro := []model.K8sClusterAccessGrant{{
		Preset: ClusterReadonly, ClusterID: 1, Namespaces: "app-dev",
	}}
	if err := Check(ro, "1", "dev", "patch", []string{"app-dev"}); err == nil {
		t.Fatal("readonly must not patch/restart")
	}
	if err := Check(ro, "1", "dev", "get", []string{"app-dev"}); err != nil {
		t.Fatalf("readonly get: %v", err)
	}
}

func TestCheck_ExecNeedsReadonlyPlusExec(t *testing.T) {
	t.Parallel()
	onlyExec := []model.K8sClusterAccessGrant{{Preset: ClusterPodExec, ClusterID: 1}}
	if err := Check(onlyExec, "1", "dev", "exec", nil); err == nil {
		t.Fatal("exec without readonly must fail")
	}
	both := []model.K8sClusterAccessGrant{
		{Preset: ClusterReadonly, ClusterID: 1, Namespaces: "app-dev"},
		{Preset: ClusterPodExec, ClusterID: 1, Namespaces: "app-dev"},
	}
	if err := Check(both, "1", "dev", "exec", []string{"app-dev"}); err != nil {
		t.Fatalf("readonly+exec: %v", err)
	}
	if err := Check(both, "1", "dev", "exec", []string{"other"}); err == nil {
		t.Fatal("exec outside whitelist")
	}
}

func TestCheck_BlacklistWins(t *testing.T) {
	t.Parallel()
	g := []model.K8sClusterAccessGrant{{
		Preset: ClusterAdmin, ClusterID: 1, Namespaces: "", BlacklistNamespaces: "kube-system",
	}}
	if err := Check(g, "1", "dev", "get", []string{"kube-system"}); err == nil {
		t.Fatal("blacklist")
	}
	if err := Check(g, "1", "dev", "get", []string{"default"}); err != nil {
		t.Fatalf("other ns: %v", err)
	}
}

func TestActionFromHTTP(t *testing.T) {
	t.Parallel()
	if ActionFromHTTP("POST", "/api/v1/deployments/restart") != "patch" {
		t.Fatal("restart is patch")
	}
	if ActionFromHTTP("POST", "/api/v1/pods/exec") != "exec" {
		t.Fatal("exec")
	}
	if ActionFromHTTP("GET", "/api/v1/pods") != "get" {
		t.Fatal("get")
	}
	if ActionFromHTTP("DELETE", "/api/v1/deployments") != "delete" {
		t.Fatal("delete")
	}
}

func TestExpandToStoredRoles(t *testing.T) {
	t.Parallel()
	got := ExpandToStoredRoles("readonly_exec", nil)
	if len(got) != 2 || got[0] != ClusterReadonly || got[1] != ClusterPodExec {
		t.Fatalf("got %v", got)
	}
	if ExpandToStoredRoles("admin", nil)[0] != ClusterAdmin {
		t.Fatal("admin")
	}
}
