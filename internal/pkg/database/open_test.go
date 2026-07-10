package database

import (
	"testing"

	"yunshu/internal/config"
)

func TestNormalizeDriver(t *testing.T) {
	cases := map[string]string{
		"":            "mysql",
		"mysql":       "mysql",
		"MYSQL":       "mysql",
		"postgres":    "postgres",
		"postgresql":  "postgres",
		"pg":          "postgres",
		"unsupported": "unsupported",
	}
	for in, want := range cases {
		if got := NormalizeDriver(in); got != want {
			t.Fatalf("NormalizeDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildPostgresDSN(t *testing.T) {
	dsn := buildPostgresDSN(config.DatabaseConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		User:     "yunshu",
		Password: "secret",
		DBName:   "yunshu",
		SSLMode:  "disable",
		TimeZone: "Asia/Shanghai",
	})
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
	if want := "postgres://"; dsn[:len(want)] != want {
		t.Fatalf("expected postgres URL scheme, got %q", dsn)
	}
}
