package logplatform

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"yunshu/internal/pkg/sshserver"
)

const loggieRestartUnit = "loggie.service"
const loggieBinaryName = "loggie"
const startScriptFilename = "start.sh"

// DefaultLoggieBinaryURL 官方 release 直链（裸二进制，非 tar）。
const DefaultLoggieBinaryURL = "https://github.com/loggie-io/loggie/releases/download/v1.5.0/loggie"

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

func (s *LoggieAgentService) resolveBinaryURL(override string) string {
	u := strings.TrimSpace(override)
	if u == "" {
		u = strings.TrimSpace(s.loggieCfg.BinaryURL)
	}
	if u == "" {
		u = DefaultLoggieBinaryURL
	}
	return u
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
	// 仅热更配置时单元可能尚未安装：先确认 unit 存在再 reload，否则只落盘成功提示安装。
	cmd := fmt.Sprintf(
		`set +e; UNIT=%q; `+
			`if systemctl cat "$UNIT" >/dev/null 2>&1; then `+
			`systemctl reload-or-restart "$UNIT" 2>&1 || systemctl restart "$UNIT" 2>&1; exit $?; `+
			`else echo "CONFIG_UPLOADED unit=$UNIT not installed yet; run Install first"; exit 0; fi`,
		unit,
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
	res, err := cli.Exec(ctx, fmt.Sprintf("systemctl %s %s", action, s.unitName()), 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("systemctl %s exit=%d: %s", action, res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
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

func (s *LoggieAgentService) installLoggieOverSSH(ctx context.Context, serverID uint, bundle LoggiePipelineBundle, binaryURL string) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置，无法远程安装")
	}
	cli, _, err := sshserver.DialServer(ctx, s.aead, "loggie.install", s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()

	deployDir := strings.TrimRight(strings.TrimSpace(bundle.DeployDir), "/")
	if deployDir == "" {
		deployDir = s.defaultDeployDir()
	}
	unit := s.unitName()
	binURL := s.resolveBinaryURL(binaryURL)

	var out, errOut strings.Builder
	mkdir := fmt.Sprintf("mkdir -p %q", deployDir)
	res, err := cli.Exec(ctx, mkdir, 4096)
	out.WriteString(res.Stdout)
	errOut.WriteString(res.Stderr)
	if err != nil || res.ExitCode != 0 {
		return out.String(), errOut.String(), fmt.Errorf("mkdir deploy dir: %w", err)
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
		{"pipeline.yml", []byte(bundle.PipelineYAML + "\n"), 0o644},
		{pipelinesOnlyFilename, []byte(pipelines + "\n"), 0o644},
		{bundle.EnvFilename, []byte(bundle.EnvFile), 0o644},
		{bundle.HeartbeatFilename, []byte(bundle.HeartbeatScript), 0o755},
		{startScriptFilename, []byte(renderStartScript()), 0o755},
		{"loggie.service", []byte(renderSystemdUnit(deployDir)), 0o644},
	}
	for _, f := range files {
		if len(f.content) == 0 {
			continue
		}
		remote := path.Join(deployDir, f.name)
		if err := cli.UploadBytes(ctx, remote, f.content, f.perm); err != nil {
			return out.String(), errOut.String(), fmt.Errorf("upload %s: %w", remote, err)
		}
	}

	installCmd := buildInstallRemoteScript(deployDir, unit, binURL)
	res, err = cli.Exec(ctx, installCmd, 65536)
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
# Yunshu Loggie start helper — 与二进制同目录
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
BIN="$DIR/loggie"
SYS="$DIR/pipeline.yml"
PIPE="$DIR/pipelines.yml"

if [[ ! -x "$BIN" ]]; then
  echo "missing binary: $BIN" >&2
  exit 1
fi
if [[ ! -f "$SYS" || ! -f "$PIPE" ]]; then
  echo "missing pipeline.yml or pipelines.yml under $DIR" >&2
  exit 1
fi

cmd="${1:-start}"
case "$cmd" in
  start|run)
    exec "$BIN" -config.system="$SYS" -config.pipeline="$PIPE"
    ;;
  foreground)
    exec "$BIN" -config.system="$SYS" -config.pipeline="$PIPE"
    ;;
  *)
    echo "usage: $0 [start|foreground]" >&2
    exit 1
    ;;
esac
`
}

func buildInstallRemoteScript(deployDir, unit, binaryURL string) string {
	// 直链裸二进制（如 .../v1.5.0/loggie）下载到 $DEPLOY/loggie
	var b strings.Builder
	b.WriteString("set -euo pipefail; ")
	b.WriteString(fmt.Sprintf("DEPLOY=%q; UNIT=%q; BINURL=%q; BIN=%q/loggie; ", deployDir, unit, binaryURL, deployDir))
	b.WriteString(`if [[ -z "$BINURL" ]]; then echo "binary_url required"; exit 1; fi; `)
	b.WriteString(`ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac; `)
	b.WriteString(`URL="${BINURL//\{arch\}/$ARCH}"; `)
	b.WriteString(`TMP=$(mktemp); `)
	b.WriteString(`(curl -fsSL "$URL" -o "$TMP" || wget -qO "$TMP" "$URL"); `)
	b.WriteString(`# 官方 release 多为裸二进制；若误下到压缩包则解压取 loggie `)
	b.WriteString(`if file "$TMP" 2>/dev/null | grep -qiE 'gzip|tar archive'; then `)
	b.WriteString(`TD=$(mktemp -d); tar -xzf "$TMP" -C "$TD"; F=$(find "$TD" -type f -name loggie | head -1); `)
	b.WriteString(`install -m 755 "$F" "$BIN"; rm -rf "$TD" "$TMP"; `)
	b.WriteString(`elif file "$TMP" 2>/dev/null | grep -qi 'Zip'; then `)
	b.WriteString(`TD=$(mktemp -d); unzip -qo "$TMP" -d "$TD"; F=$(find "$TD" -type f -name loggie | head -1); `)
	b.WriteString(`install -m 755 "$F" "$BIN"; rm -rf "$TD" "$TMP"; `)
	b.WriteString(`else install -m 755 "$TMP" "$BIN"; rm -f "$TMP"; fi; `)
	b.WriteString(`chmod +x "$BIN" "$DEPLOY/start.sh" "$DEPLOY/heartbeat.sh" 2>/dev/null || true; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT"; sudo systemctl daemon-reload; sudo systemctl enable "$UNIT"; sudo systemctl restart "$UNIT"; `)
	b.WriteString(`echo "installed: $BIN"; ls -la "$DEPLOY"; systemctl is-active "$UNIT" || true`)
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
