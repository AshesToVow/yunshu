package config

import (
	"fmt"
	"strings"

	"yunshu/internal/pkg/crypto"
)

// 仓库内示例/占位凭据黑名单：出现在生产配置中即视为未完成密钥交付。
// 这些值曾经（或仍然）出现在 configs/config.yaml 与文档中，属公开已知值。
var wellKnownSecrets = map[string]struct{}{
	"change-me-32bytes-base64": {},
	"change-me":                {},
	"changeme":                 {},
	"YunshuJWTSecretKey2026MustBeAtLeast32BytesLong!!": {},
	"yunshu-secret": {},
	"secret":        {},
	"123456":        {},
}

// IsWellKnownSecret 判断给定凭据是否为仓库内公开占位值。
func IsWellKnownSecret(s string) bool {
	_, ok := wellKnownSecrets[strings.TrimSpace(s)]
	return ok
}

// IsProd 生产环境判定，统一收口，避免各处字面量比较写法不一致。
func (c *Config) IsProd() bool {
	if c == nil {
		return false
	}
	env := strings.ToLower(strings.TrimSpace(c.App.Env))
	return env == "prod" || env == "production"
}

// Validate 启动期安全闸门。
//
// 设计取舍：
//   - 空值在任何环境都直接失败——没有密钥就无法解密既有凭据，继续启动只会把
//     故障推迟到第一次业务调用，且报错点远离根因。
//   - 弱密钥（长度不合规、走 SHA-256 派生）与公开占位值在 prod 直接失败，
//     在非 prod 仅返回告警，保证本地开发与 CI 不被阻断。
//
// 返回的 warnings 由调用方写日志；error 非 nil 时调用方必须终止启动。
func (c *Config) Validate() (warnings []string, err error) {
	if c == nil {
		return nil, fmt.Errorf("config is nil")
	}
	prod := c.IsProd()

	// ---- security.encryption_key ----
	rawKey := strings.TrimSpace(c.Security.EncryptionKey)
	if rawKey == "" {
		return warnings, fmt.Errorf("security.encryption_key is required but not configured; " +
			"generate one via: openssl rand -base64 32 (or set ENCRYPTION_KEY)")
	}
	info, kerr := crypto.InspectKeyString(rawKey)
	if kerr != nil {
		return warnings, fmt.Errorf("security.encryption_key invalid: %w", kerr)
	}
	if IsWellKnownSecret(rawKey) {
		msg := "security.encryption_key is a well-known placeholder shipped in this repository; " +
			"anyone can decrypt cloud AK/SK, SSH and kubeconfig ciphertext. " +
			"generate one via: openssl rand -base64 32"
		if prod {
			return warnings, fmt.Errorf("%s", msg)
		}
		warnings = append(warnings, msg)
	}
	if !info.Strong() {
		// 派生路径能跑通，但密钥熵等于原始字符串熵；且一旦后续换成规范 32 字节密钥，
		// 历史密文将永久无法解密——必须在启动期就让运维看见，而非静默降级。
		msg := "security.encryption_key is not a 32-byte key (base64 or raw); " +
			"falling back to SHA-256 derivation. entropy equals the raw string, and switching to a " +
			"proper key later will make existing ciphertext undecryptable. " +
			"generate one via: openssl rand -base64 32"
		if prod {
			return warnings, fmt.Errorf("%s", msg)
		}
		warnings = append(warnings, msg)
	}

	// ---- auth.jwt_secret ----
	jwtSecret := strings.TrimSpace(c.Auth.JWTSecret)
	if jwtSecret == "" {
		return warnings, fmt.Errorf("auth.jwt_secret is required but not configured (or set JWT_SECRET)")
	}
	if IsWellKnownSecret(jwtSecret) {
		msg := "auth.jwt_secret is the repository default; tokens can be forged by anyone with source access"
		if prod {
			return warnings, fmt.Errorf("%s", msg)
		}
		warnings = append(warnings, msg)
	}
	if len(jwtSecret) < 32 {
		msg := fmt.Sprintf("auth.jwt_secret is %d bytes; HMAC-SHA256 signing keys should be >= 32 bytes", len(jwtSecret))
		if prod {
			return warnings, fmt.Errorf("%s", msg)
		}
		warnings = append(warnings, msg)
	}

	return warnings, nil
}

// AutoMigrateEnabled 决定进程启动时是否执行 GORM AutoMigrate。
//
// 语义：显式配置优先；未配置时按环境推断——非生产默认开启（本地/CI 便利），
// 生产默认关闭（表结构变更必须走独立的 `migrate` 命令，避免滚动发布期间
// 多副本并发改表、以及无法回滚的隐式 DDL）。
func (c *Config) AutoMigrateEnabled() bool {
	if c == nil {
		return false
	}
	if c.Database.AutoMigrate != nil {
		return *c.Database.AutoMigrate
	}
	if c.MySQL.AutoMigrate != nil {
		return *c.MySQL.AutoMigrate
	}
	return !c.IsProd()
}
