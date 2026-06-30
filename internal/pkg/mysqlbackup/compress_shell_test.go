package mysqlbackup

import (
	"strings"
	"testing"
)

func TestXtrabackupScriptRequiresGzip(t *testing.T) {
	q := func(s string) string { return "'" + s + "'" }
	cli, xb, log := FormatMysqlConnectShell("/export/servers/data/my3306/run/mysqld.sock", "10.10.10.103", 3306, "root", q)
	script := BuildXtrabackupRemoteScript(XtrabackupRemoteScriptParams{
		DataDir: "/data", LogDir: "/log", Basename: "test",
		MySQLPass: "'x'", ToolPref: XtrabackupToolInnobackupex,
		InnobackupexBin: "/usr/bin/innobackupex",
		ConnectLog: log, CLIConnect: cli, XBConnect: xb, ShellQuote: q,
	})
	for _, sub := range []string{
		"resolve_physical_backup_bin",
		"pick_physical_backup_tool",
		"INNOBACKUPEX_BIN_PRESET='/usr/bin/innobackupex'",
		"XTRABACKUP_TOOL_PRESET='innobackupex'",
		"--no-timestamp",
		"--apply-log",
		"resolve_gzip",
		`tar -cf - -C "$TMP"`,
		BackupCompletedMarker,
		"tool=$XB_KIND",
	} {
		if !strings.Contains(script, sub) {
			t.Fatalf("script missing %q", sub)
		}
	}
}

func TestXtrabackupScriptAutoMode(t *testing.T) {
	q := func(s string) string { return "'" + s + "'" }
	cli, xb, log := FormatMysqlConnectShell("", "127.0.0.1", 3306, "root", q)
	script := BuildXtrabackupRemoteScript(XtrabackupRemoteScriptParams{
		DataDir: "/data", LogDir: "/log", Basename: "test",
		MySQLPass: "'x'", ToolPref: XtrabackupToolAuto,
		ConnectLog: log, CLIConnect: cli, XBConnect: xb, ShellQuote: q,
	})
	if !strings.Contains(script, "XTRABACKUP_TOOL_PRESET='auto'") {
		t.Fatal("missing auto preset")
	}
	if !strings.Contains(script, `--target-dir="$TMP"`) {
		t.Fatal("missing xtrabackup target-dir branch")
	}
}
