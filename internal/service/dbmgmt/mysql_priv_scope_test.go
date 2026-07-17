package dbmgmt

import (
	"testing"

	"yunshu/internal/model"
)

func TestSplitMySQLPrivilegesDatabaseLevel(t *testing.T) {
	privs := []string{
		"SELECT", "INSERT", "GRANT", "SUPER", "PROCESS", "CREATE",
	}
	split := splitMySQLPrivileges(privs, model.DbAppUserPrivDatabase)
	if len(split.Database) != 3 {
		t.Fatalf("database privs = %v, want 3", split.Database)
	}
	if len(split.Global) != 2 {
		t.Fatalf("global privs = %v, want SUPER PROCESS", split.Global)
	}
	if !split.WithGrantOption {
		t.Fatal("expected GRANT OPTION")
	}
}

func TestBuildGrantStmtsDatabaseLevel(t *testing.T) {
	req := &model.DbAppUserRequest{
		PrivLevel:    model.DbAppUserPrivDatabase,
		DatabaseName: "test",
		MySQLUser:    "u1",
	}
	stmts := buildGrantStmtsForHost(req, "10.0.0.1", []string{"SELECT", "SUPER", "GRANT"})
	if len(stmts) != 2 {
		t.Fatalf("got %d stmts: %v", len(stmts), stmts)
	}
	if !contains(stmts[0], "GRANT SELECT ON `test`.*") || contains(stmts[0], "WITH GRANT OPTION") {
		t.Fatalf("unexpected db stmt: %s", stmts[0])
	}
	if !contains(stmts[1], "GRANT SUPER ON *.*") || !contains(stmts[1], "WITH GRANT OPTION") {
		t.Fatalf("unexpected global stmt: %s", stmts[1])
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
