package system

import (
	"context"
	"strings"
	"testing"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/password"
	"yunshu/internal/pkg/totp"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

const mfaTestKey = "0123456789abcdef0123456789abcdef"

type fakeAuthUserRepo struct {
	user *model.User
}

func (f *fakeAuthUserRepo) GetByID(context.Context, uint) (*model.User, error) {
	return f.user, nil
}

func (f *fakeAuthUserRepo) GetByUsername(context.Context, string) (*model.User, error) {
	return f.user, nil
}

func (f *fakeAuthUserRepo) GetByUsernameForAuth(context.Context, string) (*model.User, error) {
	return f.user, nil
}

func (f *fakeAuthUserRepo) GetByEmail(context.Context, string) (*model.User, error) {
	return f.user, nil
}

func (f *fakeAuthUserRepo) GetByEmailForAuth(context.Context, string) (*model.User, error) {
	return f.user, nil
}

func (f *fakeAuthUserRepo) Create(context.Context, *model.User) error { return nil }

func (f *fakeAuthUserRepo) Save(context.Context, *model.User) error { return nil }

func newMFATestService(t *testing.T) (*AuthService, *fakeAuthUserRepo, time.Time) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	hash, err := password.Hash("Passw0rd!x")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	now := time.Unix(1_700_000_000, 0)
	repo := &fakeAuthUserRepo{user: &model.User{
		ID:       7,
		Username: "alice",
		Password: hash,
		Status:   model.StatusEnabled,
	}}
	svc := NewAuthService(repo, rdb, func(context.Context) dictconfig.PasswordPolicyConfig {
		return dictconfig.PasswordPolicyConfig{}
	}, config.AuthConfig{AccessTokenTTLMinutes: 30, RefreshTokenTTLHours: 24},
		config.SecurityConfig{EncryptionKey: mfaTestKey}, nil, "Yunshu")
	svc.now = func() time.Time { return now }
	return svc, repo, now
}

func TestMFASetupEnableDisableFlow(t *testing.T) {
	svc, repo, now := newMFATestService(t)
	ctx := context.Background()

	if _, err := svc.SetupMFA(ctx, repo.user.ID, MFASetupRequest{Password: "wrong"}); err == nil || !strings.Contains(err.Error(), "密码不正确") {
		t.Fatalf("wrong password err = %v", err)
	}

	setup, err := svc.SetupMFA(ctx, repo.user.ID, MFASetupRequest{Password: "Passw0rd!x"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if setup.Secret == "" || !strings.Contains(setup.OtpauthURL, "otpauth://totp/") {
		t.Fatalf("unexpected setup response: %+v", setup)
	}
	if repo.user.MFASecret == setup.Secret {
		t.Fatal("TOTP secret must be stored encrypted")
	}
	if repo.user.MFAEnabled {
		t.Fatal("setup alone must not enable MFA")
	}

	if _, err = svc.EnableMFA(ctx, repo.user.ID, MFAEnableRequest{Code: "000000"}); err == nil || !strings.Contains(err.Error(), "验证码不正确") {
		t.Fatalf("wrong code err = %v", err)
	}

	code, err := totp.Code(setup.Secret, now)
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	enabled, err := svc.EnableMFA(ctx, repo.user.ID, MFAEnableRequest{Code: code})
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if len(enabled.RecoveryCodes) != mfaRecoveryCodeCount {
		t.Fatalf("recovery codes = %d, want %d", len(enabled.RecoveryCodes), mfaRecoveryCodeCount)
	}
	if !repo.user.MFAEnabled || repo.user.MFAEnabledAt == nil {
		t.Fatal("user should be MFA enabled")
	}
	if strings.Contains(repo.user.MFARecoveryCodes, enabled.RecoveryCodes[0]) {
		t.Fatal("recovery codes must be stored hashed")
	}

	// 恢复码可用一次，用后作废
	ok, err := svc.verifySecondFactor(ctx, repo.user, enabled.RecoveryCodes[0])
	if err != nil || !ok {
		t.Fatalf("recovery code verify = %v, %v", ok, err)
	}
	ok, err = svc.verifySecondFactor(ctx, repo.user, enabled.RecoveryCodes[0])
	if err != nil || ok {
		t.Fatalf("reused recovery code = %v, %v, want false", ok, err)
	}

	if err = svc.DisableMFA(ctx, repo.user.ID, MFADisableRequest{Password: "Passw0rd!x", Code: "000000"}); err == nil || !strings.Contains(err.Error(), "验证码不正确") {
		t.Fatalf("disable wrong code err = %v", err)
	}
	freshCode, err := totp.Code(setup.Secret, now)
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if err = svc.DisableMFA(ctx, repo.user.ID, MFADisableRequest{Password: "Passw0rd!x", Code: freshCode}); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if repo.user.MFAEnabled || repo.user.MFASecret != "" || repo.user.MFARecoveryCodes != "" || repo.user.MFAEnabledAt != nil {
		t.Fatalf("MFA fields not cleared: %+v", repo.user)
	}
}

func TestSetupMFARequiresEncryptionKey(t *testing.T) {
	t.Parallel()
	svc, repo, _ := newMFATestService(t)
	svc.security.EncryptionKey = " "
	if _, err := svc.SetupMFA(context.Background(), repo.user.ID, MFASetupRequest{Password: "Passw0rd!x"}); err == nil || !strings.Contains(err.Error(), "加密密钥") {
		t.Fatalf("missing encryption key err = %v, want explicit message", err)
	}
}

func TestVerifyMFALoginFlow(t *testing.T) {
	svc, repo, now := newMFATestService(t)
	ctx := context.Background()

	setup, err := svc.SetupMFA(ctx, repo.user.ID, MFASetupRequest{Password: "Passw0rd!x"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	code, err := totp.Code(setup.Secret, now)
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if _, err = svc.EnableMFA(ctx, repo.user.ID, MFAEnableRequest{Code: code}); err != nil {
		t.Fatalf("enable: %v", err)
	}

	challenge, err := svc.beginMFA(ctx, repo.user, model.LoginSourcePassword)
	if err != nil {
		t.Fatalf("beginMFA: %v", err)
	}
	if !challenge.MFARequired || challenge.MFAChallenge == "" {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
	if challenge.AccessToken != "" {
		t.Fatal("first factor must not issue a session when MFA is required")
	}

	if _, err = svc.VerifyMFALogin(ctx, MFAVerifyRequest{Challenge: challenge.MFAChallenge, Code: "123"}); err == nil || !strings.Contains(err.Error(), "验证码不正确") {
		t.Fatalf("bad code err = %v", err)
	}

	resp, err := svc.VerifyMFALogin(ctx, MFAVerifyRequest{Challenge: challenge.MFAChallenge, Code: code})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if resp.AccessToken == "" || resp.User.ID != repo.user.ID {
		t.Fatalf("session not issued: %+v", resp)
	}
	if resp.LoginSource != model.LoginSourcePassword {
		t.Fatalf("login source = %q, want password", resp.LoginSource)
	}

	// 挑战一次性：成功后不可复用
	if _, err = svc.VerifyMFALogin(ctx, MFAVerifyRequest{Challenge: challenge.MFAChallenge, Code: code}); err == nil {
		t.Fatal("challenge should be consumed after success")
	}
}

func TestVerifyMFAChallengeLockout(t *testing.T) {
	svc, repo, now := newMFATestService(t)
	ctx := context.Background()

	setup, err := svc.SetupMFA(ctx, repo.user.ID, MFASetupRequest{Password: "Passw0rd!x"})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	code, err := totp.Code(setup.Secret, now)
	if err != nil {
		t.Fatalf("totp code: %v", err)
	}
	if _, err = svc.EnableMFA(ctx, repo.user.ID, MFAEnableRequest{Code: code}); err != nil {
		t.Fatalf("enable: %v", err)
	}

	challenge, err := svc.beginMFA(ctx, repo.user, model.LoginSourcePassword)
	if err != nil {
		t.Fatalf("beginMFA: %v", err)
	}
	var lastErr error
	for i := 0; i < mfaMaxVerifyAttempts; i++ {
		_, lastErr = svc.VerifyMFALogin(ctx, MFAVerifyRequest{Challenge: challenge.MFAChallenge, Code: "000000"})
	}
	if lastErr == nil || !strings.Contains(lastErr.Error(), "次数过多") {
		t.Fatalf("lockout err = %v", lastErr)
	}
	// 超限后即便验证码正确也必须重新登录
	if _, err = svc.VerifyMFALogin(ctx, MFAVerifyRequest{Challenge: challenge.MFAChallenge, Code: code}); err == nil {
		t.Fatal("challenge should be invalidated after too many failures")
	}
}
