package system

import (
	"context"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/password"
)

// PasswordPolicyResolver 由装配层注入，避免 Service 直持 *gorm.DB 读字典。
type PasswordPolicyResolver func(ctx context.Context) dictconfig.PasswordPolicyConfig

func enforcePasswordComplexity(ctx context.Context, resolve PasswordPolicyResolver, raw, username string) error {
	cfg := resolve(ctx)
	if err := password.ValidateComplexity(raw, username, cfg); err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	return nil
}

func userPasswordExpired(ctx context.Context, resolve PasswordPolicyResolver, user *model.User) bool {
	if user == nil {
		return false
	}
	cfg := resolve(ctx)
	return password.IsExpired(user.PasswordChangedAt, user.CreatedAt, cfg.ExpiryDays, time.Now())
}

func touchPasswordChanged(user *model.User) {
	if user == nil {
		return
	}
	now := time.Now()
	user.PasswordChangedAt = &now
	user.MustChangePassword = false
}

func passwordPolicyAPIResponse(ctx context.Context, resolve PasswordPolicyResolver) PasswordPolicyResponse {
	cfg := resolve(ctx)
	sum := dictconfig.PasswordPolicySummary(cfg)
	hint, _ := sum["hint"].(string)
	expiryHint, _ := sum["expiry_hint"].(string)
	return PasswordPolicyResponse{
		MinLength:      cfg.MinLength,
		MaxLength:      cfg.MaxLength,
		RequireUpper:   cfg.RequireUpper,
		RequireLower:   cfg.RequireLower,
		RequireDigit:   cfg.RequireDigit,
		RequireSpecial: cfg.RequireSpecial,
		ExpiryDays:     cfg.ExpiryDays,
		ForbidUsername: cfg.ForbidUsername,
		Hint:           hint,
		ExpiryHint:     expiryHint,
	}
}
