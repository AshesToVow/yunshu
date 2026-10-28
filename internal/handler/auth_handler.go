package handler

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service   *service.AuthService
	loginLogs *service.LoginLogService
	enforcer  *casbin.SyncedEnforcer
}

// NewAuthHandler 创建相关逻辑。
func NewAuthHandler(service *service.AuthService, loginLogs *service.LoginLogService, enforcer *casbin.SyncedEnforcer) *AuthHandler {
	return &AuthHandler{service: service, loginLogs: loginLogs, enforcer: enforcer}
}

// PermissionCapability 当前用户可直接调用的 API 能力（按钮级权限据此收敛）。
type PermissionCapability struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// PermissionsResponse 当前用户的 API 能力集合。
type PermissionsResponse struct {
	// SuperAdmin 为 true 时前端可放行全部按钮，无需逐条比对。
	SuperAdmin  bool                  `json:"super_admin"`
	Permissions []PermissionCapability `json:"permissions"`
}

// Permissions godoc
// @Summary 当前用户的 API 能力集合
// @Description 返回当前用户经角色继承得到的 Casbin 权限点，供前端做按钮级显隐；超管返回 super_admin=true。
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=PermissionsResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Router /api/v1/auth/permissions [get]
func (h *AuthHandler) Permissions(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || user == nil {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	resp := PermissionsResponse{SuperAdmin: auth.IsSuperAdminRole(user.RoleCodes), Permissions: []PermissionCapability{}}
	if resp.SuperAdmin || h.enforcer == nil {
		response.Success(c, resp)
		return
	}
	rules, err := h.enforcer.GetImplicitPermissionsForUser(service.UserSubject(user.ID))
	if err != nil {
		response.Error(c, bizerrors.Pass(c.Request.Context(), "auth", "Permissions", err))
		return
	}
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		if len(rule) < 3 {
			continue
		}
		resource := strings.TrimSpace(rule[1])
		action := strings.ToUpper(strings.TrimSpace(rule[2]))
		if resource == "" || action == "" {
			continue
		}
		key := action + " " + resource
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		resp.Permissions = append(resp.Permissions, PermissionCapability{Resource: resource, Action: action})
	}
	response.Success(c, resp)
}

func loginErrMessage(err error) string {
	if biz, ok := bizerrors.As(err); ok {
		return biz.Message
	}
	return "登录失败"
}

func (h *AuthHandler) recordLogin(c *gin.Context, username, source string, success bool, detail string, userID *uint) {
	if h.loginLogs == nil || username == "" {
		return
	}
	status := model.LoginLogStatusFail
	if success {
		status = model.LoginLogStatusSuccess
	}
	entry := model.LoginLog{
		Username:  username,
		IP:        c.ClientIP(),
		Status:    status,
		Detail:    detail,
		UserAgent: c.GetHeader("User-Agent"),
		Source:    source,
		UserID:    userID,
	}
	_ = h.loginLogs.Record(c.Request.Context(), entry)
}

// SendEmailCode godoc
// @Summary Send verification code
// @Description Send an email verification code for login or registration and cache it in Redis.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendEmailCodeRequest true "Send email code request"
// @Success 200 {object} response.Body{data=service.SendEmailCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 409 {object} response.Body "email already registered"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/verification-code [post]
func (h *AuthHandler) SendEmailCode(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.SendEmailCodeRequest) (*service.SendEmailCodeResponse, error) {
		return h.service.SendEmailCodeWithIP(ctx, service.SendEmailCodeWithIPRequest{
			SendEmailCodeRequest: req,
			ClientIP:             c.ClientIP(),
		})
	})
}

// SendLoginCodeByUsername godoc
// @Summary Send login verification code by username
// @Description Send an email verification code to the user's registered email address for login.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendLoginCodeByUsernameRequest true "Send login code by username request"
// @Success 200 {object} response.Body{data=service.SendEmailCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/login-code [post]
func (h *AuthHandler) SendLoginCodeByUsername(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.SendLoginCodeByUsernameRequest) (*service.SendEmailCodeResponse, error) {
		return h.service.SendLoginCodeByUsernameWithIP(ctx, req, c.ClientIP())
	})
}

// SendPasswordLoginCode godoc
// @Summary Send password login verification code
// @Description Generate a 6-digit verification code for password login and cache it in Redis. The code is returned in the response for demonstration purposes in development. In production, it would be sent via email or SMS.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.SendPasswordLoginCodeRequest true "Send password login code request"
// @Success 200 {object} response.Body{data=service.SendPasswordLoginCodeResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 409 {object} response.Body "cooldown in effect"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/password-login-code [post]
func (h *AuthHandler) SendPasswordLoginCode(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.SendPasswordLoginCodeRequest) (*service.SendPasswordLoginCodeResponse, error) {
		return h.service.SendPasswordLoginCode(ctx, req.Username)
	})
}

// Login godoc
// @Summary Login with password
// @Description Login with username, password and verification code. This is kept as a fallback entry for administrators.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Password login request"
// @Success 200 {object} response.Body{data=service.LoginResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		h.recordLogin(c, req.Username, model.LoginSourcePassword, false, loginErrMessage(err), nil)
		response.Error(c, err)
		return
	}
	// 已启用 MFA 时不发会话、不记登录成功，等第二因子通过后再落日志。
	if data.MFARequired {
		response.Success(c, publicLoginResponse(data))
		return
	}
	uid := data.User.ID
	h.recordLogin(c, data.User.Username, model.LoginSourcePassword, true, "登录成功", &uid)
	h.applySessionCookies(c, data.AccessToken, data.RefreshToken, data.ExpiresAt, data.RefreshExpiresAt)
	response.Success(c, publicLoginResponse(data))
}

// VerifyMFA godoc
// @Summary 登录二次验证
// @Description 使用登录返回的 mfa_challenge 与 TOTP/恢复码换取会话。
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.MFAVerifyRequest true "二次验证请求"
// @Success 200 {object} response.Body{data=service.LoginResponse} "success"
// @Failure 400 {object} response.Body "验证码不正确或挑战已过期"
// @Router /api/v1/auth/mfa/verify [post]
func (h *AuthHandler) VerifyMFA(c *gin.Context) {
	var req service.MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.service.VerifyMFALogin(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	uid := data.User.ID
	source := data.LoginSource
	if source == "" {
		source = model.LoginSourcePassword
	}
	h.recordLogin(c, data.User.Username, source, true, "二次验证通过", &uid)
	h.applySessionCookies(c, data.AccessToken, data.RefreshToken, data.ExpiresAt, data.RefreshExpiresAt)
	response.Success(c, publicLoginResponse(data))
}

// SetupMFA godoc
// @Summary 生成 TOTP 密钥
// @Description 校验当前密码后生成待确认的 TOTP 密钥与 otpauth 链接。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.MFASetupRequest true "校验密码"
// @Success 200 {object} response.Body{data=service.MFASetupResponse} "success"
// @Router /api/v1/auth/mfa/setup [post]
func (h *AuthHandler) SetupMFA(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || user == nil {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.MFASetupRequest) (*service.MFASetupResponse, error) {
		return h.service.SetupMFA(ctx, user.ID, req)
	})
}

// EnableMFA godoc
// @Summary 启用 TOTP
// @Description 校验一次 TOTP 后启用二次验证，并一次性返回恢复码。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.MFAEnableRequest true "TOTP 验证码"
// @Success 200 {object} response.Body{data=service.MFAEnableResponse} "success"
// @Router /api/v1/auth/mfa/enable [post]
func (h *AuthHandler) EnableMFA(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || user == nil {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.MFAEnableRequest) (*service.MFAEnableResponse, error) {
		return h.service.EnableMFA(ctx, user.ID, req)
	})
}

// DisableMFA godoc
// @Summary 关闭 TOTP
// @Description 需要同时校验当前密码与 TOTP/恢复码。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.MFADisableRequest true "关闭二次验证"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Router /api/v1/auth/mfa/disable [post]
func (h *AuthHandler) DisableMFA(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || user == nil {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSONOK(c, gin.H{"message": "已关闭二次验证"}, func(ctx context.Context, req service.MFADisableRequest) error {
		return h.service.DisableMFA(ctx, user.ID, req)
	})
}

// EmailLogin godoc
// @Summary Login with email verification code
// @Description Login with email and a 6-digit verification code delivered through SMTP and cached in Redis.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.EmailLoginRequest true "Email verification login request"
// @Success 200 {object} response.Body{data=service.LoginResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 403 {object} response.Body "无访问权限"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/email-login [post]
func (h *AuthHandler) EmailLogin(c *gin.Context) {
	var req service.EmailLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.service.EmailLogin(c.Request.Context(), req)
	if err != nil {
		h.recordLogin(c, req.Email, model.LoginSourceEmail, false, loginErrMessage(err), nil)
		response.Error(c, err)
		return
	}
	if data.MFARequired {
		response.Success(c, publicLoginResponse(data))
		return
	}
	uid := data.User.ID
	h.recordLogin(c, data.User.Username, model.LoginSourceEmail, true, "登录成功", &uid)
	h.applySessionCookies(c, data.AccessToken, data.RefreshToken, data.ExpiresAt, data.RefreshExpiresAt)
	response.Success(c, publicLoginResponse(data))
}

// Register godoc
// @Summary Register with email verification code
// @Description Register a new user by submitting username, email, password and the verification code sent by email.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Email registration request"
// @Success 201 {object} response.Body{data=service.RegisterResponse} "created"
// @Failure 400 {object} response.Body "bad request"
// @Failure 409 {object} response.Body "conflict"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	ServeJSON201(c, h.service.Register)
}

// Logout godoc
// @Summary Logout
// @Description Logout the current user by invalidating the current token and clearing auth cookies.
// @Tags Auth
// @Produce json
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	tokenID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		tokenID = claims.TokenID
	} else if raw := auth.ExtractAccessToken(c); raw != "" {
		if claims, err := auth.ParseToken(h.service.JWTSecret(), raw); err == nil {
			tokenID = claims.TokenID
		}
	}
	refreshToken := auth.ExtractRefreshToken(c)
	if err := h.service.Logout(c.Request.Context(), tokenID, refreshToken); err != nil {
		// 仍清 Cookie，避免客户端卡死在半失效态
		h.clearSessionCookies(c)
		response.Error(c, err)
		return
	}
	h.clearSessionCookies(c)
	response.Success(c, gin.H{"message": "退出登录成功"})
}

// Refresh godoc
// @Summary Refresh session cookies
// @Description Rotate refresh token (Redis one-time) and issue new HttpOnly access/refresh cookies.
// @Tags Auth
// @Produce json
// @Success 200 {object} response.Body{data=service.RefreshResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	rt := auth.ExtractRefreshToken(c)
	if rt == "" {
		response.Error(c, constants.ErrLoginSessionExpired)
		return
	}
	data, err := h.service.RefreshSession(c.Request.Context(), rt)
	if err != nil {
		h.clearSessionCookies(c)
		response.Error(c, err)
		return
	}
	h.applySessionCookies(c, data.AccessToken, data.RefreshToken, data.ExpiresAt, data.RefreshExpiresAt)
	response.Success(c, service.RefreshResponse{
		ExpiresAt:        data.ExpiresAt,
		RefreshExpiresAt: data.RefreshExpiresAt,
	})
}

func publicLoginResponse(data *service.LoginResponse) service.LoginResponse {
	if data == nil {
		return service.LoginResponse{}
	}
	out := *data
	out.Token = ""
	out.AccessToken = ""
	out.RefreshToken = ""
	return out
}

func (h *AuthHandler) applySessionCookies(c *gin.Context, access, refresh string, accessExp, refreshExp time.Time) {
	auth.WriteSessionCookies(c, auth.SessionCookies{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		Secure:           h.service.CookieSecure(),
		Domain:           h.service.CookieDomain(),
	})
}

func (h *AuthHandler) clearSessionCookies(c *gin.Context) {
	auth.ClearSessionCookies(c, h.service.CookieSecure(), 0, h.service.CookieDomain())
}

// Me godoc
// @Summary Get current user
// @Description Get the profile of the current authenticated user.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 404 {object} response.Body "用户不存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}

	data, err := h.service.Me(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

// UpdateProfile godoc
// @Summary 更新当前用户资料
// @Description 更新当前登录用户的昵称和邮箱。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.UpdateProfileRequest true "更新资料请求"
// @Success 200 {object} response.Body{data=service.UserDetailResponse} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 409 {object} response.Body "邮箱已存在"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/me [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.UpdateProfileRequest) (*service.UserDetailResponse, error) {
		return h.service.UpdateProfile(ctx, user.ID, req)
	})
}

// ChangePassword godoc
// @Summary 修改当前用户密码
// @Description 使用旧密码校验后修改当前登录用户密码。
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} response.Body{data=MessageData} "success"
// @Failure 400 {object} response.Body "bad request"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 403 {object} response.Body "无访问权限（仅管理员角色可在个人中心修改自己的密码）"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSONOK(c, gin.H{"message": "密码修改成功"}, func(ctx context.Context, req service.ChangePasswordRequest) error {
		return h.service.ChangePassword(ctx, user.ID, req)
	})
}

// GetPasswordPolicy godoc
// @Summary 获取当前密码策略
// @Tags Auth
// @Produce json
// @Success 200 {object} response.Body
// @Router /api/v1/auth/password-policy [get]
func (h *AuthHandler) GetPasswordPolicy(c *gin.Context) {
	response.Success(c, h.service.GetPasswordPolicy(c.Request.Context()))
}

// CreateWSTicket godoc
// @Summary Issue one-time WebSocket handshake ticket
// @Description Exchange current Bearer session for a short-lived, single-use ticket used in WebSocket query (avoids JWT in URL).
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.CreateWSTicketRequest false "optional scope for audit"
// @Success 200 {object} response.Body{data=service.WSTicketResponse} "success"
// @Failure 401 {object} response.Body "未登录或登录已失效"
// @Failure 500 {object} response.Body "服务器内部错误"
// @Router /api/v1/auth/ws-ticket [post]
func (h *AuthHandler) CreateWSTicket(c *gin.Context) {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		response.Error(c, constants.ErrUnauthorized)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.CreateWSTicketRequest) (*service.WSTicketResponse, error) {
		return h.service.CreateWSTicket(ctx, claims.UserID, claims.TokenID, req.Scope)
	})
}
