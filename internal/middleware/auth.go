package middleware

import (
	"errors"
	"strings"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/password"
	"yunshu/internal/pkg/response"
	"yunshu/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func respondSessionStoreError(c *gin.Context, _ *logx.Logger, err error) {
	switch {
	case errors.Is(err, store.ErrSessionNotFound):
		response.Error(c, constants.ErrLoginSessionExpired)
	case errors.Is(err, store.ErrRedisRequired), errors.Is(err, store.ErrRedisUnavailable):
		logx.With(c.Request.Context(), "component", "http.auth").Error("redis session validation failed", "error", err)
		response.Error(c, constants.ErrInternal)
	default:
		logx.With(c.Request.Context(), "component", "http.auth").Error("session validation failed", "error", err)
	}
}

func passwordChangeAllowed(method, fullPath string) bool {
	p := strings.TrimSpace(fullPath)
	switch {
	case method == "PUT" && strings.HasSuffix(p, "/auth/password"):
		return true
	case method == "GET" && strings.HasSuffix(p, "/auth/me"):
		return true
	case method == "GET" && strings.HasSuffix(p, "/auth/password-policy"):
		return true
	case method == "POST" && strings.HasSuffix(p, "/auth/logout"):
		return true
	case method == "POST" && strings.HasSuffix(p, "/auth/ws-ticket"):
		return true
	default:
		return false
	}
}

func Auth(secret string, redisClient *redis.Client, userRepo interfaces.UserRepository, logger *logx.Logger, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := auth.ExtractAccessToken(c)
		if tokenString == "" {
			response.Error(c, constants.ErrMissingAuthHeader)
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(secret, tokenString)
		if err != nil {
			logx.With(c.Request.Context(), "component", "http.auth").Warn("parse token failed", "error", err)
			response.Error(c, constants.ErrAccessTokenInvalid)
			c.Abort()
			return
		}

		if err = store.ValidateAccessTokenSession(c.Request.Context(), redisClient, claims.TokenID); err != nil {
			respondSessionStoreError(c, logger, err)
			c.Abort()
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), claims.UserID)
		if err != nil {
			response.Error(c, constants.ErrAccountPrincipalNotFound)
			c.Abort()
			return
		}
		if user.Status != model.StatusEnabled {
			response.Error(c, constants.ErrAccountDisabled)
			c.Abort()
			return
		}

		groupCodes := make([]string, 0, len(user.Groups))
		for _, g := range user.Groups {
			if g.Status == model.StatusDisabled {
				continue
			}
			if code := strings.TrimSpace(g.Code); code != "" {
				groupCodes = append(groupCodes, code)
			}
		}
		cfg := dictconfig.ResolvePasswordPolicy(c.Request.Context(), db)
		expired := password.IsExpired(user.PasswordChangedAt, user.CreatedAt, cfg.ExpiryDays, time.Now())
		mustChange := user.MustChangePassword || expired
		currentUser := &auth.CurrentUser{
			ID:                 user.ID,
			Username:           user.Username,
			Nickname:           user.Nickname,
			Status:             user.Status,
			DepartmentID:       user.DepartmentID,
			RoleCodes:          model.ExtractEnabledRoleCodes(user.Roles),
			GroupCodes:         groupCodes,
			MustChangePassword: mustChange,
		}

		c.Set(auth.ContextClaimsKey, claims)
		c.Set(auth.ContextUserKey, currentUser)
		c.Request = c.Request.WithContext(logx.WithUser(c.Request.Context(), user.ID, user.Username))

		if mustChange && !passwordChangeAllowed(c.Request.Method, c.FullPath()) {
			response.Error(c, constants.ErrBadRequestWithMsg("密码已过期或需强制修改，请先修改密码后再继续操作"))
			c.Abort()
			return
		}
		c.Next()
	}
}
