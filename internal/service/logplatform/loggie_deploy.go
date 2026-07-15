package logplatform

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	// 安装含二进制下载/上传，勿跟前端断开一并取消；最长 5 分钟。
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
	binURL := s.resolveBinaryURL(binaryURL)
	binRemote := path.Join(deployDir, loggieBinaryName)

	var out, errOut strings.Builder
	mkdir := fmt.Sprintf("mkdir -p %q", deployDir)
	res, err := cli.Exec(installCtx, mkdir, 4096)
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

	// 由 Yunshu 主机拉取二进制再 SFTP 上传，避免目标机直连 GitHub 超时。
	binData, dlErr := downloadLoggieBinary(installCtx, binURL)
	if dlErr != nil {
		out.WriteString("platform download failed: " + dlErr.Error() + "\n")
		out.WriteString("fallback: remote curl (may be slow)\n")
		res, err = cli.Exec(installCtx, buildRemoteDownloadScript(deployDir, binURL), 65536)
		out.WriteString(res.Stdout)
		errOut.WriteString(res.Stderr)
		if err != nil || res.ExitCode != 0 {
			return out.String(), errOut.String(), fmt.Errorf("download binary failed: %v; remote=%s", dlErr, strings.TrimSpace(res.Stdout+res.Stderr))
		}
	} else {
		if err := cli.UploadBytes(installCtx, binRemote, binData, 0o755); err != nil {
			return out.String(), errOut.String(), fmt.Errorf("upload binary: %w", err)
		}
		out.WriteString(fmt.Sprintf("uploaded binary %s (%d bytes) from platform\n", binRemote, len(binData)))
	}

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

func downloadLoggieBinary(ctx context.Context, binaryURL string) ([]byte, error) {
	url := strings.TrimSpace(binaryURL)
	if url == "" {
		return nil, fmt.Errorf("binary_url empty")
	}
	client := &http.Client{Timeout: 4 * time.Minute}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "yunshu-loggie-install/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	// Loggie 裸二进制通常几十 MB，限制 128MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return nil, err
	}
	if len(data) < 1024 {
		return nil, fmt.Errorf("download too small (%d bytes), not a binary", len(data))
	}
	if len(data) < 4 || data[0] != 0x7f || string(data[1:4]) != "ELF" {
		return nil, fmt.Errorf("downloaded content is not ELF (wrong URL or HTML error page)")
	}
	return data, nil
}

func buildRemoteDownloadScript(deployDir, binaryURL string) string {
	var b strings.Builder
	b.WriteString("set -euo pipefail; ")
	b.WriteString(fmt.Sprintf("DEPLOY=%q; BINURL=%q; BIN=%q/loggie; ", deployDir, binaryURL, deployDir))
	b.WriteString(`ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; esac; `)
	b.WriteString(`URL="${BINURL//\{arch\}/$ARCH}"; TMP=$(mktemp); `)
	b.WriteString(`(curl -fsSL --connect-timeout 20 --max-time 240 "$URL" -o "$TMP" || wget -q --timeout=240 -O "$TMP" "$URL"); `)
	b.WriteString(`if file "$TMP" 2>/dev/null | grep -qiE 'gzip|tar archive'; then `)
	b.WriteString(`TD=$(mktemp -d); tar -xzf "$TMP" -C "$TD"; F=$(find "$TD" -type f -name loggie | head -1); install -m 755 "$F" "$BIN"; rm -rf "$TD" "$TMP"; `)
	b.WriteString(`else install -m 755 "$TMP" "$BIN"; rm -f "$TMP"; fi; `)
	b.WriteString(`chmod +x "$BIN"; ls -la "$BIN"`)
	return b.String()
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
	// 官方 GitHub release 的 loggie 资产仅为 linux/amd64
	b.WriteString(`case "$ARCH" in `)
	b.WriteString(`x86_64|amd64) echo "$INFO" | grep -qiE 'x86-64|x86_64|AMD64|Intel 80386' || { echo "arch mismatch: host amd64, binary=$INFO"; exit 1; } ;; `)
	b.WriteString(`aarch64|arm64) echo "ERROR: host is arm64; official Loggie release binary is amd64 only (Exec format error). Use amd64 host or build arm64 Loggie."; exit 1 ;; `)
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
