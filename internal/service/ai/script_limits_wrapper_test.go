package ai

import (
	"os/exec"
	"strings"
	"testing"
)

func TestScriptLimitShellPrelude(t *testing.T) {
	py := scriptLimitShellPrelude(30, "python3")
	if !strings.Contains(py, "ulimit -t 45") {
		t.Fatalf("期望 CPU 限制为 timeout+15，实际 %q", py)
	}
	if !strings.Contains(py, "ulimit -v 524288") {
		t.Fatalf("期望 python 虚拟内存 512MiB，实际 %q", py)
	}
	goPrelude := scriptLimitShellPrelude(120, "go")
	if !strings.Contains(goPrelude, "ulimit -v 2097152") {
		t.Fatalf("期望 go 虚拟内存 2GiB，实际 %q", goPrelude)
	}
	if !strings.HasSuffix(py, `exec "$0" "$@"`) {
		t.Fatalf("前导命令必须以 exec 结尾，实际 %q", py)
	}
}

func TestWrapWithScriptLimits(t *testing.T) {
	cmd := exec.Command("python3", "/data/scripts/a.py", "x y")
	prelude := scriptLimitShellPrelude(10, "python3")
	wrapWithScriptLimits(cmd, prelude)
	if cmd.Path != "/bin/sh" {
		t.Fatalf("Path 应为 /bin/sh，实际 %q", cmd.Path)
	}
	want := []string{"sh", "-c", prelude, "python3", "/data/scripts/a.py", "x y"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("Args 长度不符: %#v", cmd.Args)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("Args[%d] 期望 %q 实际 %q", i, want[i], cmd.Args[i])
		}
	}

	empty := &exec.Cmd{Path: "/usr/bin/env"}
	wrapWithScriptLimits(empty, prelude)
	if empty.Args[3] != "/usr/bin/env" {
		t.Fatalf("Args 为空时应回退 Path，实际 %#v", empty.Args)
	}
}
