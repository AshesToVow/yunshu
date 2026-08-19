package k8s

import "strings"

// IsK8sReadAPIPath 判断是否为控制台使用的「资源列表/详情」类只读 API。
// 用于 K8sScopeAuthorize 对 GET 强制集群档位；勿单独作为 Casbin 放行依据。
func IsK8sReadAPIPath(path string) bool {
	p := strings.TrimSpace(path)
	k8sPrefixes := []string{
		"/api/v1/k8s-policies/cluster-auth-matrix",
		"/api/v1/k8s-policies/user-cluster-auth",
		"/api/v1/clusters",
		"/api/v1/pods",
		"/api/v1/namespaces",
		"/api/v1/nodes",
		"/api/v1/deployments",
		"/api/v1/statefulsets",
		"/api/v1/daemonsets",
		"/api/v1/cronjobs",
		"/api/v1/jobs",
		"/api/v1/configmaps",
		"/api/v1/secrets",
		"/api/v1/k8s-services",
		"/api/v1/persistentvolumes",
		"/api/v1/persistentvolumeclaims",
		"/api/v1/storageclasses",
		"/api/v1/ingresses",
		"/api/v1/events",
		"/api/v1/crds",
		"/api/v1/crs",
		"/api/v1/rbac",
		"/api/v1/serviceaccounts",
		"/api/v1/horizontal-pod-autoscalers",
		"/api/v1/helm/releases",
		"/api/v1/helm/harbor",
		"/api/v1/network-policies",
		"/api/v1/k8s/search",
		"/api/v1/k8s/topology",
		"/api/v1/pods/diagnose",
		"/api/v1/ingresses/diagnose",
	}
	for _, prefix := range k8sPrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// IsK8sClusterGrantReadBypassPath 集群档位可否绕过 Casbin GET。
// 仅覆盖工作负载/命名空间等资源只读；敏感与运维面 API 必须显式 Casbin。
func IsK8sClusterGrantReadBypassPath(path string) bool {
	p := strings.TrimSpace(path)
	if p == "" || !IsK8sReadAPIPath(p) {
		return false
	}
	if strings.HasPrefix(p, "/api/v1/k8s-policies") {
		return false
	}
	if strings.HasPrefix(p, "/api/v1/secrets") {
		return false
	}
	if strings.HasPrefix(p, "/api/v1/helm/harbor") {
		return false
	}
	if strings.Contains(p, "/exec") || strings.Contains(p, "/file") {
		return false
	}
	return true
}
