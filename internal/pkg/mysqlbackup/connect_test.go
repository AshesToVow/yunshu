package mysqlbackup

import "testing"

func TestFormatMysqlConnectShellSocket(t *testing.T) {
	q := func(s string) string { return "'" + s + "'" }
	cli, xb, log := FormatMysqlConnectShell("/export/servers/data/my3306/run/mysqld.sock", "10.10.10.103", 3306, "root", q)
	if cli != "-S '/export/servers/data/my3306/run/mysqld.sock' -u'root'" {
		t.Fatalf("unexpected cli: %s", cli)
	}
	if xb != "--socket='/export/servers/data/my3306/run/mysqld.sock' --user='root'" {
		t.Fatalf("unexpected xb: %s", xb)
	}
	if log != "socket=/export/servers/data/my3306/run/mysqld.sock user=root" {
		t.Fatalf("unexpected log: %s", log)
	}
}

func TestFormatMysqldumpConnectArgsSocket(t *testing.T) {
	q := func(s string) string { return "'" + s + "'" }
	args, log := FormatMysqldumpConnectArgs("/export/servers/data/my3306/run/mysqld.sock", "10.10.10.103", 3306, "root", q)
	if args != "-S '/export/servers/data/my3306/run/mysqld.sock' -u'root'" {
		t.Fatalf("unexpected args: %s", args)
	}
	if log != "socket=/export/servers/data/my3306/run/mysqld.sock user=root" {
		t.Fatalf("unexpected log: %s", log)
	}
}

func TestFormatMysqldumpConnectArgsTCP(t *testing.T) {
	q := func(s string) string { return "'" + s + "'" }
	args, log := FormatMysqldumpConnectArgs("", "10.10.10.103", 3306, "root", q)
	if args != "-h'10.10.10.103' -P3306 -u'root'" {
		t.Fatalf("unexpected args: %s", args)
	}
	if log != "host=10.10.10.103 port=3306 user=root" {
		t.Fatalf("unexpected log: %s", log)
	}
}

func TestNormalizeMysqlSocket(t *testing.T) {
	if _, err := NormalizeMysqlSocket("relative.sock"); err == nil {
		t.Fatal("expected error for relative path")
	}
	got, err := NormalizeMysqlSocket("/tmp/mysql.sock")
	if err != nil || got != "/tmp/mysql.sock" {
		t.Fatalf("unexpected: %q %v", got, err)
	}
}
