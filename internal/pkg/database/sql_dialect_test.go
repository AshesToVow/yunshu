package database

import (
	"strings"
	"testing"
)

func TestSQLDeleteDictDuplicatesDialects(t *testing.T) {
	if got := SQLDeleteDictDuplicatesByLabel("postgres"); got == "" || !containsAll(got, "USING", "dict_entries") {
		t.Fatalf("postgres label SQL unexpected: %q", got)
	}
	if got := SQLDeleteDictDuplicatesByLabel("mysql"); got == "" || !containsAll(got, "DELETE d1 FROM", "JOIN") {
		t.Fatalf("mysql label SQL unexpected: %q", got)
	}
	if got := SQLDeleteDictDuplicatesByValue("postgres"); got == "" || !containsAll(got, "TRIM(d1.value)", "USING") {
		t.Fatalf("postgres value SQL unexpected: %q", got)
	}
}

func TestSQLCreateAgentDiscoveryUniqueIndex(t *testing.T) {
	pg, err := SQLCreateAgentDiscoveryUniqueIndex("postgres")
	if err != nil || !containsAll(pg, "left(value, 512)", "IF NOT EXISTS") {
		t.Fatalf("postgres index SQL: %q err=%v", pg, err)
	}
	my, err := SQLCreateAgentDiscoveryUniqueIndex("mysql")
	if err != nil || !containsAll(my, "value(512)") {
		t.Fatalf("mysql index SQL: %q err=%v", my, err)
	}
}

func TestSQLDropIndexIfExists(t *testing.T) {
	if got := SQLDropIndexIfExists("postgres", "roles", "idx_roles_name"); !containsAll(got, "DROP INDEX IF EXISTS", "idx_roles_name") {
		t.Fatalf("postgres drop index SQL: %q", got)
	}
	if got := SQLDropIndexIfExists("mysql", "roles", "idx_roles_name"); !containsAll(got, "ALTER TABLE", "`roles`", "DROP INDEX", "`idx_roles_name`") {
		t.Fatalf("mysql drop index SQL: %q", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
