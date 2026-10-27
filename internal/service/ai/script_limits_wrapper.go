package ai

import (
	"fmt"
	"os/exec"
)

// scriptLimitShellPrelude 生成 sh 前导命令：在 exec 目标程序前设置 rlimit。
// Go 的 syscall.SysProcAttr 没有 rlimit 字段，只能借助 ulimit 内置命令；
// exec 后 PID 与进程组不变，限制由目标程序及其子进程继承。
func scriptLimitShellPrelude(timeoutSec int, scriptLang string) string {
	// ulimit -v 单位为 KB；go run 需要编译，虚拟内存放宽到 2GiB
	memKB := 512 * 1024
	if scriptLang == "go" {
		memKB = 2 * 1024 * 1024
	}
	return fmt.Sprintf(
		"ulimit -t %d; ulimit -n 256; ulimit -f 65536; ulimit -v %d 2>/dev/null || true; exec \"$0\" \"$@\"",
		timeoutSec+15, memKB,
	)
}

// wrapWithScriptLimits 把 cmd 改写为 sh -c <prelude> <原命令...>。
func wrapWithScriptLimits(cmd *exec.Cmd, prelude string) {
	args := cmd.Args
	if len(args) == 0 {
		args = []string{cmd.Path}
	}
	cmd.Path = "/bin/sh"
	cmd.Args = append([]string{"sh", "-c", prelude}, args...)
}
