package mysqlbackup

import (
	"strings"
	"testing"
)

func TestMysqldumpScriptResolvesBin(t *testing.T) {
	script := BuildMysqldumpRemoteScript(MysqldumpRemoteScriptParams{
		WorkDir: "/export/backup", Basename: "test_3306",
		MySQLPass: "'pw'", DumpFlags: "--single-transaction", DumpTarget: "mydb",
		DumpTargetLabel: "mydb", MysqldumpBin: "/export/servers/app/mysql-5.7.22/bin/mysqldump",
		ConnectArgs: "-S '/export/servers/data/my3306/run/mysqld.sock' -u'root'",
		ConnectLog:  "socket=/export/servers/data/my3306/run/mysqld.sock user=root",
		ShellQuote:  func(s string) string { return "'" + s + "'" },
	})
	for _, sub := range []string{
		"resolve_mysqldump",
		"MYSQLDUMP_BIN_PRESET='/export/servers/app/mysql-5.7.22/bin/mysqldump'",
		`MYSQLDUMP_BIN=$(resolve_mysqldump)`,
		`DUMP_CMD=("$MYSQLDUMP_BIN"`,
		"mysqldump bin=$MYSQLDUMP_BIN",
		"-S '/export/servers/data/my3306/run/mysqld.sock'",
		"connect=socket=/export/servers/data/my3306/run/mysqld.sock user=root",
		"mysqldump pipeline failed",
	} {
		if !strings.Contains(script, sub) {
			t.Fatalf("script missing %q", sub)
		}
	}
}

func TestNormalizeMysqldumpBin(t *testing.T) {
	if _, err := NormalizeMysqldumpBin("mysqldump"); err == nil {
		t.Fatal("expected error for relative path")
	}
	got, err := NormalizeMysqldumpBin("/usr/bin/mysqldump")
	if err != nil || got != "/usr/bin/mysqldump" {
		t.Fatalf("unexpected: %q %v", got, err)
	}
}
