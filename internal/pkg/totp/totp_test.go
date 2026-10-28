package totp

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// RFC 4226 Appendix D 的测试向量（secret = "12345678901234567890"）。
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestHOTPRFCVectors(t *testing.T) {
	want := []string{"755224", "287082", "359152", "969429", "338314", "254676", "287922", "162583", "399871", "520489"}
	for counter, expected := range want {
		got, err := hotp(rfcSecret, uint64(counter), Digits)
		if err != nil {
			t.Fatalf("counter %d: %v", counter, err)
		}
		if got != expected {
			t.Fatalf("counter %d: got %s want %s", counter, got, expected)
		}
	}
}

func TestGenerateSecretRoundTrip(t *testing.T) {
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	code, err := Code(secret, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatalf("code: %v", err)
	}
	if len(code) != Digits {
		t.Fatalf("unexpected code length: %q", code)
	}
	if !Validate(secret, code, time.Unix(1_700_000_000, 0)) {
		t.Fatal("own code should validate")
	}
}

func TestValidateSkewAndRejection(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	prev, err := Code(rfcSecret, now.Add(-Period))
	if err != nil {
		t.Fatal(err)
	}
	next, err := Code(rfcSecret, now.Add(Period))
	if err != nil {
		t.Fatal(err)
	}
	if !Validate(rfcSecret, prev, now) || !Validate(rfcSecret, next, now) {
		t.Fatal("±1 步内的验证码应通过")
	}
	tooOld, err := Code(rfcSecret, now.Add(-2*Period))
	if err != nil {
		t.Fatal(err)
	}
	if Validate(rfcSecret, tooOld, now) {
		t.Fatal("超过容差窗口的验证码应拒绝")
	}
	if Validate(rfcSecret, "", now) || Validate(rfcSecret, "12345", now) || Validate(rfcSecret, "abcdef", now) {
		t.Fatal("格式不合法的验证码应拒绝")
	}
	if Validate("not-base32!!", "123456", now) {
		t.Fatal("非法密钥应拒绝")
	}
	// 宽松格式：小写、空格、填充符都应归一化
	current, err := Code(rfcSecret, now)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(rfcSecret) + "===="
	if !Validate(lower, current, now) {
		t.Fatal("小写/填充密钥应可归一化")
	}
}

func TestKeyURI(t *testing.T) {
	uri := KeyURI("Yunshu 云枢", "alice@example.com", rfcSecret)
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Scheme != "otpauth" || parsed.Host != "totp" {
		t.Fatalf("unexpected uri: %s", uri)
	}
	q := parsed.Query()
	if q.Get("secret") != rfcSecret {
		t.Fatalf("secret mismatch: %s", q.Get("secret"))
	}
	if q.Get("digits") != "6" || q.Get("period") != "30" || q.Get("algorithm") != "SHA1" {
		t.Fatalf("unexpected params: %v", q)
	}
	if q.Get("issuer") != "Yunshu 云枢" {
		t.Fatalf("issuer mismatch: %s", q.Get("issuer"))
	}
}
