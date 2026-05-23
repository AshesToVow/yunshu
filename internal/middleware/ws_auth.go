package middleware

import (
	"strings"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/logutil"
	"yunshu/internal/pkg/response"
	"yunshu/internal/interfaces"
	"yunshu/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// WSAuth authenticates websocket handshake requests.
// Browsers can't set custom headers for WebSocket; use a one-time ticket from POST /api/v1/auth/ws-ticket:
//   GET .../ws?ticket=<uuid>
func WSAuth(redisClient *redis.Client, userRepo interfaces.UserRepository, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticket := strings.TrimSpace(c.Query("ticket"))
		if ticket == "" {
			response.Error(c, constants.ErrWSMissingTicketParam)
			c.Abort()
			return
		}
		if authenticateWSTicket(c, redisClient, userRepo, ticket) {
			c.Next()
		}
	}
}

func authenticateWSTicket(c *gin.Context, redisClient *redis.Client, userRepo interfaces.UserRepository, ticket string) bool {
	userID, tokenID, err := store.ConsumeWSTicket(c.Request.Context(), redisClient, ticket)
	if err != nil {
		logutil.HTTP("http.ws_auth").Warn("consume ws ticket failed", "error", err)
		response.Error(c, constants.ErrWSTicketInvalid)
		c.Abort()
		return false
	}
	if !authenticateWSSession(c, redisClient, userRepo, userID, tokenID) {
		return false
	}
	c.Set(auth.ContextClaimsKey, &auth.Claims{UserID: userID, TokenID: tokenID})
	return true
}

func authenticateWSSession(c *gin.Context, redisClient *redis.Client, userRepo interfaces.UserRepository, userID uint, tokenID string) bool {
	if err := store.ValidateAccessTokenSession(c.Request.Context(), redisClient, tokenID); err != nil {
		respondSessionStoreError(c, loggerFromGin(c), err)
		c.Abort()
		return false
	}

	user, err := userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, constants.ErrAccountPrincipalNotFound)
		c.Abort()
		return false
	}
	if user.Status != model.StatusEnabled {
		response.Error(c, constants.ErrAccountDisabled)
		c.Abort()
		return false
	}

	groupCodes := make([]string, 0, len(user.Groups))
	for _, g := range user.Groups {
		if code := strings.TrimSpace(g.Code); code != "" {
			groupCodes = append(groupCodes, code)
		}
	}
	currentUser := &auth.CurrentUser{
		ID:         user.ID,
		Username:   user.Username,
		Nickname:   user.Nickname,
		Status:     user.Status,
		RoleCodes:  model.ExtractRoleCodes(user.Roles),
		GroupCodes: groupCodes,
	}

	c.Set(auth.ContextUserKey, currentUser)
	return true
}

// loggerFromGin WS 中间件未注入 logger 时兜底 nil（respondSessionStoreError 允许 nil）。
func loggerFromGin(c *gin.Context) *logx.Logger {
	if v, ok := c.Get("logger"); ok {
		if lg, ok := v.(*logx.Logger); ok {
			return lg
		}
	}
	return nil
}
