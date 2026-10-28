// Package totp 实现 RFC 6238 的 TOTP（RFC 4226 HOTP 的时间步变体）。
// 仅供登录二次验证使用：默认 6 位、30 秒步长、SHA-1、±1 步容差。
// 这里用 SHA-1 是 RFC 6238 的默认算法，主流验证器 App 均按此生成，换算法会导致兼容性问题。
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	secretBytes = 20
	// Digits 验证码位数。
	Digits = 6
	// Period 时间步长。
	Period = 30 * time.Second
	// skew 允许前后各偏移几个时间步，抵消客户端与服务器时钟误差。
	skew = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret 生成 160bit 随机密钥（base32 无填充）。
func GenerateSecret() (string, error) {
	buf := make([]byte, secretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

// KeyURI 生成 otpauth:// 链接，供验证器 App 扫码或手动录入。
func KeyURI(issuer, account, secret string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "Yunshu"
	}
	account = strings.TrimSpace(account)
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", normalizeSecret(secret))
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprintf("%d", Digits))
	q.Set("period", fmt.Sprintf("%d", int(Period.Seconds())))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// Validate 校验 code 是否匹配密钥在 at 时间点（含 ±1 步容差）的 TOTP。
func Validate(secret, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != Digits || normalizeSecret(secret) == "" {
		return false
	}
	counter := at.Unix() / int64(Period.Seconds())
	for offset := int64(-skew); offset <= skew; offset++ {
		c := counter + offset
		if c < 0 {
			continue
		}
		expect, err := hotp(secret, uint64(c), Digits)
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(expect), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

// Code 返回 secret 在 at 时刻的 TOTP，主要供测试与调试使用。
func Code(secret string, at time.Time) (string, error) {
	return hotp(secret, uint64(at.Unix()/int64(Period.Seconds())), Digits)
}

func hotp(secret string, counter uint64, digits int) (string, error) {
	key, err := b32.DecodeString(normalizeSecret(secret))
	if err != nil {
		return "", fmt.Errorf("totp: invalid secret: %w", err)
	}
	if len(key) == 0 {
		return "", fmt.Errorf("totp: empty secret")
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	mod := uint32(1)
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, value%mod), nil
}

func normalizeSecret(secret string) string {
	s := strings.ToUpper(strings.TrimSpace(secret))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.TrimRight(s, "=")
}
