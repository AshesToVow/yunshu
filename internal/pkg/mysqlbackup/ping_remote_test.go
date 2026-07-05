package mysqlbackup

import (
	"strings"
	"testing"
)

func TestDeriveMysqlBinFromMysqldump(t *testing.T) {
	got := DeriveMysqlBinFromMysqldump("/export/servers/app/mysql-5.7.22/bin/mysqldump")
	want := "/export/servers/app/mysql-5.7.22/bin/mysql"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildMysqlPingRemoteScriptUsesResolveMysql(t *testing.T) {
	script := BuildMysqlPingRemoteScript(
		"/export/servers/data/my3306/run/mysql.sock",
		"127.0.0.1", 3306, "root", "secret",
		"/export/servers/app/mysql-5.7.22/bin/mysqldump",
		func(s string) string { return "'" + s + "'" },
	)
	for _, want := range []string{
		"resolve_mysql",
		"MYSQLDUMP_BIN_PRESET='/export/servers/app/mysql-5.7.22/bin/mysqldump'",
		"MYSQL_BIN_PRESET='/export/servers/app/mysql-5.7.22/bin/mysql'",
		`"$MYSQL_BIN" --no-defaults -S '/export/servers/data/my3306/run/mysql.sock' -u'root'`,
		"status=1i",
		"status=0i error=",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
}
