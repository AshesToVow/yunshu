package system

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/password"
	"yunshu/internal/pkg/totp"
	"yunshu/internal/store"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	mfaChallengeTTL      = 5 * time.Minute
	mfaMaxVerifyAttempts = 5
	mfaRecoveryCodeCount = 8
	// 恢复码字符集去掉易混淆的 I/O/0/1。
	recoveryAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// beginMFA 在第一因子（密码/邮箱验证码）通过后签发挑战，替代直接下发会话。
func (s *AuthService) beginMFA(ctx context.Context, user *model.User, source string) (*LoginResponse, error) {
	if s.redis == nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "beginMFA", constants.ErrMsgaf4823214b6e)
	}
	token := uuid.NewString()
	if err := store.SaveMFAChallenge(ctx, s.redis, token, user.ID, source, mfaChallengeTTL); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "beginMFA", err)
	}
	return &LoginResponse{MFARequired: true, MFAChallenge: token}, nil
}

// VerifyMFALogin 校验第二因子并签发会话。
func (s *AuthService) VerifyMFALogin(ctx context.Context, req MFAVerifyRequest) (*LoginResponse, error) {
	if s.redis == nil {
		return nil, bizerrors.InternalMsg(ctx, "auth", "VerifyMFALogin", constants.ErrMsgaf4823214b6e)
	}
	challenge := strings.TrimSpace(req.Challenge)
	userID, source, err := store.GetMFAChallenge(ctx, s.redis, challenge)
	if err != nil {
		if errors.Is(err, store.ErrMFAChallengeInvalid) {
			return nil, constants.ErrBadRequestWithMsg("二次验证已过期，请重新登录")
		}
		return nil, bizerrors.Pass(ctx, "auth", "VerifyMFALogin", err)
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "VerifyMFALogin", err)
	}
	if user.Status != model.StatusEnabled {
		return nil, constants.ErrAccountDisabled
	}
	if !user.MFAEnabled || strings.TrimSpace(user.MFASecret) == "" {
		_ = store.ConsumeMFAChallenge(ctx, s.redis, challenge)
		return nil, constants.ErrBadRequestWithMsg("该账号未启用二次验证，请重新登录")
	}

	ok, err := s.verifySecondFactor(ctx, user, req.Code)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "VerifyMFALogin", err)
	}
	if !ok {
		exceeded, incrErr := store.IncrMFAChallengeAttempts(ctx, s.redis, challenge, mfaMaxVerifyAttempts)
		if incrErr != nil {
			return nil, bizerrors.Pass(ctx, "auth", "VerifyMFALogin", incrErr)
		}
		if exceeded {
			return nil, constants.ErrBadRequestWithMsg("验证码错误次数过多，请重新登录")
		}
		return nil, constants.ErrBadRequestWithMsg("验证码不正确")
	}

	if err := store.ConsumeMFAChallenge(ctx, s.redis, challenge); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "VerifyMFALogin", err)
	}
	s.clearLoginFailures(ctx, user.ID)
	resp, err := s.issueLoginResponse(ctx, user)
	if err != nil {
		return nil, err
	}
	resp.LoginSource = source
	return resp, nil
}

// SetupMFA 生成待确认的 TOTP 密钥；确认前不改变启用状态。
func (s *AuthService) SetupMFA(ctx context.Context, userID uint, req MFASetupRequest) (*MFASetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "SetupMFA", err)
	}
	if password.Compare(user.Password, req.Password) != nil {
		return nil, constants.ErrBadRequestWithMsg("当前密码不正确")
	}
	if user.MFAEnabled {
		return nil, constants.ErrBadRequestWithMsg("已启用二次验证，如需更换请先关闭")
	}
	// 密钥需加密落库；未配置 EncryptionKey 时直接拒绝，避免生成无法保存的密钥。
	if strings.TrimSpace(s.security.EncryptionKey) == "" {
		return nil, constants.ErrBadRequestWithMsg("系统未配置加密密钥，暂不支持开启二次验证")
	}

	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SetupMFA", err)
	}
	encrypted, err := s.encryptMFASecret(secret)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SetupMFA", err)
	}
	user.MFASecret = encrypted
	user.MFARecoveryCodes = ""
	if err = s.userRepo.Save(ctx, user); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "SetupMFA", err)
	}
	return &MFASetupResponse{
		Secret:     secret,
		OtpauthURL: totp.KeyURI(s.appName, user.Username, secret),
	}, nil
}

// EnableMFA 校验一次 TOTP 后正式启用，并一次性返回恢复码明文。
func (s *AuthService) EnableMFA(ctx context.Context, userID uint, req MFAEnableRequest) (*MFAEnableResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrUserNotFound
		}
		return nil, bizerrors.Pass(ctx, "auth", "EnableMFA", err)
	}
	if user.MFAEnabled {
		return nil, constants.ErrBadRequestWithMsg("二次验证已启用")
	}
	secret, err := s.decryptMFASecret(user)
	if err != nil || secret == "" {
		return nil, constants.ErrBadRequestWithMsg("请先获取二次验证密钥")
	}
	if !totp.Validate(secret, req.Code, s.currentTime()) {
		return nil, constants.ErrBadRequestWithMsg("验证码不正确")
	}

	codes, err := generateRecoveryCodes(mfaRecoveryCodeCount)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "EnableMFA", err)
	}
	hashes, err := hashRecoveryCodes(codes)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "EnableMFA", err)
	}
	now := s.currentTime()
	user.MFARecoveryCodes = hashes
	user.MFAEnabled = true
	user.MFAEnabledAt = &now
	if err = s.userRepo.Save(ctx, user); err != nil {
		return nil, bizerrors.Pass(ctx, "auth", "EnableMFA", err)
	}
	return &MFAEnableResponse{RecoveryCodes: codes}, nil
}

// DisableMFA 需要同时提供密码与当前验证码（或恢复码），防止会话被盗后关闭二次验证。
func (s *AuthService) DisableMFA(ctx context.Context, userID uint, req MFADisableRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrUserNotFound
		}
		return bizerrors.Pass(ctx, "auth", "DisableMFA", err)
	}
	if password.Compare(user.Password, req.Password) != nil {
		return constants.ErrBadRequestWithMsg("当前密码不正确")
	}
	if !user.MFAEnabled {
		return constants.ErrBadRequestWithMsg("该账号未启用二次验证")
	}
	ok, err := s.verifySecondFactor(ctx, user, req.Code)
	if err != nil {
		return bizerrors.Pass(ctx, "auth", "DisableMFA", err)
	}
	if !ok {
		return constants.ErrBadRequestWithMsg("验证码不正确")
	}
	user.MFAEnabled = false
	user.MFASecret = ""
	user.MFARecoveryCodes = ""
	user.MFAEnabledAt = nil
	if err = s.userRepo.Save(ctx, user); err != nil {
		return bizerrors.Pass(ctx, "auth", "DisableMFA", err)
	}
	return nil
}

// verifySecondFactor 先按 TOTP 校验，再用恢复码兜底；恢复码成功使用后立即作废。
func (s *AuthService) verifySecondFactor(ctx context.Context, user *model.User, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return false, nil
	}
	secret, err := s.decryptMFASecret(user)
	if err != nil {
		return false, err
	}
	if len(code) == totp.Digits && totp.Validate(secret, code, s.currentTime()) {
		return true, nil
	}
	normalized := normalizeRecoveryCode(code)
	if len(normalized) != 8 {
		return false, nil
	}
	hashes := parseRecoveryCodes(user.MFARecoveryCodes)
	for i, hash := range hashes {
		if password.Compare(hash, normalized) != nil {
			continue
		}
		hashes = append(hashes[:i], hashes[i+1:]...)
		encoded, err := json.Marshal(hashes)
		if err != nil {
			return false, err
		}
		user.MFARecoveryCodes = string(encoded)
		if err := s.userRepo.Save(ctx, user); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (s *AuthService) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *AuthService) mfaCipher() (cipher.AEAD, error) {
	return crypto.NewAESGCMFromKeyString(s.security.EncryptionKey)
}

func (s *AuthService) encryptMFASecret(secret string) (string, error) {
	aead, err := s.mfaCipher()
	if err != nil {
		return "", err
	}
	return crypto.EncryptString(aead, secret)
}

func (s *AuthService) decryptMFASecret(user *model.User) (string, error) {
	raw := strings.TrimSpace(user.MFASecret)
	if raw == "" {
		return "", nil
	}
	aead, err := s.mfaCipher()
	if err != nil {
		return "", err
	}
	return crypto.DecryptString(aead, raw)
}

func generateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, 0, count)
	buf := make([]byte, 8)
	for i := 0; i < count; i++ {
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		var sb strings.Builder
		for j := range buf {
			sb.WriteByte(recoveryAlphabet[int(buf[j])%len(recoveryAlphabet)])
		}
		raw := sb.String()
		codes = append(codes, raw[:4]+"-"+raw[4:])
	}
	return codes, nil
}

func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	return strings.ReplaceAll(code, " ", "")
}

func hashRecoveryCodes(codes []string) (string, error) {
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hash, err := password.Hash(normalizeRecoveryCode(code))
		if err != nil {
			return "", err
		}
		hashes = append(hashes, hash)
	}
	encoded, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func parseRecoveryCodes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var hashes []string
	if err := json.Unmarshal([]byte(raw), &hashes); err != nil {
		return nil
	}
	return hashes
}
