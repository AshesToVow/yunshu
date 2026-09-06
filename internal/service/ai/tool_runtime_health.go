package ai

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ToolRuntimeHealth 脚本工具运行时健康（Python/Shell 等）。
type ToolRuntimeHealth struct {
	OK           bool              `json:"ok"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	DataRoot     string            `json:"data_root"`
	DataRootOK   bool              `json:"data_root_ok"`
	PythonBin    string            `json:"python_bin"`
	PythonOK     bool              `json:"python_ok"`
	PythonVer    string            `json:"python_version,omitempty"`
	ShellBin     string            `json:"shell_bin"`
	ShellOK      bool              `json:"shell_ok"`
	EnvHints     map[string]string `json:"env_hints"`
	CheckedAt    time.Time         `json:"checked_at"`
	Suggestions  []string          `json:"suggestions,omitempty"`
}

// ToolRuntimeHealth 检查 AI 脚本工具依赖（容器内是否有 python 等）。
func (s *Service) ToolRuntimeHealth(ctx context.Context) *ToolRuntimeHealth {
	out := &ToolRuntimeHealth{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		DataRoot:  s.dataRoot(),
		EnvHints:  map[string]string{},
		CheckedAt: time.Now(),
	}
	if st, err := os.Stat(out.DataRoot); err == nil && st.IsDir() {
		out.DataRootOK = true
	}
	if v := strings.TrimSpace(os.Getenv("YUNSHU_AI_PYTHON")); v != "" {
		out.EnvHints["YUNSHU_AI_PYTHON"] = v
	}
	out.EnvHints["YUNSHU_AI_DATA_DIR"] = strings.TrimSpace(os.Getenv("YUNSHU_AI_DATA_DIR"))

	py := s.pythonBin()
	out.PythonBin = py
	if p, err := exec.LookPath(py); err == nil {
		out.PythonBin = p
		out.PythonOK = true
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cctx, p, "-V")
		if b, e := cmd.CombinedOutput(); e == nil {
			out.PythonVer = strings.TrimSpace(string(b))
		}
	}

	shell := "sh"
	if _, err := exec.LookPath("bash"); err == nil {
		shell = "bash"
	}
	out.ShellBin = shell
	if _, err := exec.LookPath(shell); err == nil {
		out.ShellOK = true
	}

	if !out.PythonOK {
		out.Suggestions = append(out.Suggestions,
			"当前进程环境无 Python：docker-compose 请在 Dockerfile.backend 安装 python3 后重建镜像，或设置 YUNSHU_AI_PYTHON",
			"linux.*.check 脚本工具依赖 Python；远端主机请用 probe_server_metrics",
		)
	}
	if !out.DataRootOK {
		out.Suggestions = append(out.Suggestions, "data/ai 目录不可用：检查 YUNSHU_AI_DATA_DIR 与镜像内 /app/data/ai")
	}
	// 抽样确认脚本文件在
	sample := filepath.Join(out.DataRoot, "tools", "linux", "disk_check", "run.py")
	if _, err := os.Stat(sample); err != nil && out.DataRootOK {
		out.Suggestions = append(out.Suggestions, "缺少 "+sample+"，请确认镜像已 COPY data/ai 或挂载种子目录")
	}

	out.OK = out.PythonOK && out.ShellOK && out.DataRootOK
	return out
}
