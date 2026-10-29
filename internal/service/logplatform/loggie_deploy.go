package logplatform

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"yunshu/internal/pkg/sshserver"
)

const loggieRestartUnit = "loggie.service"
const loggieBinaryName = "loggie"
const startScriptFilename = "start.sh"
const heartbeatTimerUnit = "loggie-heartbeat.timer"
const heartbeatOneshotUnit = "loggie-heartbeat.service"
const defaultLoggieRunUser = "loggie"
const defaultLoggieRunGroup = "loggie"
const loggieDataDir = "/data/loggie"

var loggieRunIdentityPattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

// validateLoggieRunIdentity 校验运行用户/组名：该值会写进 systemd 单元与远端 shell 脚本，
// 必须在 SSH 下发前按 POSIX 账户名收窄字符集。
func validateLoggieRunIdentity(user, group string) error {
	for _, item := range []struct {
		label string
		value string
	}{
		{"运行用户", strings.TrimSpace(user)},
		{"运行组", strings.TrimSpace(group)},
	} {
		if item.value == "" {
			continue
		}
		if !loggieRunIdentityPattern.MatchString(item.value) {
			return fmt.Errorf("%s只能包含小写字母、数字、下划线和短横线，以字母或下划线开头，最长 32 字符: %q", item.label, item.value)
		}
	}
	return nil
}

// resolveLoggieRunIdentity 归一化运行身份：组缺省跟随用户，两者都缺省时回到内置默认值。
func resolveLoggieRunIdentity(user, group string) (string, string) {
	u := strings.TrimSpace(user)
	g := strings.TrimSpace(group)
	if u == "" {
		u = defaultLoggieRunUser
	}
	if g == "" {
		g = u
	}
	return u, g
}

var allowedLoggieDeployRoots = []string{
	"/export/",
	"/opt/",
	"/data/",
	"/app/",
	"/srv/",
	"/usr/local/",
	"/var/lib/",
	"/home/",
	"/logs/",
	"/workspace/",
	"/tmp/",
}

// validateLoggieDeployDir 限制部署目录必须为绝对路径且落在白名单根目录的子目录下。
// 该值会进入 systemd 单元并参与远端 sudo rm -rf，必须在 SSH 下发前校验。
func validateLoggieDeployDir(dir string) error {
	d := strings.TrimSpace(dir)
	if d == "" {
		return nil
	}
	if len(d) > 200 {
		return fmt.Errorf("部署目录过长（最多 200 字符）")
	}
	if strings.ContainsAny(d, "\r\n\x00") {
		return fmt.Errorf("部署目录包含非法控制字符")
	}
	if !strings.HasPrefix(d, "/") {
		return fmt.Errorf("部署目录必须为绝对路径")
	}
	if strings.Contains(d, "//") {
		return fmt.Errorf("部署目录包含重复斜杠")
	}
	cleaned := path.Clean(d)
	if cleaned != strings.TrimRight(d, "/") {
		return fmt.Errorf("部署目录不允许包含 . 或 .. 段")
	}
	for _, r := range cleaned {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '/', r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("部署目录包含非法字符: %q", r)
		}
	}
	for _, root := range allowedLoggieDeployRoots {
		if cleaned+"/" != root && strings.HasPrefix(cleaned+"/", root) {
			return nil
		}
	}
	return fmt.Errorf("部署目录必须位于允许的目录（%s）", strings.Join(allowedLoggieDeployRoots, "、"))
}

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
	if err := validateLoggieDeployDir(deployDir); err != nil {
		return "", "", err
	}
	runUser, runGroup := resolveLoggieRunIdentity(bundle.RunUser, bundle.RunGroup)
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
		{"loggie.service", []byte(renderSystemdUnit(deployDir, runUser, runGroup)), 0o644},
		{"loggie-heartbeat.service", []byte(renderHeartbeatUnit(deployDir, runUser, runGroup)), 0o644},
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
	// 是否“已安装”按单元文件是否存在判断：`systemctl cat` 对坏单元同样报错，
	// 用它判断会导致损坏的 unit 永远无法通过热更修复，只能反复提示去点安装。
	cmd := fmt.Sprintf(
		`set +e; DEPLOY=%q; UNIT=%q; `+
			`chmod +x "$DEPLOY/heartbeat.sh" "$DEPLOY/start.sh" 2>/dev/null; `+
			`if [ -f "/etc/systemd/system/$UNIT" ]; then `+
			`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT" 2>/dev/null; `+
			`sudo cp "$DEPLOY/loggie-heartbeat.service" /etc/systemd/system/loggie-heartbeat.service 2>/dev/null; `+
			`sudo cp "$DEPLOY/loggie-heartbeat.timer" /etc/systemd/system/loggie-heartbeat.timer 2>/dev/null; `+
			shellServiceUserProvision(runUser, runGroup)+
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
	if err := validateLoggieDeployDir(dir); err != nil {
		return "", "", err
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
	if err := validateLoggieDeployDir(deployDir); err != nil {
		return "", "", err
	}
	runUser, runGroup := resolveLoggieRunIdentity(bundle.RunUser, bundle.RunGroup)
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
		{"loggie.service", []byte(renderSystemdUnit(deployDir, runUser, runGroup)), 0o644},
		{"loggie-heartbeat.service", []byte(renderHeartbeatUnit(deployDir, runUser, runGroup)), 0o644},
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

	res, err = cli.Exec(installCtx, buildFinalizeInstallScript(deployDir, unit, runUser, runGroup), 65536)
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
//
// 单元文件里的路径不能用 %q 拼接：systemd 的路径类指令不剥双引号，
// WorkingDirectory="/export/loggie" 会被判为 "path is not absolute"。
// deployDir 已由 validateLoggieDeployDir 限定为 [a-zA-Z0-9/-_.]，直接 %s 即可。
func renderSystemdUnit(deployDir, runUser, runGroup string) string {
	user, group := resolveLoggieRunIdentity(runUser, runGroup)
	bin := path.Join(deployDir, loggieBinaryName)
	systemCfg := path.Join(deployDir, "pipeline.yml")
	pipelineCfg := path.Join(deployDir, pipelinesOnlyFilename)
	return fmt.Sprintf(`[Unit]
Description=Loggie log collector (Yunshu)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
Group=%s
Environment=DBUS_SESSION_BUS_ADDRESS=disabled:
WorkingDirectory=%s
ExecStart=%s -config.system=%s -config.pipeline=%s
Restart=on-failure
RestartSec=5
LimitNOFILE=65535
# 非 root 运行；CAP_DAC_READ_SEARCH 仅用于读取属主为 root 的日志文件
AmbientCapabilities=CAP_DAC_READ_SEARCH
CapabilityBoundingSet=CAP_DAC_READ_SEARCH
NoNewPrivileges=true
ProtectSystem=full
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ProtectClock=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
UMask=0027

[Install]
WantedBy=multi-user.target
`, user, group, deployDir, bin, systemCfg, pipelineCfg)
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

func renderHeartbeatUnit(deployDir, runUser, runGroup string) string {
	user, group := resolveLoggieRunIdentity(runUser, runGroup)
	return fmt.Sprintf(`[Unit]
Description=Yunshu Loggie heartbeat (oneshot)
After=network-online.target

[Service]
Type=oneshot
User=%s
Group=%s
WorkingDirectory=%s
EnvironmentFile=-%s
ExecStart=/bin/bash %s
Nice=10
NoNewPrivileges=true
ProtectSystem=full
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
LockPersonality=true
`, user, group, deployDir, path.Join(deployDir, "loggie-heartbeat.env"), path.Join(deployDir, "heartbeat.sh"))
}

// shellServiceUserProvision 创建运行用户/组并准备数据目录；
// 创建失败时回退为 root 运行，避免升级后 Agent 直接起不来。
func shellServiceUserProvision(runUser, runGroup string) string {
	user, group := resolveLoggieRunIdentity(runUser, runGroup)
	var b strings.Builder
	fmt.Fprintf(&b, "RUN_USER=%q; RUN_GROUP=%q; DATA_DIR=%q; ", user, group, loggieDataDir)
	// 部分精简镜像没有 getent，组存在性判断回退到 /etc/group
	b.WriteString(`group_exists() { getent group "$1" >/dev/null 2>&1 || grep -q "^$1:" /etc/group 2>/dev/null; }; `)
	// 先建组：Debian 的 useradd --system 不会自动建同名组，缺组时 Group= 会让单元起不来。
	b.WriteString(`if ! group_exists "$RUN_GROUP"; then `)
	b.WriteString(`sudo groupadd -r "$RUN_GROUP" >/dev/null 2>&1 || sudo addgroup --system "$RUN_GROUP" >/dev/null 2>&1 || true; fi; `)
	b.WriteString(`if ! id -u "$RUN_USER" >/dev/null 2>&1; then `)
	b.WriteString(`sudo useradd --system --no-create-home --shell /usr/sbin/nologin --gid "$RUN_GROUP" "$RUN_USER" >/dev/null 2>&1 || `)
	b.WriteString(`sudo adduser --system --no-create-home --shell /usr/sbin/nologin --ingroup "$RUN_GROUP" "$RUN_USER" >/dev/null 2>&1 || true; fi; `)
	b.WriteString(`if id -u "$RUN_USER" >/dev/null 2>&1 && group_exists "$RUN_GROUP"; then `)
	b.WriteString(`sudo install -d -o "$RUN_USER" -g "$RUN_GROUP" -m 0750 "$DATA_DIR" 2>/dev/null || true; `)
	b.WriteString(`sudo chown -R "$RUN_USER:$RUN_GROUP" "$DATA_DIR" 2>/dev/null || true; `)
	// Loggie 的 registry（sqlite）落在 <部署目录>/data，属主不对会只读失败。
	b.WriteString(`sudo install -d -o "$RUN_USER" -g "$RUN_GROUP" -m 0750 "$DEPLOY/data" 2>/dev/null || true; `)
	b.WriteString(`sudo chown -R "$RUN_USER:$RUN_GROUP" "$DEPLOY/data" 2>/dev/null || true; `)
	b.WriteString(`sudo chmod -R a+rX "$DEPLOY" 2>/dev/null || true; `)
	b.WriteString(`else echo "warn: user $RUN_USER or group $RUN_GROUP unavailable; falling back to root"; `)
	b.WriteString(`sudo sed -i "/^User=$RUN_USER$/d;/^Group=$RUN_GROUP$/d;/^AmbientCapabilities=/d;/^CapabilityBoundingSet=/d" "/etc/systemd/system/$UNIT" /etc/systemd/system/loggie-heartbeat.service 2>/dev/null || true; fi; `)
	return b.String()
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

func buildFinalizeInstallScript(deployDir, unit, runUser, runGroup string) string {
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
	b.WriteString(`aarch64|arm64) echo "$INFO" | grep -qiE 'aarch64|ARM64' || { echo "arch mismatch: host arm64, binary=$INFO"; exit 1; } ;; `)
	b.WriteString(`*) echo "unsupported arch: $ARCH"; exit 1 ;; esac; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie.service" "/etc/systemd/system/$UNIT"; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie-heartbeat.service" /etc/systemd/system/loggie-heartbeat.service; `)
	b.WriteString(`sudo cp "$DEPLOY/loggie-heartbeat.timer" /etc/systemd/system/loggie-heartbeat.timer; `)
	b.WriteString(shellServiceUserProvision(runUser, runGroup))
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
	b.WriteString(`case "$DEPLOY" in /*/*) ;; *) echo "refuse unsafe deploy dir: $DEPLOY"; exit 1 ;; esac; `)
	b.WriteString(`case "$DEPLOY" in `)
	b.WriteString(`""|"/"|"/bin"|"/boot"|"/data"|"/dev"|"/etc"|"/home"|"/lib"|"/lib32"|"/lib64"|"/logs"|`)
	b.WriteString(`"/opt"|"/proc"|"/root"|"/run"|"/sbin"|"/srv"|"/sys"|"/tmp"|"/usr"|"/var"|"/workspace"|"/app"|"/export"|"/usr/local") `)
	b.WriteString(`echo "refuse unsafe deploy dir: $DEPLOY"; exit 1 ;; esac; `)
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
