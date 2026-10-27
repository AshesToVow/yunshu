package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runScriptTool 在沙箱内执行脚本工具。
func (s *Service) runScriptTool(ctx context.Context, def toolDefRow, argsJSON string) (string, error) {
	root, err := filepath.Abs(s.dataRoot())
	if err != nil {
		return "", err
	}
	rel := filepath.Clean(filepath.FromSlash(def.ScriptPath))
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("非法脚本路径")
	}
	full := filepath.Join(root, rel)
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absFull, root+string(os.PathSeparator)) && absFull != root {
		return "", fmt.Errorf("脚本路径逃逸")
	}
	if _, err := os.Stat(absFull); err != nil {
		return "", fmt.Errorf("脚本不存在: %s", def.ScriptPath)
	}

	timeout := def.TimeoutSec
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120
	}
	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	lang := strings.ToLower(strings.TrimSpace(def.ScriptLang))
	switch lang {
	case "python27", "python", "python3", "py":
		interp := s.pythonBin()
		cmd = exec.CommandContext(cctx, interp, absFull)
	case "shell", "sh", "bash":
		shell := "sh"
		if _, err := exec.LookPath("bash"); err == nil {
			shell = "bash"
		}
		cmd = exec.CommandContext(cctx, shell, absFull)
	case "go":
		goBin, err := exec.LookPath("go")
		if err != nil {
			return "", fmt.Errorf("未找到 go 工具链: %w", err)
		}
		cmd = exec.CommandContext(cctx, goBin, "run", absFull)
	default:
		return "", fmt.Errorf("不支持的 script_lang: %s", def.ScriptLang)
	}
	cmd.Dir = filepath.Dir(absFull)
	// 环境变量白名单：防止脚本继承 DB DSN、encryptionKey、LLM Key 等敏感配置。
	// 确需透传的变量经 YUNSHU_AI_SCRIPT_ENV（逗号分隔）显式放行。
	cmd.Env = buildScriptEnv()
	applyScriptLimits(cmd, timeout, lang)
	cmd.Stdin = strings.NewReader(argsJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		msg := err.Error()
		if se := strings.TrimSpace(stderr.String()); se != "" {
			msg = se
		}
		return "", fmt.Errorf("脚本执行失败: %s", truncateStr(msg, 2000))
	}
	if out == "" {
		out = `{"ok":true}`
	}
	// 校验 JSON；非 JSON 则包装
	var js any
	if json.Unmarshal([]byte(out), &js) != nil {
		wrapped, _ := json.Marshal(map[string]any{"ok": true, "output": out})
		out = string(wrapped)
	}
	return truncateStr(out, 24_000), nil
}

// scriptEnvWhitelist 允许脚本进程继承的环境变量白名单。
var scriptEnvWhitelist = []string{"PATH", "HOME", "TMPDIR", "TMP", "TEMP", "LANG", "LC_ALL", "LC_CTYPE", "TERM"}

func buildScriptEnv() []string {
	allow := make(map[string]struct{}, len(scriptEnvWhitelist))
	for _, k := range scriptEnvWhitelist {
		allow[strings.ToLower(k)] = struct{}{}
	}
	if extra := strings.TrimSpace(os.Getenv("YUNSHU_AI_SCRIPT_ENV")); extra != "" {
		for _, k := range strings.Split(extra, ",") {
			if k = strings.TrimSpace(k); k != "" {
				allow[strings.ToLower(k)] = struct{}{}
			}
		}
	}
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if _, ok := allow[strings.ToLower(k)]; ok {
			env = append(env, kv)
		}
	}
	return env
}

func (s *Service) pythonBin() string {	if v := strings.TrimSpace(os.Getenv("YUNSHU_AI_PYTHON")); v != "" {
		return v
	}
	for _, c := range []string{"python2.7", "python2", "python3", "python"} {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return "python"
}

type toolDefRow struct {
	Name                string
	Description         string
	Runtime             string
	HandlerKey          string
	ScriptLang          string
	ScriptPath          string
	TimeoutSec          int
	InputSchemaJSON     string
	Permission          string
	RiskLevel           string
	RequireConfirmation bool
	Enabled             bool
}
