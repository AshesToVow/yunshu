package dictmask

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ValueHash 返回字典值（去首尾空白后）的 SHA-256 十六进制摘要，用于 (dict_type, value) 唯一约束。
// 字典值可能是 MEDIUMTEXT 级 kubeconfig，无法直接建唯一索引，因此索引摘要。
func ValueHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

// SensitiveDictType 判断字典类型是否应按「敏感凭据」脱敏（options 不落明文）。
func SensitiveDictType(dictType string) bool {
	t := strings.ToLower(strings.TrimSpace(dictType))
	if t == "" {
		return false
	}
	// 显名单类
	switch t {
	case "alert_webhook_token", "alert_enrich_prometheus_token",
		"minio_secret_key", "mail_password", "cicd_jenkins_api_token",
		"auth_jwt_secret",
		"k8s_kubeconfig_template", "k8s_direct_config",
		"ai_openai_api_key", "ai_deepseek_api_key", "ai_anthropic_api_key":
		return true
	}
	suffixes := []string{
		"_password", "_private_key", "_sk", "_secret", "_token",
		"_corp_secret", "_app_secret", "_api_key", "_access_key",
	}
	for _, s := range suffixes {
		if strings.HasSuffix(t, s) {
			return true
		}
	}
	// Access Key 仍属高敏：options 仅脱敏预览
	if strings.HasSuffix(t, "_ak") {
		return true
	}
	return false
}

// Preview 返回用于下拉展示的脱敏文案（非可逆、不可用于业务回填）。
func Preview(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "（空）"
	}
	runes := []rune(s)
	n := len(runes)
	if n <= 8 {
		return "****"
	}
	head := string(runes[:4])
	tail := string(runes[n-4:])
	if strings.Contains(s, "BEGIN") && strings.Contains(s, "KEY") {
		return head + " …（" + strconv.Itoa(utf8.RuneCountInString(s)) + " 字符）… " + tail
	}
	return head + " … " + tail
}
