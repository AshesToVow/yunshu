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
}
