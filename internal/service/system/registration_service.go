package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"
	"yunshu/internal/store"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RegistrationService struct {
	regRepo       interfaces.RegistrationRequestRepository
	userRepo      interfaces.UserRepository
	redis         *redis.Client
	resolvePolicy PasswordPolicyResolver
	authCfg       config.AuthConfig
	mailer        mailer.Sender
	appName       string
	// writeMu 串行化提交申请，避免“先查后插”并发产生重复待审申请。
	writeMu sync.Mutex
}

// NewRegistrationService 创建相关逻辑。
func NewRegistrationService(
	regRepo interfaces.RegistrationRequestRepository,
	userRepo interfaces.UserRepository,
	redis *redis.Client,
	resolvePolicy PasswordPolicyResolver,
	authCfg config.AuthConfig,
	mailer mailer.Sender,
	appName string,
) *RegistrationService {
	return &RegistrationService{
		regRepo:       regRepo,
		userRepo:      userRepo,
		redis:         redis,
		resolvePolicy: resolvePolicy,
		authCfg:       authCfg,
		mailer:        mailer,
		appName:       appName,
	}
}

type ApplyRegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email,max=128"`
	Nickname string `json:"nickname" binding:"required,max=128"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
}

type ReviewRequest struct {
	Status  int    `json:"status" binding:"required,oneof=1 2"`
	Comment string `json:"comment"`
}

// Apply 提交申请相关的业务逻辑。
func (s *RegistrationService) Apply(ctx context.Context, req ApplyRegisterRequest) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	email := normalizeEmail(req.Email)
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)

	if err := s.ensureRegistrationDoesNotExist(ctx, username, email); err != nil {
		return bizerrors.Pass(ctx, "registration", "Apply", err)
	}
	if err := s.validateEmailCode(ctx, emailCodeSceneRegister, email, req.Code); err != nil {
		return bizerrors.Pass(ctx, "registration", "Apply", err)
	}
	if err := enforcePasswordComplexity(ctx, s.resolvePolicy, req.Password, username); err != nil {
		return err
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return bizerrors.Pass(ctx, "registration", "Apply", err)
	}

	regReq := model.RegistrationRequest{
		Username: username,
		Email:    email,
		Nickname: nickname,
		Password: hashedPassword,
		Status:   model.RegistrationPending,
	}

	if err = s.regRepo.Create(ctx, &regReq); err != nil {
		return bizerrors.Pass(ctx, "registration", "Apply", err)
	}

	s.clearEmailCode(ctx, emailCodeSceneRegister, email)
	return nil
}

// List 查询列表相关的业务逻辑。
func (s *RegistrationService) List(ctx context.Context, keyword string, status *int, page, pageSize int) ([]model.RegistrationRequest, int64, error) {
	list, total, err := s.regRepo.List(ctx, repository.RegistrationRequestListParams{
		Keyword:  keyword,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, bizerrors.Pass(ctx, "registration", "List", err)
	}
	s.enrichReviewerUsernames(ctx, list)
	return list, total, nil
}

func (s *RegistrationService) enrichReviewerUsernames(ctx context.Context, list []model.RegistrationRequest) {
	ids := make([]uint, 0, len(list))
	seen := make(map[uint]struct{}, len(list))
	for _, item := range list {
		if item.ReviewerID == nil || *item.ReviewerID == 0 {
			continue
		}
		id := *item.ReviewerID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil {
		return
	}
	names := make(map[uint]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Username
	}
	for i := range list {
		if list[i].ReviewerID == nil {
			continue
		}
		if name, ok := names[*list[i].ReviewerID]; ok {
			list[i].ReviewerUsername = name
		}
	}
}

// Review 执行对应的业务逻辑。
func (s *RegistrationService) Review(ctx context.Context, id uint, reviewerID uint, req ReviewRequest) error {
	regReq, err := s.regRepo.GetByID(ctx, id)
	if err != nil {
		return constants.ErrRegistrationRequestNotFound
	}
	if regReq.Status != model.RegistrationPending {
		return constants.ErrRegistrationAlreadyProcessed
	}

	newStatus := model.RegistrationRequestStatus(req.Status)
	if newStatus == model.RegistrationApproved {
		now := time.Now()
		user := model.User{
			Username:          regReq.Username,
			Email:             &regReq.Email,
			Password:          regReq.Password,
			PasswordChangedAt: &now,
			Nickname:          regReq.Nickname,
			Status:            model.StatusEnabled,
			Roles:             []model.Role{},
		}
		// 状态更新与建号同事务：建号失败则申请保持待审，避免“已通过却无账号”。
		if err := s.regRepo.ApproveWithUser(ctx, id, reviewerID, req.Comment, &user); err != nil {
			if errors.Is(err, repository.ErrRegistrationNotPending) {
				return constants.ErrRegistrationAlreadyProcessed
			}
			return bizerrors.Pass(ctx, "registration", "Review", err)
		}
		return nil
	}

	if err := s.regRepo.UpdateStatus(ctx, id, newStatus, reviewerID, req.Comment); err != nil {
		if errors.Is(err, repository.ErrRegistrationNotPending) {
			return constants.ErrRegistrationAlreadyProcessed
		}
		return bizerrors.Pass(ctx, "registration", "Review", err)
	}
	return nil
}

func (s *RegistrationService) ensureRegistrationDoesNotExist(ctx context.Context, username, email string) error {
	count, err := s.regRepo.CountPending(ctx, username, email)
	if err != nil {
		return bizerrors.Pass(ctx, "registration", "ensureRegistrationDoesNotExist", err)
	}
	if count > 0 {
		return constants.ErrRegistrationDuplicatePending
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return constants.ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerrors.Pass(ctx, "registration", "ensureRegistrationDoesNotExist", err)
	}

	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return constants.ErrEmailAlreadyRegistered
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return bizerrors.Pass(ctx, "registration", "ensureRegistrationDoesNotExist", err)
	}

	return nil
}

func (s *RegistrationService) validateEmailCode(ctx context.Context, scene, email, code string) error {
	if s.redis == nil {
		// 无 Redis 无法校验验证码，必须拒绝而不是放行任意验证码。
		return bizerrors.InternalMsg(ctx, "registration", "api", constants.ErrMsgaf4823214b6e)
	}
	key := store.EmailCodeKey(scene, email)
	val, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return constants.ErrCaptchaExpired
		}
		return bizerrors.Pass(ctx, "registration", "validateEmailCode", err)
	}
	if val != code {
		return s.burnEmailCodeOnVerifyFailure(ctx, scene, email)
	}
	return nil
}

// burnEmailCodeOnVerifyFailure 累计校验失败次数；达到上限作废验证码，防在线爆破注册验证码。
func (s *RegistrationService) burnEmailCodeOnVerifyFailure(ctx context.Context, scene, email string) error {
	failKey := store.EmailCodeVerifyFailKey(scene, email)
	n, err := s.redis.Incr(ctx, failKey).Result()
	if err == nil {
		if n == 1 {
			ttl := 10 * time.Minute
			if s.authCfg.EmailCodeTTLSeconds > 0 {
				ttl = time.Duration(s.authCfg.EmailCodeTTLSeconds) * time.Second
			}
			s.redis.Expire(ctx, failKey, ttl)
		}
		if n >= emailCodeMaxVerifyFailures {
			s.redis.Del(ctx, store.EmailCodeKey(scene, email), failKey)
			return constants.ErrBadRequestWithMsg("验证码错误次数过多，请重新获取")
		}
	}
	return constants.ErrCaptchaIncorrect
}

func (s *RegistrationService) clearEmailCode(ctx context.Context, scene, email string) {
	if s.redis == nil {
		return
	}
	s.redis.Del(ctx, store.EmailCodeKey(scene, email), store.EmailCodeVerifyFailKey(scene, email))
}
