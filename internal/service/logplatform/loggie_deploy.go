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
		{bundle.EnvFilename, []byte(bundle.EnvFile), 0o644},
		{bundle.HeartbeatFilename, []byte(bundle.HeartbeatScript), 0o755},
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
	cmd := fmt.Sprintf("systemctl reload-or-restart %s 2>&1 || systemctl restart %s 2>&1", unit, unit)
	res, err := cli.Exec(ctx, cmd, 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("reload-or-restart loggie exit=%d: %s", res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
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
	binURL := strings.TrimSpace(binaryURL)
	if binURL == "" {
		binURL = strings.TrimSpace(s.loggieCfg.BinaryURL)
	}

	var out, errOut strings.Builder
	mkdir := fmt.Sprintf("mkdir -p %s/bin %s", deployDir, deployDir)
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

func renderSystemdUnit(deployDir string) string {
	return fmt.Sprintf(`[Unit]
Description=Loggie log collector (Yunshu)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
Environment=DBUS_SESSION_BUS_ADDRESS=disabled:
WorkingDirectory=%s
ExecStart=%s/bin/loggie -config.system=%s/pipeline.yml -config.pipeline=%s/pipelines.yml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`, deployDir, deployDir, deployDir, deployDir)
}

func buildInstallRemoteScript(deployDir, unit, binaryURL string) string {
	var b strings.Builder
	b.WriteString("set -e; ")
	b.WriteString(fmt.Sprintf("DEPLOY=%q; UNIT=%q; BINURL=%q; ", deployDir, unit, binaryURL))
	b.WriteString(`ARCH=$(uname -m); case "$ARCH" in x86_64|amd64) ARCH=amd64;; aarch64|arm64) ARCH=arm64;; *) echo "unsupported arch $ARCH"; exit 1;; esac; `)
	b.WriteString(`if [[ -n "$BINURL" ]]; then URL="${BINURL//\{arch\}/$ARCH}"; TMP=$(mktemp -d); cd "$TMP"; `)
	b.WriteString(`(curl -fsSL "$URL" -o pkg.bin || wget -qO pkg.bin "$URL"); `)
	b.WriteString(`if file pkg.bin 2>/dev/null | grep -qi 'gzip\|tar'; then tar -xzf pkg.bin; elif file pkg.bin 2>/dev/null | grep -qi 'Zip'; then unzip -o pkg.bin; else cp pkg.bin loggie; fi; `)
	b.WriteString(`BIN=$(find . -type f -name loggie | head -1); if [[ -z "$BIN" ]]; then BIN=./loggie; fi; install -m 755 "$BIN" "$DEPLOY/bin/loggie"; rm -rf "$TMP"; `)
	b.WriteString(`elif [[ ! -x "$DEPLOY/bin/loggie" ]]; then echo "binary_url empty and $DEPLOY/bin/loggie missing"; exit 1; fi; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT"; sudo systemctl daemon-reload; sudo systemctl enable "$UNIT"; sudo systemctl restart "$UNIT"; `)
	b.WriteString(`echo "loggie installed"; systemctl is-active "$UNIT" || true`)
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
