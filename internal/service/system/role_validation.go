package system

import (
	"regexp"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

var roleCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// normalizeRoleCode 角色编码仅允许小写字母/数字/下划线/短横线，避免大小写与空白变体伪造内置角色。
func normalizeRoleCode(raw string) (string, error) {
	code := strings.TrimSpace(raw)
	if code == "" {
		return "", constants.ErrBadRequestWithMsg("角色编码不能为空")
	}
	if strings.EqualFold(code, auth.SuperAdminRoleCode) {
		return "", constants.ErrBadRequestWithMsg("super-admin 为内置保留角色编码，不允许使用")
	}
	if !roleCodePattern.MatchString(code) {
		return "", constants.ErrBadRequestWithMsg("角色编码仅支持小写字母、数字、下划线和短横线（最长 64 位），且必须以字母或数字开头")
	}
	return code, nil
}

func isBuiltinSuperAdminRole(code string) bool {
	return strings.TrimSpace(code) == auth.SuperAdminRoleCode
}
