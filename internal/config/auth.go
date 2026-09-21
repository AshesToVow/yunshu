package config

import "strings"

// ApplyDefaults 填充 Auth 零值字段；CookieSecure 依赖 appEnv 推断生产环境。
func (c *AuthConfig) ApplyDefaults(appEnv string) {
	if c.AccessTokenTTLMinutes <= 0 {
		c.AccessTokenTTLMinutes = 15
	}
	if c.RefreshTokenTTLHours <= 0 {
		c.RefreshTokenTTLHours = 168 // 7d
	}
	if c.EmailCodeTTLSeconds <= 0 {
		c.EmailCodeTTLSeconds = 600
	}
	if c.EmailCodeCooldownSeconds <= 0 {
		c.EmailCodeCooldownSeconds = 60
	}
	if c.LoginMaxFailAttempts <= 0 {
		c.LoginMaxFailAttempts = 5
	}
	if c.LoginLockSeconds <= 0 {
		c.LoginLockSeconds = 900
	}
	if c.CSPEnabled == nil {
		v := true
		c.CSPEnabled = &v
	}
	if c.CookieSecure == nil {
		e := strings.ToLower(strings.TrimSpace(appEnv))
		v := e == "prod" || e == "production"
		c.CookieSecure = &v
	}
}
