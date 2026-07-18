package system

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/config"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/password"
	"yunshu/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	emailCodeSceneLogin    = "login"
	emailCodeSceneRegister = "register"
)

type AuthService struct {
	userRepo repositoryAuthReader
	redis    *redis.Client
	cfg      config.AuthConfig
	mailer   mailer.Sender
	appName  string
}

type repositoryAuthReader interface {
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByUsernameForAuth(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByEmailForAuth(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Save(ctx context.Context, user *model.User) error
}

// NewAuthService 创建相关逻辑。
func NewAuthService(
	userRepo repositoryAuthReader,
	redisClient *redis.Client,
	cfg config.AuthConfig,
	emailSender mailer.Sender,
	appName string,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redisClient,
		cfg:      cfg,
		mailer:   emailSender,
		appName:  appName,
	}
}

// SendEmailCode 发送相关的业务逻辑。
func (s *AuthService) SendEmailCode(ctx context.Context, req SendEmailCodeRequest) (*SendEmailCodeResponse, error) {
	email := normalizeEmail(req.Email)
	scene := strings.TrimSpace(req.Scene)

	if err := s.ensureEmailCodeDependencies(ctx); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendEmailCode", err)
	}
	if err := s.validateScenePreconditions(ctx, scene, email); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendEmailCode", err)
	}
	if err := s.ensureEmailCodeCooldown(ctx, scene, email); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendEmailCode", err)
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendEmailCode", err)
	}

	codeTTL := s.emailCodeTTL()
	cooldownTTL := s.emailCodeCooldown()
	codeKey := store.EmailCodeKey(scene, email)
	cooldownKey := store.EmailCodeCooldownKey(scene, email)

	pipe := s.redis.Pipeline()
	pipe.Set(ctx, codeKey, code, codeTTL)
	pipe.Set(ctx, cooldownKey, "1", cooldownTTL)
	if _, err = pipe.Exec(ctx); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendEmailCode", err)
	}

	subject, body := s.buildVerificationEmail(scene, code, codeTTL)
	if err = s.mailer.Send(ctx, email, subject, body); err != nil {
		_ = s.redis.Del(ctx, codeKey, cooldownKey).Err()
		return nil, bizerrors.Internalf(ctx, "auth", "SendEmailCode", err, constants.ErrMsg52c1dc6bb947)
	}

	return &SendEmailCodeResponse{
		Email:      email,
		Scene:      scene,
		ExpiresIn:  int(codeTTL.Seconds()),
		CooldownIn: int(cooldownTTL.Seconds()),
	}, nil
}

// SendEmailCodeWithIP behaves like SendEmailCode but also enforces a per-IP sending limit.
func (s *AuthService) SendEmailCodeWithIP(ctx context.Context, req SendEmailCodeWithIPRequest) (*SendEmailCodeResponse, error) {
	// enforce per-IP send limit (e.g., 20 sends per minute)
	if s.redis != nil {
		ipKey := store.EmailSendIPKey(req.ClientIP)
		limit := int64(20)
		if n, err := s.redis.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				s.redis.Expire(ctx, ipKey, time.Minute)
			}
			if n > limit {
				return nil, constants.ErrCaptchaIPRateLimited
			}
		}
	}

	// Delegate to existing logic
	return s.SendEmailCode(ctx, SendEmailCodeRequest{Email: req.Email, Scene: req.Scene})
}

// SendLoginCodeByUsername 发送相关的业务逻辑。
func (s *AuthService) SendLoginCodeByUsername(ctx context.Context, req SendLoginCodeByUsernameRequest) (*SendEmailCodeResponse, error) {
	username := strings.TrimSpace(req.Username)
	user, err := s.userRepo.GetByUsernameForAuth(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "SendLoginCodeByUsername", err)
	}

	if user.Status != model.StatusEnabled {
		return nil, constants.ErrAccountDisabled
	}

	// Reuse SendEmailCode logic with the user's email
	if user.Email == nil {
		return nil, constants.ErrEmailNotBound
	}
	return s.SendEmailCode(ctx, SendEmailCodeRequest{
		Email: *user.Email,
		Scene: emailCodeSceneLogin,
	})
}

// Login 登录相关的业务逻辑。
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	user, err := s.userRepo.GetByUsernameForAuth(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, bizerrors.Reject(ctx, "auth", "Login", constants.ErrPasswordIncorrect, "reason", "user_not_found", "username", username)
		}
		return nil, bizerrors.Pass(ctx, "auth", "Login", err, "username", username)
	}

	if user.Status != model.StatusEnabled {
		return nil, constants.ErrAccountDisabled
	}

	// 密码校验前先检查是否处于失败锁定期，防止暴力破解。
	if err = s.ensureLoginNotLocked(ctx, username); err != nil {
		return nil, bizerrors.Reject(ctx, "auth", "Login", err, "reason", "locked", "username", username)
	}

	if err = password.Compare(user.Password, req.Password); err != nil {
		s.recordLoginFailure(ctx, username)
		return nil, bizerrors.Reject(ctx, "auth", "Login", constants.ErrPasswordIncorrect, "reason", "bad_password", "username", username)
	}

	// Validate password login code
	if err = s.validatePasswordLoginCode(ctx, req.CaptchaKey, req.Code); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "Login", err)
	}
	s.clearPasswordLoginCode(ctx, req.CaptchaKey)
	s.clearLoginFailures(ctx, username)

	return s.issueLoginResponse(ctx, user)
}

// EmailLogin 执行对应的业务逻辑。
func (s *AuthService) EmailLogin(ctx context.Context, req EmailLoginRequest) (*LoginResponse, error) {
	email := normalizeEmail(req.Email)
	user, err := s.userRepo.GetByEmailForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "EmailLogin", err)
	}

	if user.Status != model.StatusEnabled {
		return nil, constants.ErrAccountDisabled
	}
	if err = s.validateEmailCode(ctx, emailCodeSceneLogin, email, req.Code); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "EmailLogin", err)
	}
	s.clearEmailCode(ctx, emailCodeSceneLogin, email)

	return s.issueLoginResponse(ctx, user)
}

// Register 注册相关的业务逻辑。
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	email := normalizeEmail(req.Email)
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)

	if err := s.ensureUserDoesNotExist(ctx, username, email); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "Register", err)
	}
	if err := s.validateEmailCode(ctx, emailCodeSceneRegister, email, req.Code); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "Register", err)
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "Register", err)
	}

	user := model.User{
		Username: username,
		Email:    &email,
		Password: hashedPassword,
		Nickname: nickname,
		Status:   model.StatusEnabled,
		Roles:    []model.Role{},
	}
	if err = s.userRepo.Create(ctx, &user); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "Register", err)
	}

	s.clearEmailCode(ctx, emailCodeSceneRegister, email)

	return &RegisterResponse{
		Message: "注册成功，请等待管理员审核并分配角色",
		User:    NewUserDetailResponse(user),
	}, nil
}

// Logout 退出登录相关的业务逻辑。
func (s *AuthService) Logout(ctx context.Context, tokenID string) error {
	if s.redis == nil {
		return bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil
	}
	userIDStr, err := s.redis.Get(ctx, store.AccessTokenKey(tokenID)).Result()
	if err == nil {
		if uid, parseErr := parseUintUserID(userIDStr); parseErr == nil {
			_ = store.UnregisterUserAccessToken(ctx, s.redis, uid, tokenID)
		}
	}
	return s.redis.Del(ctx, store.AccessTokenKey(tokenID)).Err()
}

// Me 执行对应的业务逻辑。
func (s *AuthService) Me(ctx context.Context, userID uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "Me", err)
	}

	response := NewUserDetailResponse(*user)
	return &response, nil
}

// UpdateProfile updates current user's profile fields.
func (s *AuthService) UpdateProfile(ctx context.Context, userID uint, req UpdateProfileRequest) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "UpdateProfile", err)
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		return nil, constants.ErrNicknameRequired
	}
	user.Nickname = nickname

	if strings.TrimSpace(req.Email) != "" {
		email := normalizeEmail(req.Email)
		existing, findErr := s.userRepo.GetByEmail(ctx, email)
		if findErr == nil && existing.ID != user.ID {
			return nil, constants.ErrEmailAlreadyRegistered
		}
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil, findErr
		}
		user.Email = &email
	}
	user.Phone = strings.TrimSpace(req.Phone)

	if err = s.userRepo.Save(ctx, user); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "UpdateProfile", err)
	}
	resp := NewUserDetailResponse(*user)
	return &resp, nil
}

// ChangePassword updates current user's password and invalidates all user tokens.
func (s *AuthService) ChangePassword(ctx context.Context, userID uint, req ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return bizerrors.Pass(ctx, "auth", "ChangePassword", err)
	}

	if err = password.Compare(user.Password, req.OldPassword); err != nil {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg0767f3889e05)
	}
	if strings.TrimSpace(req.NewPassword) == strings.TrimSpace(req.OldPassword) {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg6ca55409b3c2)
	}

	hashed, err := password.Hash(req.NewPassword)
	if err != nil {
		return bizerrors.Pass(ctx, "auth", "ChangePassword", err)
	}
	user.Password = hashed
	if err = s.userRepo.Save(ctx, user); err != nil {
		return bizerrors.Pass(ctx, "auth", "ChangePassword", err)
	}

	// 使该用户的所有 token 失效（强制重新登录）
	if s.redis != nil {
		if err := store.InvalidateAllUserAccessTokens(ctx, s.redis, userID); err != nil {
			return bizerrors.Pass(ctx, "auth", "ChangePassword", err)
		}
	}

	return nil
}

func (s *AuthService) issueLoginResponse(ctx context.Context, user *model.User) (*LoginResponse, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute)
	tokenID := uuid.NewString()

	token, err := auth.GenerateToken(s.cfg.JWTSecret, auth.Claims{
		UserID:   user.ID,
		Username: user.Username,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "issueLoginResponse", err)
	}

	if s.redis == nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}
	if err = s.redis.Set(ctx, store.AccessTokenKey(tokenID), user.ID, time.Until(expiresAt)).Err(); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "issueLoginResponse", err)
	}
	if err = store.RegisterUserAccessToken(ctx, s.redis, user.ID, tokenID, time.Until(expiresAt)); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "issueLoginResponse", err)
	}

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      NewUserDetailResponse(*user),
	}, nil
}

const wsTicketTTLSeconds = 30

// CreateWSTicket 为已登录用户签发短效、一次性 WebSocket 握手票据。
func (s *AuthService) CreateWSTicket(ctx context.Context, userID uint, tokenID, scope string) (*WSTicketResponse, error) {
	if s.redis == nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}
	if userID == 0 || strings.TrimSpace(tokenID) == "" {
		return nil, constants.ErrUnauthorized
	}
	ticket := uuid.NewString()
	if err := store.SaveWSTicket(ctx, s.redis, ticket, userID, tokenID, scope, wsTicketTTLSeconds); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "CreateWSTicket", err)
	}
	return &WSTicketResponse{Ticket: ticket, ExpiresIn: wsTicketTTLSeconds}, nil
}

// SendPasswordLoginCode generates captcha image using base64Captcha.
func (s *AuthService) SendPasswordLoginCode(ctx context.Context, username string) (*SendPasswordLoginCodeResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg390ccdec9f3f)
	}
	if s.redis == nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}

	// Check if user exists
	if _, err := s.userRepo.GetByUsername(ctx, username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "SendPasswordLoginCode", err)
	}

	// Check cooldown
	cooldownKey := store.PasswordLoginCodeCooldownKey(username)
	exists, err := s.redis.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendPasswordLoginCode", err)
	}
	if exists > 0 {
		ttl, err := s.redis.TTL(ctx, cooldownKey).Result()
		if err == nil && ttl > 0 {
			return &SendPasswordLoginCodeResponse{
				CaptchaKey: "", Image: "", ExpiresIn: int(s.emailCodeTTL().Seconds()),
				CooldownIn: int(ttl.Seconds()),
			}, constants.ErrCaptchaCoolingDown
		}
	}

	// 生成图形验证码。答案写入 Redis（而非进程内存），保证多副本部署下
	// 生成与校验可跨实例共享；否则不同副本的 DefaultMemStore 互不可见，校验必失败。
	driver := base64Captcha.NewDriverDigit(
		80,  // height
		240, // width
		4,   // length
		0.35, // maxSkew — lower skew for readability
		24,  // dotCount — fewer noise dots
	)
	captchaKey, answer, _ := driver.GenerateIdQuestionAnswer()
	item, err := driver.DrawCaptcha(answer)
	if err != nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsg6f15f7c820be)
	}

	codeTTL := s.emailCodeTTL()
	if err = s.redis.Set(ctx, store.PasswordLoginCodeKey(captchaKey), answer, codeTTL).Err(); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendPasswordLoginCode", err)
	}
	cooldownTTL := s.emailCodeCooldown()
	if err = s.redis.Set(ctx, cooldownKey, "1", cooldownTTL).Err(); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SendPasswordLoginCode", err)
	}

	return &SendPasswordLoginCodeResponse{
		CaptchaKey: captchaKey,
		// Keep old frontend contract: return raw base64 only (without data URL prefix).
		Image:      strings.TrimPrefix(item.EncodeB64string(), "data:image/png;base64,"),
		ExpiresIn:  int(codeTTL.Seconds()),
		CooldownIn: int(cooldownTTL.Seconds()),
	}, nil
}

// validatePasswordLoginCode 校验密码登录图形验证码。答案存于 Redis（见 SendPasswordLoginCode），
// 校验成功即删除，保证一次性且跨副本一致。
func (s *AuthService) validatePasswordLoginCode(ctx context.Context, captchaKey, code string) error {
	captchaKey = strings.TrimSpace(captchaKey)
	code = strings.TrimSpace(code)
	if captchaKey == "" || code == "" {
		return constants.ErrUnauthorizedWithMsg(constants.ErrMsgdb0b98dd46b0)
	}
	if s.redis == nil {
		return bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}

	key := store.PasswordLoginCodeKey(captchaKey)
	answer, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return constants.ErrCaptchaInvalidOrExpired
		}
		return bizerrors.Pass(ctx, "auth", "validatePasswordLoginCode", err)
	}
	if !strings.EqualFold(strings.TrimSpace(answer), code) {
		return constants.ErrCaptchaInvalidOrExpired
	}
	return nil
}

// clearPasswordLoginCode 校验通过后删除 Redis 中的一次性验证码，防止重放。
func (s *AuthService) clearPasswordLoginCode(ctx context.Context, captchaKey string) {
	captchaKey = strings.TrimSpace(captchaKey)
	if s.redis == nil || captchaKey == "" {
		return
	}
	_ = s.redis.Del(ctx, store.PasswordLoginCodeKey(captchaKey)).Err()
}

func (s *AuthService) ensureEmailCodeDependencies(ctx context.Context) error {
	if s.redis == nil {
		return bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsg1222f2978c2d)
	}
	return nil
}

func (s *AuthService) validateScenePreconditions(ctx context.Context, scene, email string) error {
	switch scene {
	case emailCodeSceneLogin:
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return constants.ErrUserNotFound
			}
			return bizerrors.Pass(ctx, "auth", "validateScenePreconditions", err)
		}
		if user.Status != model.StatusEnabled {
			return constants.ErrAccountDisabled
		}
	case emailCodeSceneRegister:
		if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
			return constants.ErrEmailAlreadyRegistered
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return bizerrors.Pass(ctx, "auth", "validateScenePreconditions", err)
		}
	default:
		return constants.ErrBadRequestWithMsg(constants.ErrMsga94172c66b0b)
	}

	return nil
}

func (s *AuthService) ensureEmailCodeCooldown(ctx context.Context, scene, email string) error {
	ttl, err := s.redis.TTL(ctx, store.EmailCodeCooldownKey(scene, email)).Result()
	if err != nil {
		return bizerrors.Pass(ctx, "auth", "ensureEmailCodeCooldown", err)
	}
	if ttl > 0 {
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte5ea7331dbac, int(ttl.Seconds())))
	}
	return nil
}

func (s *AuthService) validateEmailCode(ctx context.Context, scene, email, code string) error {
	if s.redis == nil {
		return bizerrors.InternalMsg(ctx, "auth", "api", constants.ErrMsgaf4823214b6e)
	}

	storedCode, err := s.redis.Get(ctx, store.EmailCodeKey(scene, email)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgfa17d917b702)
		}
		return bizerrors.Pass(ctx, "auth", "validateEmailCode", err)
	}

	if strings.TrimSpace(code) != storedCode {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg4f8238574720)
	}

	return nil
}

func (s *AuthService) clearEmailCode(ctx context.Context, scene, email string) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, store.EmailCodeKey(scene, email), store.EmailCodeCooldownKey(scene, email)).Err()
}

func (s *AuthService) ensureUserDoesNotExist(ctx context.Context, username, email string) error {
	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return constants.ErrEmailAlreadyRegistered
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerrors.Pass(ctx, "auth", "ensureUserDoesNotExist", err)
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return constants.ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerrors.Pass(ctx, "auth", "ensureUserDoesNotExist", err)
	}

	return nil
}

func (s *AuthService) emailCodeTTL() time.Duration {
	if s.cfg.EmailCodeTTLSeconds <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(s.cfg.EmailCodeTTLSeconds) * time.Second
}

func (s *AuthService) emailCodeCooldown() time.Duration {
	if s.cfg.EmailCodeCooldownSeconds <= 0 {
		return time.Minute
	}
	return time.Duration(s.cfg.EmailCodeCooldownSeconds) * time.Second
}

// loginLockThreshold 触发锁定的连续失败次数（<=0 视为关闭锁定）。
func (s *AuthService) loginLockThreshold() int {
	return s.cfg.LoginMaxFailAttempts
}

// loginLockDuration 锁定持续时长。
func (s *AuthService) loginLockDuration() time.Duration {
	if s.cfg.LoginLockSeconds <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(s.cfg.LoginLockSeconds) * time.Second
}

// ensureLoginNotLocked 在校验密码前检查该用户是否处于失败锁定期。
// Redis 不可用或未启用锁定（阈值<=0）时不阻断登录，避免把可用性问题升级为不可登录。
func (s *AuthService) ensureLoginNotLocked(ctx context.Context, username string) error {
	if s.redis == nil || s.loginLockThreshold() <= 0 || username == "" {
		return nil
	}
	key := store.LoginFailLockKey(username)
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		// Redis 异常不阻断登录（后续密码校验仍会拦截错误凭证）。
		return nil
	}
	if exists == 0 {
		return nil
	}
	return constants.ErrLoginTooManyAttempts
}

// recordLoginFailure 累加连续失败次数，达阈值后写入锁定标记。
func (s *AuthService) recordLoginFailure(ctx context.Context, username string) {
	if s.redis == nil || s.loginLockThreshold() <= 0 || username == "" {
		return
	}
	countKey := store.LoginFailCountKey(username)
	n, err := s.redis.Incr(ctx, countKey).Result()
	if err != nil {
		return
	}
	if n == 1 {
		// 首次失败设置计数窗口，等于锁定时长，超期自动清零。
		_ = s.redis.Expire(ctx, countKey, s.loginLockDuration()).Err()
	}
	if int(n) >= s.loginLockThreshold() {
		_ = s.redis.Set(ctx, store.LoginFailLockKey(username), "1", s.loginLockDuration()).Err()
	}
}

// clearLoginFailures 登录成功后清除失败计数与锁定标记。
func (s *AuthService) clearLoginFailures(ctx context.Context, username string) {
	if s.redis == nil || username == "" {
		return
	}
	_ = s.redis.Del(ctx, store.LoginFailCountKey(username), store.LoginFailLockKey(username)).Err()
}

func (s *AuthService) buildVerificationEmail(scene, code string, ttl time.Duration) (string, string) {
	sceneLabel := "login"
	if scene == emailCodeSceneRegister {
		sceneLabel = "registration"
	}

	subject := fmt.Sprintf("[%s] %s verification code", s.safeAppName(), strings.Title(sceneLabel))
	body := strings.Join([]string{
		fmt.Sprintf("You are using %s for %s.", s.safeAppName(), sceneLabel),
		fmt.Sprintf("Verification code: %s", code),
		fmt.Sprintf("Expires in: %d minutes", int(ttl.Minutes())),
		"If you did not request this code, you can ignore this email.",
	}, "\n")

	return subject, body
}

func (s *AuthService) safeAppName() string {
	if strings.TrimSpace(s.appName) == "" {
		return "YunShu CMDB"
	}
	return strings.TrimSpace(s.appName)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", constants.ErrBadRequestWithMsg(constants.ErrMsgb77c1b087c0b)
	}

	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}

	number, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", bizerrors.Pass(context.Background(), "auth", "generateNumericCode", err)
	}

	return fmt.Sprintf("%0*d", length, number.Int64()), nil
}

// drawLine draws a line on the image
// func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
// 	dx := abs(x2 - x1)
// 	dy := abs(y2 - y1)
// 	sx, sy := 1, 1
// 	if x1 > x2 {
// 		sx = -1
// 	}
// 	if y1 > y2 {
// 		sy = -1
// 	}
// 	err := dx - dy

// 	for {
// 		if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
// 			img.Set(x1, y1, c)
// 		}
// 		if x1 == x2 && y1 == y2 {
// 			break
// 		}
// 		e2 := 2 * err
// 		if e2 > -dy {
// 			err -= dy
// 			x1 += sx
// 		}
// 		if e2 < dx {
// 			err += dx
// 			y1 += sy
// 		}
// 	}
// }

// // drawChar draws a character on the image with larger pixels for better visibility
// func drawChar(img *image.RGBA, x, y int, ch string, c color.RGBA) {
// 	charMap := map[rune][][]bool{
// 		'0': {
// 			{true, true, true, true, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 		'1': {
// 			{false, false, true, false, false},
// 			{false, true, true, false, false},
// 			{true, true, true, false, false},
// 			{false, true, true, false, false},
// 			{false, true, true, false, false},
// 			{false, true, true, false, false},
// 			{true, true, true, true, true},
// 		},
// 		'2': {
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{true, true, true, true, true},
// 			{true, false, false, false, false},
// 			{true, false, false, false, false},
// 			{true, true, true, true, true},
// 		},
// 		'3': {
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 		'4': {
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 		},
// 		'5': {
// 			{true, true, true, true, true},
// 			{true, false, false, false, false},
// 			{true, false, false, false, false},
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 		'6': {
// 			{true, true, true, true, true},
// 			{true, false, false, false, false},
// 			{true, false, false, false, false},
// 			{true, true, true, true, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 		'7': {
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, true, false},
// 			{false, false, true, false, false},
// 			{false, true, false, false, false},
// 			{false, true, false, false, false},
// 			{false, true, false, false, false},
// 		},
// 		'8': {
// 			{true, true, true, true, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 		'9': {
// 			{true, true, true, true, true},
// 			{true, false, false, false, true},
// 			{true, false, false, false, true},
// 			{true, true, true, true, true},
// 			{false, false, false, false, true},
// 			{false, false, false, false, true},
// 			{true, true, true, true, true},
// 		},
// 	}

// 	if bitmap, ok := charMap[rune(ch[0])]; ok {
// 		pixelSize := 6
// 		for i, row := range bitmap {
// 			for j, pixel := range row {
// 				if pixel {
// 					for py := 0; py < pixelSize; py++ {
// 						for px := 0; px < pixelSize; px++ {
// 							tx := x + j*pixelSize + px
// 							ty := y + i*pixelSize + py
// 							if tx >= 0 && tx < img.Bounds().Dx() && ty >= 0 && ty < img.Bounds().Dy() {
// 								img.Set(tx, ty, c)
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// }

// abs returns the absolute value of x
// func abs(x int) int {
// 	if x < 0 {
// 		return -x
// 	}
// 	return x
// }

func parseUintUserID(raw string) (uint, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
