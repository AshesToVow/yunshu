package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// KeyInfo 描述 encryption_key 的实际强度来源，供启动期校验判断是否允许放行。
type KeyInfo struct {
	// Base64 为 true 表示 key 本身是合法 base64 且解码后正好 32 字节（推荐形态）。
	Base64 bool
	// Raw 为 true 表示 key 本身就是 32 字节原文。
	Raw bool
	// Derived 为 true 表示 key 长度不合规，实际密钥由 SHA-256 派生而来。
	// 这是历史兼容路径：能跑通，但熵完全取决于原始字符串，弱口令即弱密钥。
	Derived bool
}

// Strong 表示密钥是显式的 32 字节材料，而非从任意字符串 SHA-256 派生。
func (k KeyInfo) Strong() bool { return k.Base64 || k.Raw }

// InspectKeyString 解析 encryption_key 的形态，不产生 cipher，仅用于启动期校验与告警。
func InspectKeyString(key string) (KeyInfo, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return KeyInfo{}, errors.New("encryption_key is required")
	}
	if kb, err := base64.StdEncoding.DecodeString(key); err == nil && len(kb) == 32 {
		return KeyInfo{Base64: true}, nil
	}
	if len([]byte(key)) == 32 {
		return KeyInfo{Raw: true}, nil
	}
	return KeyInfo{Derived: true}, nil
}

// NewAESGCMFromKeyString creates AES-GCM from a key string.
// Preferred input: base64(32 bytes) or raw 32 bytes.
// Backward-compatible fallback: derive a 32-byte key from any non-empty string via SHA-256.
// 注意：派生路径仅为兼容既有密文而保留，启动期应由 config.ValidateSecurity 拦截弱密钥。
func NewAESGCMFromKeyString(key string) (cipher.AEAD, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("encryption_key is required")
	}

	kb, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		// fallback: treat as raw bytes
		kb = []byte(key)
	}
	if len(kb) != 32 {
		sum := sha256.Sum256([]byte(key))
		kb = sum[:]
	}

	block, err := aes.NewCipher(kb)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func EncryptString(aead cipher.AEAD, plaintext string) (string, error) {
	if aead == nil {
		return "", errors.New("aead is nil")
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := aead.Seal(nil, nonce, []byte(plaintext), nil)
	out := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

func DecryptString(aead cipher.AEAD, ciphertextB64 string) (string, error) {
	if aead == nil {
		return "", errors.New("aead is nil")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ciphertextB64))
	if err != nil {
		return "", err
	}
	ns := aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
