package constants

import "strings"

// HasPermissionResourceWildcard 权限 resource 含 Casbin keyMatch2 通配（* / /*）时禁止入库。
func HasPermissionResourceWildcard(resource string) bool {
	r := strings.TrimSpace(resource)
	if r == "" {
		return false
	}
	return strings.Contains(r, "*")
}
