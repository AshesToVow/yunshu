package system

import (
	"context"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/password"

	"gorm.io/gorm"
)

func resolvePasswordPolicy(ctx context.Context, db *gorm.DB) dictconfig.PasswordPolicyConfig {
	return dictconfig.ResolvePasswordPolicy(ctx, db)
}

func enforcePasswordComplexity(ctx context.Context, db *gorm.DB, raw, username string) error {
	cfg := resolvePasswordPolicy(ctx, db)
	if err := password.ValidateComplexity(raw, username, cfg); err != nil {
		return constants.ErrBadRequestWithMsg(err.Error())
	}
	return nil
}

func userPasswordExpired(ctx context.Context, db *gorm.DB, user *model.User) bool {
	if user == nil {
		return false
	}
	cfg := resolvePasswordPolicy(ctx, db)
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

func passwordPolicyAPIResponse(ctx context.Context, db *gorm.DB) PasswordPolicyResponse {
	cfg := resolvePasswordPolicy(ctx, db)
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
