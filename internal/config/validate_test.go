package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func strongKey() string {
	return base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
}

func strongJWT() string {
	return strings.Repeat("j", 48)
}

func newValidConfig(env string) *Config {
	c := &Config{}
	c.App.Env = env
	c.Security.EncryptionKey = strongKey()
	c.Auth.JWTSecret = strongJWT()
	return c
}

func TestIsProd(t *testing.T) {
	cases := map[string]bool{
		"prod":       true,
		"production": true,
		" PROD ":     true,
		"Production": true,
		"dev":        false,
		"test":       false,
		"":           false,
	}
	for env, want := range cases {
		c := &Config{}
		c.App.Env = env
		if got := c.IsProd(); got != want {
			t.Errorf("IsProd(env=%q) = %v, want %v", env, got, want)
		}
	}
	var nilCfg *Config
	if nilCfg.IsProd() {
		t.Error("nil config IsProd() = true, want false")
	}
}

func TestValidateHappyPath(t *testing.T) {
	for _, env := range []string{"dev", "prod"} {
		warnings, err := newValidConfig(env).Validate()
		if err != nil {
			t.Fatalf("env=%s Validate() err = %v, want nil", env, err)
		}
		if len(warnings) != 0 {
			t.Fatalf("env=%s Validate() warnings = %v, want none", env, warnings)
		}
	}
}

func TestValidateEmptySecretsAlwaysFail(t *testing.T) {
	for _, env := range []string{"dev", "prod"} {
		c := newValidConfig(env)
		c.Security.EncryptionKey = "   "
		if _, err := c.Validate(); err == nil {
			t.Fatalf("env=%s empty encryption_key: err = nil, want error", env)
		}

		c = newValidConfig(env)
		c.Auth.JWTSecret = ""
		if _, err := c.Validate(); err == nil {
			t.Fatalf("env=%s empty jwt_secret: err = nil, want error", env)
		}
	}
}

func TestValidateProdRejectsWeakSecrets(t *testing.T) {
	t.Run("well-known encryption_key", func(t *testing.T) {
		c := newValidConfig("prod")
		c.Security.EncryptionKey = "change-me-32bytes-base64"
		if _, err := c.Validate(); err == nil {
			t.Fatal("err = nil, want error")
		}
	})
	t.Run("derived encryption_key", func(t *testing.T) {
		c := newValidConfig("prod")
		c.Security.EncryptionKey = "some-random-but-short-key"
		if _, err := c.Validate(); err == nil {
			t.Fatal("err = nil, want error")
		}
	})
	t.Run("well-known jwt_secret", func(t *testing.T) {
		c := newValidConfig("prod")
		c.Auth.JWTSecret = "YunshuJWTSecretKey2026MustBeAtLeast32BytesLong!!"
		if _, err := c.Validate(); err == nil {
			t.Fatal("err = nil, want error")
		}
	})
	t.Run("short jwt_secret", func(t *testing.T) {
		c := newValidConfig("prod")
		c.Auth.JWTSecret = strings.Repeat("j", 31)
		if _, err := c.Validate(); err == nil {
			t.Fatal("err = nil, want error")
		}
	})
}

func TestValidateDevOnlyWarns(t *testing.T) {
	c := newValidConfig("dev")
	c.Security.EncryptionKey = "change-me"     // 既是占位值也走派生路径
	c.Auth.JWTSecret = strings.Repeat("j", 10) // 过短
	warnings, err := c.Validate()
	if err != nil {
		t.Fatalf("dev Validate() err = %v, want nil (warn only)", err)
	}
	// 占位值 + 派生 + 短 jwt = 3 条告警
	if len(warnings) != 3 {
		t.Fatalf("dev Validate() warnings = %v (%d), want 3", warnings, len(warnings))
	}
}

func TestValidateNilConfig(t *testing.T) {
	var c *Config
	if _, err := c.Validate(); err == nil {
		t.Fatal("nil config Validate() err = nil, want error")
	}
}

func TestIsWellKnownSecret(t *testing.T) {
	if !IsWellKnownSecret("  change-me  ") {
		t.Error("trimmed well-known secret should be detected")
	}
	if IsWellKnownSecret(strongJWT()) {
		t.Error("strong secret must not be flagged as well-known")
	}
}

func TestAutoMigrateEnabled(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	cases := []struct {
		name     string
		env      string
		database *bool
		mysql    *bool
		want     bool
	}{
		{name: "unset dev defaults on", env: "dev", want: true},
		{name: "unset prod defaults off", env: "prod", want: false},
		{name: "unset production defaults off", env: "production", want: false},
		{name: "database explicit true in prod", env: "prod", database: boolPtr(true), want: true},
		{name: "database explicit false in dev", env: "dev", database: boolPtr(false), want: false},
		{name: "mysql fallback true in prod", env: "prod", mysql: boolPtr(true), want: true},
		{name: "mysql fallback false in dev", env: "dev", mysql: boolPtr(false), want: false},
		{name: "database wins over mysql", env: "dev", database: boolPtr(false), mysql: boolPtr(true), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{}
			c.App.Env = tc.env
			c.Database.AutoMigrate = tc.database
			c.MySQL.AutoMigrate = tc.mysql
			if got := c.AutoMigrateEnabled(); got != tc.want {
				t.Fatalf("AutoMigrateEnabled() = %v, want %v", got, tc.want)
			}
		})
	}

	var nilCfg *Config
	if nilCfg.AutoMigrateEnabled() {
		t.Error("nil config AutoMigrateEnabled() = true, want false")
	}
}
