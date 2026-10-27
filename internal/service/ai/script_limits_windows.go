//go:build windows

package ai

import "os/exec"

// applyScriptLimits Windows 仅用于本地开发，无 rlimit/进程组语义；
// 生产部署为 Linux，资源限制由 script_limits_unix.go 生效，此处保留 CommandContext 超时兜底。
func applyScriptLimits(cmd *exec.Cmd, timeoutSec int, scriptLang string) {}
