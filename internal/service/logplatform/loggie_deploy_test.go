package logplatform

import (
	"strings"
	"testing"
)

// 回归：单元文件里出现双引号会让 systemd 报 "path is not absolute"（曾用 %q 生成）。
func TestRenderSystemdUnitPathsUnquoted(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"/export/loggie", "/opt/loggie-agent"} {
		unit := renderSystemdUnit(dir)
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
	}
}

func TestRenderHeartbeatUnitPathsUnquoted(t *testing.T) {
	t.Parallel()
	dir := "/export/loggie"
	unit := renderHeartbeatUnit(dir)
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
