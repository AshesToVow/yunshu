package k8s

import (
	"crypto/cipher"
	"strings"

	cryptox "yunshu/internal/pkg/crypto"
)

// sealClusterSecret 将明文凭证加密为 base64(AES-GCM)；无密钥时原样返回（开发环境兜底）。
func sealClusterSecret(aead cipher.AEAD, plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if aead == nil {
		return plain, nil
	}
	return cryptox.EncryptString(aead, plain)
}

// openClusterSecret 解密凭证；兼容历史明文（解密失败且内容像 kubeconfig/JSON 时原样返回）。
func openClusterSecret(aead cipher.AEAD, stored string) (string, error) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return "", nil
	}
	if aead == nil {
		return stored, nil
	}
	pt, err := cryptox.DecryptString(aead, stored)
	if err == nil {
		return pt, nil
	}
	// 存量明文：以 apiVersion / { 等启发式识别，避免误把损坏密文当明文用。
	if looksLikePlainClusterSecret(stored) {
		return stored, nil
	}
	return "", err
}

func looksLikePlainClusterSecret(s string) bool {
	if strings.Contains(s, "apiVersion:") || strings.Contains(s, "kind: Config") {
		return true
	}
	if strings.HasPrefix(s, "{") && strings.Contains(s, "server") {
		return true
	}
	return false
}
