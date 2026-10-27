package dictconfig

import (
	"context"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// AuthDictTypes 数据字典中覆盖 auth.* 的 dict_type。
type AuthDictTypes struct {
	JWTSecret                string
	AccessTokenTTLMinutes    string
	RefreshTokenTTLHours     string
	CookieSecure             string
	CookieDomain             string
	CSPEnabled               string
	EmailCodeTTLSeconds      string
	EmailCodeCooldownSeconds string
	LoginMaxFailAttempts     string
	LoginLockSeconds         string
}

func DefaultAuthDictTypes() AuthDictTypes {
	return AuthDictTypes{
		JWTSecret:                "auth_jwt_secret",
		AccessTokenTTLMinutes:    "auth_access_token_ttl_minutes",
		RefreshTokenTTLHours:     "auth_refresh_token_ttl_hours",
		CookieSecure:             "auth_cookie_secure",
		CookieDomain:             "auth_cookie_domain",
		CSPEnabled:               "auth_csp_enabled",
		EmailCodeTTLSeconds:      "auth_email_code_ttl_seconds",
		EmailCodeCooldownSeconds: "auth_email_code_cooldown_seconds",
		LoginMaxFailAttempts:     "auth_login_max_fail_attempts",
		LoginLockSeconds:         "auth_login_lock_seconds",
	}
}

// ResolveAuthConfig 以 yamlBase 为底，用已启用的数据字典项覆盖（字典存在则优先）。
// 说明：JWT 中间件在启动时读取；改 auth_jwt_secret 后需重启进程。TTL/Cookie 等同理在启动覆盖。
func ResolveAuthConfig(ctx context.Context, db *gorm.DB, yamlBase config.AuthConfig, types AuthDictTypes) config.AuthConfig {
	cfg := yamlBase
	if db == nil {
		return cfg
	}

	snap := fetchDictSnapshot(ctx, db,
		types.JWTSecret, types.AccessTokenTTLMinutes, types.RefreshTokenTTLHours,
		types.CookieSecure, types.CookieDomain, types.CSPEnabled,
		types.EmailCodeTTLSeconds, types.EmailCodeCooldownSeconds,
		types.LoginMaxFailAttempts, types.LoginLockSeconds,
	)

	if v, ok := snap.getNonEmpty(types.JWTSecret); ok {
		cfg.JWTSecret = v
	}
	if v, ok := snap.get(types.AccessTokenTTLMinutes); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.AccessTokenTTLMinutes = n
		}
	}
	if v, ok := snap.get(types.RefreshTokenTTLHours); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.RefreshTokenTTLHours = n
		}
	}
	if v, ok := snap.get(types.CookieSecure); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.CookieSecure = &b
		}
	}
	if v, ok := snap.get(types.CookieDomain); ok {
		cfg.CookieDomain = v
	}
	if v, ok := snap.get(types.CSPEnabled); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.CSPEnabled = &b
		}
	}
	if v, ok := snap.get(types.EmailCodeTTLSeconds); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.EmailCodeTTLSeconds = n
		}
	}
	if v, ok := snap.get(types.EmailCodeCooldownSeconds); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.EmailCodeCooldownSeconds = n
		}
	}
	if v, ok := snap.get(types.LoginMaxFailAttempts); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.LoginMaxFailAttempts = n
		}
	}
	if v, ok := snap.get(types.LoginLockSeconds); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.LoginLockSeconds = n
		}
	}
	return cfg
}
