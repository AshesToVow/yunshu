package jenkins

import (
	"fmt"
	"net/http"
	"strings"
)

// HumanizeAPIError 将 Jenkins HTTP 错误转为可读说明（供 API 返回）。
func HumanizeAPIError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "http 401"), strings.Contains(msg, "unauthorized"):
		return "Jenkins 认证失败（HTTP 401），请检查数据字典 cicd_jenkins_username 与 cicd_jenkins_api_token 是否正确（API Token 在 Jenkins → 用户 → 设置 → API Token 生成）"
	case strings.Contains(msg, "http 403"), strings.Contains(msg, "forbidden"):
		return "Jenkins 权限不足（HTTP 403），请确认该用户具备创建/更新 Job 的权限"
	case strings.Contains(msg, "http 404"):
		return "Jenkins 资源不存在（HTTP 404），请检查 cicd_jenkins_base_url 与 Job 名称"
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"), strings.Contains(msg, "dial tcp"):
		return "无法连接 Jenkins，请检查 cicd_jenkins_base_url 网络是否可达"
	default:
		return strings.TrimSpace(err.Error())
	}
}

func httpStatusFromError(err error) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusBadRequest} {
		if strings.Contains(msg, fmt.Sprintf("HTTP %d", code)) {
			return code
		}
	}
	return 0
}

// IsUnauthorized 判断是否为 Jenkins 401 认证错误。
func IsUnauthorized(err error) bool {
	return httpStatusFromError(err) == http.StatusUnauthorized ||
		strings.Contains(strings.ToLower(err.Error()), "unauthorized")
}
