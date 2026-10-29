package logplatform

import (
	"strings"
	"testing"
)

// 回归：单元文件里出现双引号会让 systemd 报 "path is not absolute"（曾用 %q 生成）。
func TestRenderSystemdUnitPathsUnquoted(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"/export/loggie", "/opt/loggie-agent"} {
		unit := renderSystemdUnit(dir, "", "")
		if strings.Contains(unit, `"`) {
			t.Fatalf("unit for %q must not contain double quotes:\n%s", dir, unit)
		}
		wantWD := "WorkingDirectory=" + dir + "\n"
		if !strings.Contains(unit, wantWD) {
			t.Fatalf("unit for %q missing %q:\n%s", dir, wantWD, unit)
		}
		wantExec := "ExecStart=" + dir + "/loggie -config.system=" + dir + "/pipeline.yml -config.pipeline=" + dir + "/pipelines.yml\n"
		if !strings.Contains(unit, wantExec) {
			t.Fatalf("unit for %q missing %q:\n%s", dir, wantExec, unit)
		}
		if !strings.Contains(unit, "User=loggie\n") {
			t.Fatalf("default run user must be loggie:\n%s", unit)
		}
	}
}

func TestRenderSystemdUnitCustomRunIdentity(t *testing.T) {
	t.Parallel()
	unit := renderSystemdUnit("/export/loggie", "yunshu", "yunshu-app")
	for _, want := range []string{"User=yunshu\n", "Group=yunshu-app\n"} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
	if strings.Contains(unit, "User=loggie") {
		t.Fatalf("custom identity must not fall back to loggie:\n%s", unit)
	}
}

func TestResolveLoggieRunIdentity(t *testing.T) {
	t.Parallel()
	cases := []struct{ user, group, wantUser, wantGroup string }{
		{"", "", "loggie", "loggie"},
		{"yunshu", "", "yunshu", "yunshu"},
		{"", "app", "loggie", "app"},
		{" yunshu ", " app ", "yunshu", "app"},
	}
	for _, c := range cases {
		u, g := resolveLoggieRunIdentity(c.user, c.group)
		if u != c.wantUser || g != c.wantGroup {
			t.Fatalf("resolve(%q,%q)=(%q,%q), want (%q,%q)", c.user, c.group, u, g, c.wantUser, c.wantGroup)
		}
	}
}

func TestValidateLoggieRunIdentity(t *testing.T) {
	t.Parallel()
	valid := [][2]string{
		{"", ""},
		{"loggie", "loggie"},
		{"yunshu", ""},
		{"_svc", "app-group"},
		{"a1_2-3", "a1_2-3"},
	}
	for _, c := range valid {
		if err := validateLoggieRunIdentity(c[0], c[1]); err != nil {
			t.Fatalf("expected valid for (%q,%q), got %v", c[0], c[1], err)
		}
	}
	invalid := [][2]string{
		{"1root", ""},
		{"Run", ""},
		{"bad user", ""},
		{"user;rm", ""},
		{"user$(id)", ""},
		{"-dash", ""},
		{"", "Root"},
		{strings.Repeat("a", 33), ""},
	}
	for _, c := range invalid {
		if err := validateLoggieRunIdentity(c[0], c[1]); err == nil {
			t.Fatalf("expected error for (%q,%q)", c[0], c[1])
		}
	}
}

// 回归：自定义运行用户时，provison 脚本要用变量而非硬编码 loggie，
// 并把 <部署目录>/data 交给该用户（registry sqlite 只读的根因）。
func TestShellServiceUserProvisionCustomIdentity(t *testing.T) {
	t.Parallel()
	script := shellServiceUserProvision("yunshu", "yunshu")
	for _, want := range []string{
		`RUN_USER="yunshu"`,
		`RUN_GROUP="yunshu"`,
		`--gid "$RUN_GROUP" "$RUN_USER"`,
		`group_exists "$RUN_GROUP"`,
		`"$DEPLOY/data"`,
		`/^User=$RUN_USER$/d`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("provision script missing %q:\n%s", want, script)
		}
	}
	if strings.Contains(script, "useradd --system --no-create-home --shell /usr/sbin/nologin loggie") {
		t.Fatalf("provision script must not hardcode loggie:\n%s", script)
	}
	defaultScript := shellServiceUserProvision("", "")
	if !strings.Contains(defaultScript, `RUN_USER="loggie"`) {
		t.Fatalf("default identity must be loggie:\n%s", defaultScript)
	}
}

func TestRenderHeartbeatUnitPathsUnquoted(t *testing.T) {
	t.Parallel()
	dir := "/export/loggie"
	unit := renderHeartbeatUnit(dir, "", "")
	if strings.Contains(unit, `"`) {
		t.Fatalf("heartbeat unit must not contain double quotes:\n%s", unit)
	}
	for _, want := range []string{
		"WorkingDirectory=" + dir + "\n",
		"EnvironmentFile=-" + dir + "/loggie-heartbeat.env\n",
		"ExecStart=/bin/bash " + dir + "/heartbeat.sh\n",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("heartbeat unit missing %q:\n%s", want, unit)
		}
	}
}
