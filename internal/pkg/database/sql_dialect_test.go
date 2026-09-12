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

func TestSQLDeletePermissionDuplicatesDialects(t *testing.T) {
	if got := SQLDeletePermissionActiveDuplicates("mysql"); !containsAll(got, "DELETE d1 FROM", "permissions", "UPPER") {
		t.Fatalf("mysql active dedupe unexpected: %q", got)
	}
	if got := SQLDeletePermissionActiveDuplicates("postgres"); !containsAll(got, "USING", "permissions") {
		t.Fatalf("postgres active dedupe unexpected: %q", got)
	}
	if got := SQLDeletePermissionSoftDuplicatesWhenActive("mysql"); !containsAll(got, "DELETE p FROM", "deleted_at IS NOT NULL") {
		t.Fatalf("mysql soft-when-active unexpected: %q", got)
	}
	if got := SQLNormalizePermissionActionUpper("mysql"); !containsAll(got, "UPDATE permissions", "BINARY") {
		t.Fatalf("mysql normalize action unexpected: %q", got)
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

func TestSQLDropIndexIfExists_SanitizesIdent(t *testing.T) {
	// 标识符白名单：反引号/双引号/分号/空格等破坏引号闭合的字符必须被剔除
	got := SQLDropIndexIfExists("mysql", "roles`; DROP TABLE users; --", "idx`x")
	for _, bad := range []string{"DROP TABLE", ";", "--", " users"} {
		if strings.Contains(got, bad) {
			t.Fatalf("mysql drop index SQL not sanitized: %q (contains %q)", got, bad)
		}
	}
	if got != "ALTER TABLE `rolesDROPTABLEusers` DROP INDEX `idxx`" {
		t.Fatalf("unexpected sanitized mysql SQL: %q", got)
	}
	pg := SQLDropIndexIfExists("postgres", "roles", `idx"; DROP TABLE users; --`)
	if strings.Contains(pg, "DROP TABLE") || strings.Contains(pg, ";") {
		t.Fatalf("postgres drop index SQL not sanitized: %q", pg)
	}
}

func TestSQLCSVContainsID(t *testing.T) {
	if got := SQLCSVContainsID("mysql", "n.id", "e.matched_policy_ids"); !containsAll(got, "FIND_IN_SET(n.id, e.matched_policy_ids)") {
		t.Fatalf("mysql csv: %q", got)
	}
	if got := SQLCSVContainsID("postgres", "n.id", "e.matched_policy_ids"); !containsAll(got, "string_to_array", "ANY") {
		t.Fatalf("postgres csv: %q", got)
	}
	if got := SQLCSVContainsID("dm", "n.id", "e.matched_policy_ids"); !containsAll(got, "INSTR", "CAST(n.id AS VARCHAR)") {
		t.Fatalf("dameng csv: %q", got)
	}
}

func TestSQLStaleTimestampBefore(t *testing.T) {
	if got := SQLStaleTimestampBefore("mysql", "claimed_at", "5 MINUTE"); !containsAll(got, "DATE_SUB", "5 MINUTE") {
		t.Fatalf("mysql stale: %q", got)
	}
	if got := SQLStaleTimestampBefore("postgres", "claimed_at", "5 MINUTE"); !containsAll(got, "NOW() - INTERVAL", "5 minutes") {
		t.Fatalf("postgres stale: %q", got)
	}
	if got := SQLStaleTimestampBefore("dameng", "claimed_at", "5 MINUTE"); !containsAll(got, "DATEADD", "MINUTE", "SYSDATE") {
		t.Fatalf("dameng stale: %q", got)
	}
}

func TestSQLOnConflictChanged(t *testing.T) {
	if got := SQLOnConflictChanged("mysql", "type", "reason"); !containsAll(got, "VALUES(`type`)", "VALUES(`reason`)") {
		t.Fatalf("mysql onconflict: %q", got)
	}
	if got := SQLOnConflictChanged("postgres", "type", "reason"); !containsAll(got, "EXCLUDED.type", "IS DISTINCT FROM") {
		t.Fatalf("postgres onconflict: %q", got)
	}
	if got := SQLOnConflictChanged("dm", "type", "reason"); !containsAll(got, `"excluded"."type"`, `"excluded"."reason"`) {
		t.Fatalf("dameng onconflict: %q", got)
	}
}

func TestSQLBackfillAlertEventProject(t *testing.T) {
	if got := SQLBackfillAlertEventProjectFromDatasource("postgres"); !containsAll(got, "FROM alert_datasources", "SET project_id") {
		t.Fatalf("postgres datasource backfill: %q", got)
	}
	if got := SQLBackfillAlertEventProjectFromDatasource("mysql"); !containsAll(got, "INNER JOIN alert_datasources", "SET e.project_id") {
		t.Fatalf("mysql datasource backfill: %q", got)
	}
	if got := SQLBackfillAlertEventProjectFromDatasource("dm"); !containsAll(got, "SELECT d.project_id FROM alert_datasources", "EXISTS") {
		t.Fatalf("dameng datasource backfill: %q", got)
	}
	if got := SQLBackfillAlertEventProjectFromSubscriptions("postgres"); !containsAll(got, "string_to_array", "FROM alert_subscription_nodes") {
		t.Fatalf("postgres subscription backfill: %q", got)
	}
	if got := SQLBackfillAlertEventProjectFromSubscriptions("mysql"); !containsAll(got, "FIND_IN_SET", "INNER JOIN alert_subscription_nodes") {
		t.Fatalf("mysql subscription backfill: %q", got)
	}
	if got := SQLBackfillAlertEventProjectFromSubscriptions("dameng"); !containsAll(got, "INSTR", "alert_subscription_nodes") {
		t.Fatalf("dameng subscription backfill: %q", got)
	}
}

func TestSQLDeleteDictDuplicatesDameng(t *testing.T) {
	if got := SQLDeleteDictDuplicatesByLabel("dm"); !containsAll(got, "EXISTS", "dict_entries") {
		t.Fatalf("dameng label dedupe: %q", got)
	}
}

func TestNormalizeDriverDameng(t *testing.T) {
	for _, in := range []string{"dameng", "DM", "dm8"} {
		if got := NormalizeDriver(in); got != "dameng" {
			t.Fatalf("NormalizeDriver(%q)=%q, want dameng", in, got)
		}
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

