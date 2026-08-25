package password

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"yunshu/internal/dictconfig"
)

// ValidateComplexity 按策略校验明文密码。
func ValidateComplexity(raw, username string, cfg dictconfig.PasswordPolicyConfig) error {
	pw := strings.TrimSpace(raw)
	if cfg.MinLength <= 0 {
		cfg.MinLength = 8
	}
	if cfg.MaxLength <= 0 {
		cfg.MaxLength = 64
	}
	if len(pw) < cfg.MinLength {
		return fmt.Errorf("密码长度至少 %d 位", cfg.MinLength)
	}
	if len(pw) > cfg.MaxLength {
		return fmt.Errorf("密码长度不能超过 %d 位", cfg.MaxLength)
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	var missing []string
	if cfg.RequireUpper && !hasUpper {
		missing = append(missing, "大写字母")
	}
	if cfg.RequireLower && !hasLower {
		missing = append(missing, "小写字母")
	}
	if cfg.RequireDigit && !hasDigit {
		missing = append(missing, "数字")
	}
	if cfg.RequireSpecial && !hasSpecial {
		missing = append(missing, "特殊字符")
	}
	if len(missing) > 0 {
		return fmt.Errorf("密码须包含：%s", strings.Join(missing, "、"))
	}
	if cfg.ForbidUsername {
		u := strings.TrimSpace(username)
		if u != "" && strings.Contains(strings.ToLower(pw), strings.ToLower(u)) {
			return fmt.Errorf("密码不能包含用户名")
		}
	}
	return nil
}

// IsExpired 判断密码是否已过期。expiryDays<=0 表示不过期。
// changedAt 为空时用 fallback（通常为账号创建时间）作为起点。
func IsExpired(changedAt *time.Time, fallback time.Time, expiryDays int, now time.Time) bool {
	if expiryDays <= 0 {
		return false
	}
	base := fallback
	if changedAt != nil && !changedAt.IsZero() {
		base = *changedAt
	}
	if base.IsZero() {
		return false
	}
	return now.After(base.AddDate(0, 0, expiryDays))
}
