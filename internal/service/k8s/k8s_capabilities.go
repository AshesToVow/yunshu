package k8s

import (
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/k8scaps"
)

// 能力码别名，便于 service 层调用。
const (
	CapRead         = k8scaps.Read
	CapExec         = k8scaps.Exec
	CapRestart      = k8scaps.Restart
	CapScale        = k8scaps.Scale
	CapApply        = k8scaps.Apply
	CapDelete       = k8scaps.Delete
	CapSecretReveal = k8scaps.SecretReveal
	CapDestructive  = k8scaps.Destructive
)

// K8sCapabilityMeta 能力目录项（前端勾选）。
type K8sCapabilityMeta = k8scaps.Meta

// CapabilityCatalog 固定能力包列表。
func CapabilityCatalog() []K8sCapabilityMeta { return k8scaps.Catalog() }

// RequiredK8sCapability 路由所需能力（单值）。
func RequiredK8sCapability(perms []model.Permission, routePath, httpMethod, actionCode string) string {
	path := strings.TrimSpace(routePath)
	method := strings.ToUpper(strings.TrimSpace(httpMethod))
	code := strings.TrimSpace(actionCode)
	pathLower := strings.ToLower(path)
	codeLower := strings.ToLower(code)

	if method == "GET" && strings.HasSuffix(path, "/secrets/reveal") {
		return CapSecretReveal
	}
	if IsK8sNginxRestartRoute(routePath, httpMethod) {
		return CapRestart
	}
	if strings.Contains(pathLower, "exec") || strings.Contains(codeLower, "exec") ||
		strings.HasSuffix(pathLower, "/pods/debug") {
		return CapExec
	}
	if strings.Contains(pathLower, "/pods/file") &&
		(strings.Contains(pathLower, "upload") || strings.Contains(pathLower, "delete")) {
		return CapExec
	}
	if strings.Contains(pathLower, "/nodes/drain") ||
		strings.Contains(pathLower, "/rbac/apply") ||
		(strings.Contains(pathLower, "/helm/") &&
			(strings.HasSuffix(pathLower, "/uninstall") ||
				strings.HasSuffix(pathLower, "/install") ||
				strings.HasSuffix(pathLower, "/upgrade"))) {
		return CapDestructive
	}
	if strings.Contains(pathLower, "scale") || strings.Contains(codeLower, "scale") {
		return CapScale
	}
	if strings.Contains(pathLower, "restart") || strings.Contains(codeLower, "restart") ||
		strings.Contains(pathLower, "rollout") {
		return CapRestart
	}
	if method == "DELETE" {
		return CapDelete
	}
	if method == "POST" || method == "PUT" || method == "PATCH" {
		if strings.Contains(pathLower, "apply") || strings.Contains(pathLower, "create") {
			return CapApply
		}
		return CapApply
	}

	rank := RequiredK8sAccessRank(perms, routePath, httpMethod, actionCode)
	switch rank {
	case K8sAccessRankReadonly:
		return CapRead
	case K8sAccessRankReadonlyExec:
		return CapExec
	default:
		return CapDestructive
	}
}
