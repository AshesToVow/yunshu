package dictcategory

import "testing"

func TestApplyFilterCicd(t *testing.T) {
	clause, args, ok := ApplyFilter("cicd")
	if !ok {
		t.Fatal("expected ok")
	}
	if clause == "" || len(args) != 1 {
		t.Fatalf("unexpected clause=%q args=%v", clause, args)
	}
	if args[0] != "cicd_%" {
		t.Fatalf("args=%v", args)
	}
}

func TestResolveDbmgmt(t *testing.T) {
	if got := Resolve("dbmgmt_goinception_enabled"); got != "dbmgmt" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCaseInsensitive(t *testing.T) {
	if got := Resolve("CICD_ENABLED"); got != "cicd" {
		t.Fatalf("got %q", got)
	}
	if got := Resolve("mail_host"); got != "system" {
		t.Fatalf("got %q", got)
	}
	if got := Resolve("password_expiry_days"); got != "system" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAuthAndCmdb(t *testing.T) {
	if got := Resolve("auth_jwt_secret"); got != "system" {
		t.Fatalf("auth got %q", got)
	}
	if got := Resolve("cmdb_max_transfer_file_mb"); got != "cmdb" {
		t.Fatalf("cmdb got %q", got)
	}
	if got := Resolve("esmgmt_backup_scheduler_enabled"); got != "backup" {
		t.Fatalf("esmgmt got %q", got)
	}
}
