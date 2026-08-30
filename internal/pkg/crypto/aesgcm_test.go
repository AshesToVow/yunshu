package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestInspectKeyString(t *testing.T) {
	raw32 := strings.Repeat("a", 32)
	b64of32 := base64.StdEncoding.EncodeToString([]byte(raw32))

	cases := []struct {
		name    string
		key     string
		wantErr bool
		want    KeyInfo
	}{
		{name: "empty", key: "", wantErr: true},
		{name: "whitespace only", key: "   ", wantErr: true},
		{name: "base64 of 32 bytes", key: b64of32, want: KeyInfo{Base64: true}},
		{name: "base64 with surrounding spaces", key: "  " + b64of32 + "  ", want: KeyInfo{Base64: true}},
		{name: "raw 32 bytes", key: raw32, want: KeyInfo{Raw: true}},
		{name: "short string derives", key: "change-me", want: KeyInfo{Derived: true}},
		{name: "base64 of 16 bytes derives", key: base64.StdEncoding.EncodeToString([]byte(strings.Repeat("b", 16))), want: KeyInfo{Derived: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := InspectKeyString(tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("InspectKeyString(%q) err = nil, want error", tc.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("InspectKeyString(%q) unexpected err: %v", tc.key, err)
			}
			if got != tc.want {
				t.Fatalf("InspectKeyString(%q) = %+v, want %+v", tc.key, got, tc.want)
			}
		})
	}
}

func TestKeyInfoStrong(t *testing.T) {
	if !(KeyInfo{Base64: true}).Strong() {
		t.Fatal("base64 key should be strong")
	}
	if !(KeyInfo{Raw: true}).Strong() {
		t.Fatal("raw 32-byte key should be strong")
	}
	if (KeyInfo{Derived: true}).Strong() {
		t.Fatal("SHA-256 derived key must not be reported as strong")
	}
	if (KeyInfo{}).Strong() {
		t.Fatal("zero KeyInfo must not be reported as strong")
	}
}

// 派生分支必须保留：删除会让既有密文永久不可解。此用例锁定该行为。
func TestNewAESGCMFromKeyStringRoundTrip(t *testing.T) {
	keys := []string{
		base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
		strings.Repeat("k", 32),
		"change-me", // 派生路径
	}
	for _, key := range keys {
		aead, err := NewAESGCMFromKeyString(key)
		if err != nil {
			t.Fatalf("NewAESGCMFromKeyString(%q) err = %v", key, err)
		}
		ct, err := EncryptString(aead, "AKID-secret")
		if err != nil {
			t.Fatalf("EncryptString err = %v", err)
		}
		pt, err := DecryptString(aead, ct)
		if err != nil {
			t.Fatalf("DecryptString err = %v", err)
		}
		if pt != "AKID-secret" {
			t.Fatalf("round trip = %q, want %q", pt, "AKID-secret")
		}
	}
}

func TestNewAESGCMFromKeyStringRejectsEmpty(t *testing.T) {
	if _, err := NewAESGCMFromKeyString("  "); err == nil {
		t.Fatal("NewAESGCMFromKeyString(\"\") err = nil, want error")
	}
}

func TestDecryptStringRejectsShortCiphertext(t *testing.T) {
	aead, err := NewAESGCMFromKeyString(strings.Repeat("k", 32))
	if err != nil {
		t.Fatalf("setup err = %v", err)
	}
	if _, err := DecryptString(aead, base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("DecryptString on short ciphertext err = nil, want error")
	}
}