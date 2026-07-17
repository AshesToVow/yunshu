package logplatform

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/pkg/sshserver"
)

const loggieRestartUnit = "loggie.service"
const loggieBinaryName = "loggie"
const startScriptFilename = "start.sh"
const heartbeatTimerUnit = "loggie-heartbeat.timer"
const heartbeatOneshotUnit = "loggie-heartbeat.service"

func (s *LoggieAgentService) unitName() string {
	u := strings.TrimSpace(s.loggieCfg.UnitName)
	if u == "" {
		return loggieRestartUnit
	}
	return u
}

func (s *LoggieAgentService) defaultDeployDir() string {
	d := strings.TrimSpace(s.loggieCfg.DeployDir)
	if d == "" {
		return defaultLoggieDeployDir
	}
	return strings.TrimRight(d, "/")
}

func (s *LoggieAgentService) resolveOfflineBinaryPath() (string, error) {
	cfgPath := strings.TrimSpace(s.loggieCfg.OfflineBinaryPath)
	if cfgPath == "" {
		cfgPath = "deploy/loggie/binary/loggie"
	}
	var candidates []string
	if filepath.IsAbs(cfgPath) {
		candidates = append(candidates, cfgPath)
	} else {
		if wd, err := os.Getwd(); err == nil {
			candidates = append(candidates, filepath.Join(wd, cfgPath))
		}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), cfgPath))
		}
		candidates = append(candidates, filepath.Join("/app", cfgPath))
	}
	for _, c := range candidates {
		st, err := os.Stat(c)
		if err != nil || st.IsDir() {
			continue
		}
		if st.Size() < 1024 {
			continue
		}
		return c, nil
	}
	return "", fmt.Errorf("离线二进制未找到（请将 loggie 放到 deploy/loggie/binary/loggie）：%s", strings.Join(candidates, "; "))
}

func loadOfflineLoggieBinary(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 1024 {
		return nil, fmt.Errorf("offline binary too small (%d bytes): %s", len(data), path)
	}
	if len(data) < 4 || data[0] != 0x7f || string(data[1:4]) != "ELF" {
		return nil, fmt.Errorf("offline binary is not ELF: %s", path)
	}
	return data, nil
}

func (s *LoggieAgentService) deployBundleOverSSH(ctx context.Context, serverID uint, bundle LoggiePipelineBundle) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置，无法远程下发")
	}
	cli, _, err := sshserver.DialServer(ctx, s.aead, "loggie.deploy", s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()

	deployDir := strings.TrimRight(strings.TrimSpace(bundle.DeployDir), "/")
	if deployDir == "" {
		deployDir = s.defaultDeployDir()
	}
	if _, err := cli.Exec(ctx, fmt.Sprintf("mkdir -p %q", deployDir), 4096); err != nil {
		return "", "", fmt.Errorf("mkdir: %w", err)
	}
	pipelines := strings.TrimSpace(bundle.PipelinesOnlyYAML)
	if pipelines == "" {
		pipelines = bundle.PipelineYAML
	}
	files := []struct {
		name    string
		content []byte
		perm    os.FileMode
	}{
		{pipelinesOnlyFilename, []byte(pipelines + "\n"), 0o644},
		{"pipeline.yml", []byte(bundle.PipelineYAML + "\n"), 0o644},
		{bundle.EnvFilename, []byte(bundle.EnvFile), 0o644},
		{bundle.HeartbeatFilename, []byte(bundle.HeartbeatScript), 0o755},
		{startScriptFilename, []byte(renderStartScript()), 0o755},
		{"loggie.service", []byte(renderSystemdUnit(deployDir)), 0o644},
		{"loggie-heartbeat.service", []byte(renderHeartbeatUnit(deployDir)), 0o644},
		{"loggie-heartbeat.timer", []byte(renderHeartbeatTimer()), 0o644},
	}
	for _, f := range files {
		if strings.TrimSpace(f.name) == "" || len(f.content) == 0 {
			continue
		}
		remote := path.Join(deployDir, f.name)
		if err := cli.UploadBytes(ctx, remote, f.content, f.perm); err != nil {
			return "", "", fmt.Errorf("upload %s: %w", remote, err)
		}
	}
	unit := s.unitName()
	// 热更配置后必须刷新 heartbeat timer，否则 Token/env 更新后平台仍显示离线。
	cmd := fmt.Sprintf(
		`set +e; DEPLOY=%q; UNIT=%q; `+
			`chmod +x "$DEPLOY/heartbeat.sh" "$DEPLOY/start.sh" 2>/dev/null; `+
			`if systemctl cat "$UNIT" >/dev/null 2>&1; then `+
			`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT" 2>/dev/null; `+
			`sudo cp "$DEPLOY/loggie-heartbeat.service" /etc/systemd/system/loggie-heartbeat.service 2>/dev/null; `+
			`sudo cp "$DEPLOY/loggie-heartbeat.timer" /etc/systemd/system/loggie-heartbeat.timer 2>/dev/null; `+
			`sudo systemctl daemon-reload; `+
			`sudo systemctl reload-or-restart "$UNIT" 2>&1 || sudo systemctl restart "$UNIT" 2>&1; RC=$?; `+
			`sudo systemctl enable loggie-heartbeat.timer >/dev/null 2>&1; `+
			`sudo systemctl restart loggie-heartbeat.timer 2>&1; `+
			`sudo systemctl start loggie-heartbeat.service >/dev/null 2>&1 || true; `+
			`exit $RC; `+
			`else echo "CONFIG_UPLOADED unit=$UNIT not installed yet; run Install first"; exit 0; fi`,
		deployDir, unit,
	)
	res, err := cli.Exec(ctx, cmd, 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		msg := strings.TrimSpace(res.Stdout + res.Stderr)
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return res.Stdout, res.Stderr, fmt.Errorf("reload loggie failed: %s", msg)
	}
	return res.Stdout, res.Stderr, nil
}

func (s *LoggieAgentService) systemctlLoggie(ctx context.Context, serverID uint, action string) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置")
	}
	cli, _, err := sshserver.DialServer(ctx, s.aead, "loggie."+action, s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()
	res, err := cli.Exec(ctx, buildLoggieLifecycleScript(s.unitName(), action), 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("systemctl %s exit=%d: %s", action, res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
}

func buildLoggieLifecycleScript(unit, action string) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail; ")
	b.WriteString(fmt.Sprintf("UNIT=%q; HB_T=%q; HB_S=%q; ", unit, heartbeatTimerUnit, heartbeatOneshotUnit))
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "start":
		b.WriteString(`sudo systemctl start "$UNIT"; `)
		b.WriteString(`sudo systemctl enable "$HB_T" >/dev/null 2>&1 || true; `)
		b.WriteString(`sudo systemctl restart "$HB_T"; `)
		b.WriteString(`sudo systemctl start "$HB_S" || true`)
	case "stop":
		b.WriteString(`sudo systemctl stop "$UNIT" "$HB_T" "$HB_S" 2>/dev/null || true`)
	case "restart":
		b.WriteString(`sudo systemctl restart "$UNIT"; `)
		b.WriteString(`sudo systemctl enable "$HB_T" >/dev/null 2>&1 || true; `)
		b.WriteString(`sudo systemctl restart "$HB_T"; `)
		b.WriteString(`sudo systemctl start "$HB_S" || true`)
	default:
		b.WriteString(fmt.Sprintf(`sudo systemctl %s "$UNIT"`, action))
	}
	return b.String()
}

func (s *LoggieAgentService) restartLoggieOverSSH(ctx context.Context, serverID uint) (stdout, stderr string, err error) {
	return s.systemctlLoggie(ctx, serverID, "restart")
}

func (s *LoggieAgentService) startLoggieOverSSH(ctx context.Context, serverID uint) (stdout, stderr string, err error) {
	return s.systemctlLoggie(ctx, serverID, "start")
}

func (s *LoggieAgentService) stopLoggieOverSSH(ctx context.Context, serverID uint) (stdout, stderr string, err error) {
	return s.systemctlLoggie(ctx, serverID, "stop")
}

func (s *LoggieAgentService) uninstallLoggieOverSSH(ctx context.Context, serverID uint, deployDir string, removeFiles bool) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置")
	}
	uninstallCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	cli, _, err := sshserver.DialServer(uninstallCtx, s.aead, "loggie.uninstall", s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()

	dir := strings.TrimRight(strings.TrimSpace(deployDir), "/")
	if dir == "" {
		dir = s.defaultDeployDir()
	}
	res, err := cli.Exec(uninstallCtx, buildUninstallScript(dir, s.unitName(), removeFiles), 65536)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("uninstall exit=%d: %s", res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
}

func (s *LoggieAgentService) installLoggieOverSSH(ctx context.Context, serverID uint, bundle LoggiePipelineBundle, _binaryURL string) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置，无法远程安装")
	}
	installCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Minute)
	defer cancel()

	cli, _, err := sshserver.DialServer(installCtx, s.aead, "loggie.install", s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()

	deployDir := strings.TrimRight(strings.TrimSpace(bundle.DeployDir), "/")
	if deployDir == "" {
		deployDir = s.defaultDeployDir()
	}
	unit := s.unitName()
	binRemote := path.Join(deployDir, loggieBinaryName)

	var out, errOut strings.Builder
	mkdir := fmt.Sprintf("mkdir -p %q", deployDir)
	res, err := cli.Exec(installCtx, mkdir, 4096)
	out.WriteString(res.Stdout)
	errOut.WriteString(res.Stderr)
	if err != nil || res.ExitCode != 0 {
		return out.String(), errOut.String(), fmt.Errorf("mkdir deploy dir: %w", err)
	}

	localBin, err := s.resolveOfflineBinaryPath()
	if err != nil {
		return out.String(), errOut.String(), err
	}
	binData, err := loadOfflineLoggieBinary(localBin)
	if err != nil {
		return out.String(), errOut.String(), err
	}
	out.WriteString(fmt.Sprintf("offline binary: %s (%d bytes)\n", localBin, len(binData)))

	pipelines := strings.TrimSpace(bundle.PipelinesOnlyYAML)
	if pipelines == "" {
		pipelines = bundle.PipelineYAML
	}
	files := []struct {
		name    string
		content []byte
		perm    os.FileMode
	}{
		{"pipeline.yml", []byte(bundle.PipelineYAML + "\n"), 0o644},
		{pipelinesOnlyFilename, []byte(pipelines + "\n"), 0o644},
		{bundle.EnvFilename, []byte(bundle.EnvFile), 0o644},
		{bundle.HeartbeatFilename, []byte(bundle.HeartbeatScript), 0o755},
		{startScriptFilename, []byte(renderStartScript()), 0o755},
		{"loggie.service", []byte(renderSystemdUnit(deployDir)), 0o644},
		{"loggie-heartbeat.service", []byte(renderHeartbeatUnit(deployDir)), 0o644},
		{"loggie-heartbeat.timer", []byte(renderHeartbeatTimer()), 0o644},
	}
	for _, f := range files {
		if len(f.content) == 0 {
			continue
		}
		remote := path.Join(deployDir, f.name)
		if err := cli.UploadBytes(installCtx, remote, f.content, f.perm); err != nil {
			return out.String(), errOut.String(), fmt.Errorf("upload %s: %w", remote, err)
		}
	}

	if err := cli.UploadBytes(installCtx, binRemote, binData, 0o755); err != nil {
		return out.String(), errOut.String(), fmt.Errorf("upload binary: %w", err)
	}
	out.WriteString(fmt.Sprintf("uploaded binary %s (%d bytes)\n", binRemote, len(binData)))

	res, err = cli.Exec(installCtx, buildFinalizeInstallScript(deployDir, unit), 65536)
	out.WriteString(res.Stdout)
	errOut.WriteString(res.Stderr)
	if err != nil {
		return out.String(), errOut.String(), err
	}
	if res.ExitCode != 0 {
		return out.String(), errOut.String(), fmt.Errorf("install exit=%d: %s", res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return out.String(), errOut.String(), nil
}

// 目录约定：
//
//	/export/loggie/loggie          # 二进制
//	/export/loggie/*.yml / *.sh    # 配置与脚本
func renderSystemdUnit(deployDir string) string {
	bin := path.Join(deployDir, loggieBinaryName)
	return fmt.Sprintf(`[Unit]
Description=Loggie log collector (Yunshu)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=DBUS_SESSION_BUS_ADDRESS=disabled:
WorkingDirectory=%s
ExecStart=%s -config.system=%s/pipeline.yml -config.pipeline=%s/pipelines.yml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, deployDir, bin, deployDir, deployDir)
}

func renderStartScript() string {
	return `#!/usr/bin/env bash
# 手工调试用；生产请用: systemctl start loggie.service
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$DIR/loggie"
SYS="$DIR/pipeline.yml"
PIPE="$DIR/pipelines.yml"

if [[ ! -x "$BIN" ]]; then
  echo "missing binary: $BIN" >&2
  exit 1
fi
if command -v file >/dev/null 2>&1; then
  INFO="$(file -b "$BIN" || true)"
  if echo "$INFO" | grep -qiE 'ASCII|HTML|text|empty'; then
    echo "bad binary (not ELF): $BIN — $INFO" >&2
    exit 1
  fi
fi
if [[ ! -f "$SYS" || ! -f "$PIPE" ]]; then
  echo "need both pipeline.yml (system) and pipelines.yml under $DIR" >&2
  exit 1
fi

cmd="${1:-start}"
case "$cmd" in
  start|run|foreground)
    exec "$BIN" -config.system="$SYS" -config.pipeline="$PIPE"
    ;;
  *)
    echo "usage: $0 [start|foreground]" >&2
    exit 1
    ;;
esac
`
}

func renderHeartbeatUnit(deployDir string) string {
	return fmt.Sprintf(`[Unit]
Description=Yunshu Loggie heartbeat (oneshot)
After=network-online.target

[Service]
Type=oneshot
WorkingDirectory=%s
EnvironmentFile=-%s/loggie-heartbeat.env
ExecStart=/bin/bash %s/heartbeat.sh
Nice=10
`, deployDir, deployDir, deployDir)
}

func renderHeartbeatTimer() string {
	return `[Unit]
Description=Yunshu Loggie heartbeat every 60s

[Timer]
OnBootSec=30s
OnUnitActiveSec=60s
AccuracySec=5s
Unit=loggie-heartbeat.service

[Install]
WantedBy=timers.target
`
}

func buildFinalizeInstallScript(deployDir, unit string) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail; ")
	b.WriteString(fmt.Sprintf("DEPLOY=%q; UNIT=%q; BIN=%q/loggie; ", deployDir, unit, deployDir))
	b.WriteString(`ARCH=$(uname -m); echo "host_arch=$ARCH"; `)
	b.WriteString(`if [[ ! -f "$BIN" ]]; then echo "missing binary $BIN"; exit 1; fi; `)
	b.WriteString(`chmod +x "$BIN" "$DEPLOY/start.sh" "$DEPLOY/heartbeat.sh" 2>/dev/null || true; `)
	b.WriteString(`INFO=$(file -b "$BIN" 2>/dev/null || echo unknown); echo "binary_file=$INFO"; `)
	b.WriteString(`echo "$INFO" | grep -qi ELF || { echo "not an ELF binary: $INFO"; exit 1; }; `)
	b.WriteString(`case "$ARCH" in `)
	b.WriteString(`x86_64|amd64) echo "$INFO" | grep -qiE 'x86-64|x86_64|AMD64|Intel 80386' || { echo "arch mismatch: host amd64, binary=$INFO"; exit 1; } ;; `)
	b.WriteString(`aarch64|arm64) echo "ERROR: host is arm64; offline package must be linux/arm64 ELF"; exit 1 ;; `)
	b.WriteString(`*) echo "unsupported arch: $ARCH"; exit 1 ;; esac; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT"; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie-heartbeat.service" /etc/systemd/system/loggie-heartbeat.service; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie-heartbeat.timer" /etc/systemd/system/loggie-heartbeat.timer; `)
	b.WriteString(`sudo systemctl daemon-reload; `)
	b.WriteString(`sudo systemctl enable "$UNIT" loggie-heartbeat.timer; `)
	b.WriteString(`sudo systemctl restart "$UNIT"; `)
	b.WriteString(`sudo systemctl restart loggie-heartbeat.timer; `)
	b.WriteString(`sudo systemctl start loggie-heartbeat.service || true; `)
	b.WriteString(`echo "installed: $BIN"; ls -la "$DEPLOY"; `)
	b.WriteString(`systemctl is-active "$UNIT" || true; systemctl is-active loggie-heartbeat.timer || true`)
	return b.String()
}

func buildUninstallScript(deployDir, unit string, removeFiles bool) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail; ")
	b.WriteString(fmt.Sprintf("DEPLOY=%q; UNIT=%q; ", deployDir, unit))
	b.WriteString(`case "$DEPLOY" in ""|"/"|"/export"|"/opt"|"/usr"|"/var"|"/home"|"/root") echo "refuse unsafe deploy dir: $DEPLOY"; exit 1 ;; esac; `)
	b.WriteString(`sudo systemctl stop "$UNIT" loggie-heartbeat.timer loggie-heartbeat.service 2>/dev/null || true; `)
	b.WriteString(`sudo systemctl disable "$UNIT" loggie-heartbeat.timer 2>/dev/null || true; `)
	b.WriteString(`sudo rm -f "/etc/systemd/system/$UNIT" /etc/systemd/system/loggie-heartbeat.service /etc/systemd/system/loggie-heartbeat.timer; `)
	b.WriteString(`sudo systemctl daemon-reload; sudo systemctl reset-failed "$UNIT" 2>/dev/null || true; `)
	if removeFiles {
		b.WriteString(`if [[ -d "$DEPLOY" ]]; then sudo rm -rf "$DEPLOY"; echo "removed $DEPLOY"; else echo "deploy dir absent: $DEPLOY"; fi; `)
	} else {
		b.WriteString(`echo "kept deploy files under $DEPLOY"; `)
	}
	b.WriteString(`echo "uninstalled unit=$UNIT"`)
	return b.String()
}

func truncateDeployOutput(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func formatDeployTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
