package dbmgmt

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/pkg/auth"
)

func TestExtractWriteTableRefs(t *testing.T) {
	t.Parallel()
	refs := extractWriteTableRefs("UPDATE users SET a=1 WHERE id=1", "appdb")
	if len(refs) != 1 || refs[0].Table != "users" || refs[0].Schema != "appdb" {
		t.Fatalf("UPDATE refs = %+v", refs)
	}
	refs = extractWriteTableRefs("INSERT INTO other.orders (id) VALUES (1)", "appdb")
	if len(refs) != 1 || refs[0].Schema != "other" || refs[0].Table != "orders" {
		t.Fatalf("INSERT refs = %+v", refs)
	}
	refs = extractWriteTableRefs("CREATE DATABASE foo", "appdb")
	if len(refs) != 0 {
		t.Fatalf("CREATE DATABASE must not yield table refs, got %+v", refs)
	}
	refs = extractWriteTableRefs("ALTER TABLE `appdb`.`t1` ADD COLUMN x INT", "")
	if len(refs) != 1 || refs[0].Schema != "appdb" || refs[0].Table != "t1" {
		t.Fatalf("ALTER refs = %+v", refs)
	}
	refs = extractWriteTableRefs("UPDATE /*c*/ users SET a=1 WHERE id=1", "appdb")
	if len(refs) != 1 || refs[0].Table != "users" {
		t.Fatalf("commented UPDATE refs = %+v", refs)
	}
}

func TestGrantCoversWrite_TableScoped(t *testing.T) {
	t.Parallel()
	g := modelGrant("appdb", []string{"users"}, false)
	g.CanDML = true
	if grantCoversWrite(g, "appdb", "orders", false) {
		t.Fatal("table-scoped DML must not cover other tables")
	}
	if !grantCoversWrite(g, "appdb", "users", false) {
		t.Fatal("expected cover users")
	}
	if grantCoversWrite(g, "appdb", "", false) {
		t.Fatal("unknown table must require whole-db write grant")
	}
	wide := modelGrant("appdb", nil, false)
	wide.CanDML = true
	if !grantCoversWrite(wide, "appdb", "", false) {
		t.Fatal("db-level DML should cover unknown table")
	}
	instWide := modelGrant("", nil, false)
	instWide.CanDML = true
	if grantCoversWrite(instWide, "appdb", "users", false) {
		t.Fatal("empty database_name write grant must not cover writes")
	}
}

func TestForbidSelfApprove(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultDbmgmtConfig()
	cfg.ForbidSelfApprove = true
	s := &Service{cfg: cfg}
	actor := &auth.CurrentUser{ID: 7, RoleCodes: []string{"developer"}}
	if err := s.forbidSelfApprove(nil, actor, 7); err == nil {
		t.Fatal("same user must be blocked")
	}
	if err := s.forbidSelfApprove(nil, actor, 8); err != nil {
		t.Fatalf("other submitter should pass: %v", err)
	}
	admin := &auth.CurrentUser{ID: 7, RoleCodes: []string{"super-admin"}}
	if err := s.forbidSelfApprove(nil, admin, 7); err != nil {
		t.Fatalf("super admin should bypass: %v", err)
	}
}

func TestAssessSQL_BlockedDangerousOps(t *testing.T) {
	t.Parallel()
	cases := []string{
		"SELECT LOAD_FILE('/etc/passwd')",
		"SET GLOBAL max_connections = 1000",
		"FLUSH PRIVILEGES",
	}
	for _, sql := range cases {
		a := AssessSQL(sql, false)
		if !a.Blocked {
			t.Fatalf("%q must be blocked, got %+v", sql, a)
		}
	}
}
