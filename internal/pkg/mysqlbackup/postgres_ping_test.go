package mysqlbackup

import (
	"context"
	"testing"
)

func TestFormatPostgresConnectLog(t *testing.T) {
	got := FormatPostgresConnectLog("", 0, "yunshu", "", "")
	if got == "" {
		t.Fatal("expected non-empty log label")
	}
}

func TestFormatPgDumpConnectArgs(t *testing.T) {
	got := FormatPgDumpConnectArgs("127.0.0.1", 5432, "yunshu", "app", nil)
	if got == "" || !contains(got, "--dbname=app") {
		t.Fatalf("unexpected pg_dump args: %q", got)
	}
}

func TestPingPostgresRequiresUser(t *testing.T) {
	err := PingPostgres(context.Background(), "127.0.0.1", 5432, "", "secret", "postgres", "disable")
	if err == nil {
		t.Fatal("expected error for empty user")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndex(s, sub) >= 0)
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
