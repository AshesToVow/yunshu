package cicd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func signCallback(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyJenkinsCallbackSigned_WithTimestamp(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"run","run_status":"success"}`)
	now := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	sig := signCallback(secret, append(append([]byte(ts), '.'), body...))

	if err := VerifyJenkinsCallbackSigned(secret, body, sig, ts, now); err != nil {
		t.Fatalf("valid signed callback rejected: %v", err)
	}
}

func TestVerifyJenkinsCallbackSigned_RejectsStaleTimestamp(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"run"}`)
	captured := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(captured.Unix(), 10)
	sig := signCallback(secret, append(append([]byte(ts), '.'), body...))

	// 重放：签名依然合法，但时间戳已超出允许窗口。
	replayAt := captured.Add(callbackTimestampSkew + time.Minute)
	if err := VerifyJenkinsCallbackSigned(secret, body, sig, ts, replayAt); err == nil {
		t.Fatal("expected replayed callback to be rejected")
	}
}

func TestVerifyJenkinsCallbackSigned_RejectsBodyOnlySignatureWhenTimestampPresent(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"run"}`)
	now := time.Unix(1700000000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	// 只对 body 签名（未把时间戳纳入）：必须失败，否则攻击者可自带任意时间戳绕过窗口校验。
	if err := VerifyJenkinsCallbackSigned(secret, body, signCallback(secret, body), ts, now); err == nil {
		t.Fatal("expected body-only signature to be rejected when timestamp header present")
	}
}

func TestVerifyJenkinsCallbackSigned_LegacyWithoutTimestamp(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"stage"}`)
	now := time.Unix(1700000000, 0)
	if err := VerifyJenkinsCallbackSigned(secret, body, signCallback(secret, body), "", now); err != nil {
		t.Fatalf("legacy body-only callback rejected: %v", err)
	}
	if err := VerifyJenkinsCallbackSigned(secret, body, "sha256=deadbeef", "", now); err == nil {
		t.Fatal("expected bad legacy signature to be rejected")
	}
}

func TestVerifyJenkinsCallbackSigned_RejectsEmptySecret(t *testing.T) {
	if err := VerifyJenkinsCallbackSigned("  ", []byte("{}"), "sha256=abc", "", time.Now()); err == nil {
		t.Fatal("expected empty secret to be rejected")
	}
}

func TestVerifyJenkinsCallbackSigned_RejectsMalformedTimestamp(t *testing.T) {
	if err := VerifyJenkinsCallbackSigned("s", []byte("{}"), "sha256=abc", "not-a-number", time.Now()); err == nil {
		t.Fatal("expected malformed timestamp to be rejected")
	}
}

func TestBuildCallbackStatusAllowed(t *testing.T) {
	cases := []struct {
		current string
		next    string
		want    bool
	}{
		{"pending", "running", true},
		{"running", "success", true},
		{"running", "failure", true},
		{"running", "aborted", true},
		{"", "running", true},
		// 终态不可回退、不可改判；同值重复回调按幂等放行。
		{"success", "running", false},
		{"failure", "success", false},
		{"success", "success", true},
		{"aborted", "aborted", true},
		{"running", "", false},
		{"running", "bogus", false},
	}
	for _, c := range cases {
		if got := buildCallbackStatusAllowed(c.current, c.next); got != c.want {
			t.Fatalf("buildCallbackStatusAllowed(%q, %q) = %v, want %v", c.current, c.next, got, c.want)
		}
	}
}

func TestIsSensitiveParamKey(t *testing.T) {
	sensitive := []string{
		"SONAR_TOKEN", "YUNSHU_CALLBACK_HMAC_SECRET", "harborPassword",
		"GIT_PASSWORD", "someApiKey", "MY_PRIVATE_KEY", "registryCredential", "sshPassphrase",
	}
	for _, k := range sensitive {
		if !IsSensitiveParamKey(k) {
			t.Fatalf("IsSensitiveParamKey(%q) = false, want true", k)
		}
	}
	plain := []string{"branchName", "publishMode", "Tenv", "SONAR_HOST_URL", "YUNSHU_BUILD_RUN_ID", ""}
	for _, k := range plain {
		if IsSensitiveParamKey(k) {
			t.Fatalf("IsSensitiveParamKey(%q) = true, want false", k)
		}
	}
}

func TestParamsJSONMasksSensitiveValues(t *testing.T) {
	raw := map[string]string{
		"branchName":                  "main",
		"SONAR_TOKEN":                 "squ_realtoken",
		"YUNSHU_CALLBACK_HMAC_SECRET": "hmac-secret",
		"harborPassword":              "",
	}
	out := ParamsJSON(raw)
	if strings.Contains(out, "squ_realtoken") || strings.Contains(out, "hmac-secret") {
		t.Fatalf("ParamsJSON leaked credential: %s", out)
	}
	var decoded map[string]string
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("ParamsJSON produced invalid JSON: %v", err)
	}
	if decoded["branchName"] != "main" {
		t.Fatalf("non-sensitive param altered: %q", decoded["branchName"])
	}
	if decoded["SONAR_TOKEN"] != maskedParamValue || decoded["YUNSHU_CALLBACK_HMAC_SECRET"] != maskedParamValue {
		t.Fatalf("sensitive params not masked: %#v", decoded)
	}
	// 空值不必脱敏，保持原样便于排查「参数未注入」问题。
	if decoded["harborPassword"] != "" {
		t.Fatalf("empty sensitive param should stay empty, got %q", decoded["harborPassword"])
	}
	if ParamsJSON(nil) != "" {
		t.Fatal("ParamsJSON(nil) should be empty string")
	}
	// 原始 map 不能被改写：真实凭据仍要传给 Jenkins。
	if raw["SONAR_TOKEN"] != "squ_realtoken" {
		t.Fatalf("ParamsJSON mutated input map: %q", raw["SONAR_TOKEN"])
	}
}
