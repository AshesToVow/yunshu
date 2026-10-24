package k8srole

import (
	"fmt"
	"strings"

	"yunshu/internal/model"
)

// ActionFromHTTP 将控制台 HTTP 映射为 k8m CheckPermissionLogic 的 action。
// 写操作（含滚动重启 / 扩缩容 / Apply）一律视为 patch/update/create/delete，需 cluster_admin。
func ActionFromHTTP(method, routePath string) string {
	p := strings.ToLower(strings.TrimSpace(routePath))
	m := strings.ToUpper(strings.TrimSpace(method))
	if strings.Contains(p, "exec") || strings.HasSuffix(p, "/pods/debug") ||
		strings.Contains(p, "/pods/file/upload") || strings.Contains(p, "/pods/file/delete") {
		return "exec"
	}
	switch m {
	case "DELETE":
		return "delete"
	case "POST", "PUT", "PATCH":
		return "patch"
	default:
		return "get"
	}
}

func filterByRole(grants []model.K8sClusterAccessGrant, role string) []model.K8sClusterAccessGrant {
	want := Normalize(role)
	var out []model.K8sClusterAccessGrant
	for _, g := range grants {
		n := Normalize(g.Preset)
		if n == want {
			out = append(out, g)
			continue
		}
		if want == ClusterReadonly && n == k8scapsReadonlyExecAlias() {
			out = append(out, g)
		}
		if want == ClusterPodExec && n == k8scapsReadonlyExecAlias() {
			out = append(out, g)
		}
		if want == ClusterAdmin && n == k8scapsAdminAlias() {
			out = append(out, g)
		}
	}
	return out
}

func k8scapsReadonlyExecAlias() string { return "readonly_exec" }
func k8scapsAdminAlias() string        { return "admin" }

func anyNSInCSV(nsList []string, csv string) bool {
	set := map[string]struct{}{}
	for _, n := range SplitNamespaces(csv) {
		set[n] = struct{}{}
	}
	for _, n := range nsList {
		if _, ok := set[strings.TrimSpace(n)]; ok {
			return true
		}
	}
	return false
}

func allNSInCSVOrUnrestricted(nsList []string, csv string) bool {
	if strings.TrimSpace(csv) == "" {
		return true
	}
	set := map[string]struct{}{}
	for _, n := range SplitNamespaces(csv) {
		set[n] = struct{}{}
	}
	for _, n := range nsList {
		s := strings.TrimSpace(n)
		if s == "" {
			continue
		}
		if _, ok := set[s]; !ok {
			return false
		}
	}
	return true
}

func nsDenied(grants []model.K8sClusterAccessGrant, nsList []string) bool {
	for _, g := range grants {
		if strings.TrimSpace(g.BlacklistNamespaces) != "" && anyNSInCSV(nsList, g.BlacklistNamespaces) {
			return true
		}
	}
	return false
}

func nsAllowedByWhitelist(grants []model.K8sClusterAccessGrant, nsList []string) bool {
	for _, g := range grants {
		if allNSInCSVOrUnrestricted(nsList, g.Namespaces) {
			return true
		}
	}
	return false
}

func checkNS(grants []model.K8sClusterAccessGrant, nsList []string, username, cluster, verb string) error {
	if len(nsList) == 0 {
		return nil
	}
	if nsDenied(grants, nsList) {
		return fmt.Errorf("用户[%s]没有集群[%s] [%s] %s权限-进入命名空间黑名单", username, cluster, strings.Join(nsList, ","), verb)
	}
	if !nsAllowedByWhitelist(grants, nsList) {
		return fmt.Errorf("用户[%s]没有集群[%s] [%s] %s权限-不在命名空间白名单", username, cluster, strings.Join(nsList, ","), verb)
	}
	return nil
}

// Check 对齐 k8m pkg/comm.CheckPermissionLogic（不含平台管理员旁路；由调用方处理）。
// grants 须已按主体 + 集群（含 cluster_id=0）筛过。name 仅用于日志，不参与判定。
func Check(grants []model.K8sClusterAccessGrant, cluster, username, action string, nsList []string) error {
	if len(grants) == 0 {
		return fmt.Errorf("用户[%s]没有集群[%s]访问权限", username, cluster)
	}
	switch action {
	case "exec":
		admins := filterByRole(grants, ClusterAdmin)
		if len(admins) == 0 {
			ro := filterByRole(grants, ClusterReadonly)
			if len(ro) == 0 {
				return fmt.Errorf("用户[%s]没有集群[%s] 只读权限", username, cluster)
			}
			execs := filterByRole(grants, ClusterPodExec)
			if len(execs) == 0 {
				return fmt.Errorf("用户[%s]没有集群[%s] Exec权限", username, cluster)
			}
			return checkNS(execs, nsList, username, cluster, "Exec")
		}
		return checkNS(admins, nsList, username, cluster, "Exec")
	case "delete", "update", "patch", "create":
		admins := filterByRole(grants, ClusterAdmin)
		if len(admins) == 0 {
			return fmt.Errorf("用户[%s]没有集群[%s] 操作权限", username, cluster)
		}
		return checkNS(admins, nsList, username, cluster, "操作")
	default:
		readers := append(filterByRole(grants, ClusterReadonly), filterByRole(grants, ClusterAdmin)...)
		if len(readers) == 0 {
			return fmt.Errorf("用户[%s]没有集群[%s] 读取/管理员 权限", username, cluster)
		}
		return checkNS(readers, nsList, username, cluster, "读取")
	}
}

// FilterNamespaces 列表可见 NS：对每个名字做 list/get 校验。
func FilterNamespaces(grants []model.K8sClusterAccessGrant, cluster, username string, names []string) []string {
	if len(names) == 0 {
		return names
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		ns := strings.TrimSpace(n)
		if ns == "" {
			continue
		}
		if err := Check(grants, cluster, username, "get", []string{ns}); err == nil {
			out = append(out, ns)
		}
	}
	return out
}

// NamespaceAllowed 单 NS 读权限。
func NamespaceAllowed(grants []model.K8sClusterAccessGrant, cluster, username, namespace string) bool {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return true
	}
	return Check(grants, cluster, username, "get", []string{ns}) == nil
}
