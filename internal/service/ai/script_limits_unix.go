//go:build !windows

package ai

import (
	"os/exec"
	"syscall"
)

// applyScriptLimits 限制脚本进程的 CPU/内存/文件资源；Setpgid + Cancel 保证超时能杀掉整个进程组
// （否则 bash/python 的孙进程、go run 的编译产物会逃逸成孤儿）。
func applyScriptLimits(cmd *exec.Cmd, timeoutSec int, scriptLang string) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	wrapWithScriptLimits(cmd, scriptLimitShellPrelude(timeoutSec, scriptLang))
}
