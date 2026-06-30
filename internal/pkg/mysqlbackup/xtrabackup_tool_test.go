package mysqlbackup

import "testing"

func TestNormalizeXtrabackupTool(t *testing.T) {
	t.Parallel()
	got, err := NormalizeXtrabackupTool("")
	if err != nil || got != XtrabackupToolAuto {
		t.Fatalf("empty: %q %v", got, err)
	}
	got, err = NormalizeXtrabackupTool("innobackupex")
	if err != nil || got != XtrabackupToolInnobackupex {
		t.Fatalf("innobackupex: %q %v", got, err)
	}
	if _, err := NormalizeXtrabackupTool("bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDetectPhysicalBackupExecTool(t *testing.T) {
	t.Parallel()
	log := "[2026-06-30] physical backup start tool=innobackupex bin=/usr/bin/innobackupex"
	if got := DetectPhysicalBackupExecTool(log); got != XtrabackupToolInnobackupex {
		t.Fatalf("got %q", got)
	}
	if got := DetectPhysicalBackupExecTool("tool=xtrabackup archive=/a/b.tar.gz"); got != XtrabackupToolXtrabackup {
		t.Fatalf("got %q", got)
	}
}
