package dictconfig

import (
	"context"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// PasswordPolicyConfig 登录密码复杂度与过期策略（数据字典可调）。
type PasswordPolicyConfig struct {
	MinLength        int
	RequireUpper     bool
	RequireLower     bool
	RequireDigit     bool
	RequireSpecial   bool
	ExpiryDays       int
	ForbidUsername   bool
	MaxLength        int
}

// PasswordPolicyDictTypes 数据字典 dict_type。
type PasswordPolicyDictTypes struct {
	MinLength      string
	RequireUpper   string
	RequireLower   string
	RequireDigit   string
	RequireSpecial string
	ExpiryDays     string
	ForbidUsername string
	MaxLength      string
}

func DefaultPasswordPolicyDictTypes() PasswordPolicyDictTypes {
	return PasswordPolicyDictTypes{
		MinLength:      "password_min_length",
		RequireUpper:   "password_require_upper",
		RequireLower:   "password_require_lower",
		RequireDigit:   "password_require_digit",
		RequireSpecial: "password_require_special",
		ExpiryDays:     "password_expiry_days",
		ForbidUsername: "password_forbid_username",
		MaxLength:      "password_max_length",
	}
}

// DefaultPasswordPolicy 默认：8 位起、大小写+数字+特殊字符、90 天过期。
func DefaultPasswordPolicy() PasswordPolicyConfig {
	return PasswordPolicyConfig{
		MinLength:      8,
		RequireUpper:   true,
		RequireLower:   true,
		RequireDigit:   true,
		RequireSpecial: true,
		ExpiryDays:     90,
		ForbidUsername: true,
		MaxLength:      64,
	}
}

// ResolvePasswordPolicy 从数据字典解析策略；缺失项用默认值。
func ResolvePasswordPolicy(ctx context.Context, db *gorm.DB) PasswordPolicyConfig {
	cfg := DefaultPasswordPolicy()
	if db == nil {
		return cfg
	}
	types := DefaultPasswordPolicyDictTypes()
	if v, ok := FetchEnabledDictValue(ctx, db, types.MinLength); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.MinLength = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.MaxLength); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n > 0 {
			cfg.MaxLength = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.RequireUpper); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.RequireUpper = b
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.RequireLower); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.RequireLower = b
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.RequireDigit); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.RequireDigit = b
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.RequireSpecial); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.RequireSpecial = b
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.ExpiryDays); ok {
		if n, ok2 := ParseIntLoose(v); ok2 && n >= 0 {
			cfg.ExpiryDays = n
		}
	}
	if v, ok := FetchEnabledDictValue(ctx, db, types.ForbidUsername); ok {
		if b, ok2 := ParseBoolLoose(v); ok2 {
			cfg.ForbidUsername = b
		}
	}
	if cfg.MaxLength < cfg.MinLength {
		cfg.MaxLength = cfg.MinLength
	}
	return cfg
}

// PasswordPolicySummary 供前端展示的策略摘要。
func PasswordPolicySummary(cfg PasswordPolicyConfig) map[string]any {
	parts := make([]string, 0, 6)
	parts = append(parts, "长度 "+strconv.Itoa(cfg.MinLength)+"-"+strconv.Itoa(cfg.MaxLength))
	if cfg.RequireUpper {
		parts = append(parts, "大写字母")
	}
	if cfg.RequireLower {
		parts = append(parts, "小写字母")
	}
	if cfg.RequireDigit {
		parts = append(parts, "数字")
	}
	if cfg.RequireSpecial {
		parts = append(parts, "特殊字符")
	}
	if cfg.ForbidUsername {
		parts = append(parts, "不可包含用户名")
	}
	expiry := "不过期"
	if cfg.ExpiryDays > 0 {
		expiry = strconv.Itoa(cfg.ExpiryDays) + " 天"
	}
	return map[string]any{
		"min_length":      cfg.MinLength,
		"max_length":      cfg.MaxLength,
		"require_upper":   cfg.RequireUpper,
		"require_lower":   cfg.RequireLower,
		"require_digit":   cfg.RequireDigit,
		"require_special": cfg.RequireSpecial,
		"expiry_days":     cfg.ExpiryDays,
		"forbid_username": cfg.ForbidUsername,
		"hint":            strings.Join(parts, "、"),
		"expiry_hint":     expiry,
	}
}
