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
		deployDir = defaultLoggieDeployDir
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
	cmd := fmt.Sprintf("systemctl restart %s 2>&1 || systemctl reload-or-restart %s 2>&1", loggieRestartUnit, loggieRestartUnit)
	res, err := cli.Exec(ctx, cmd, 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("restart loggie exit=%d: %s", res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
}

func (s *LoggieAgentService) restartLoggieOverSSH(ctx context.Context, serverID uint) (stdout, stderr string, err error) {
	if s.aead == nil {
		return "", "", fmt.Errorf("SSH 加密密钥未配置，无法远程重启")
	}
	cli, _, err := sshserver.DialServer(ctx, s.aead, "loggie.restart", s.serverRepo, serverID)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()
	res, err := cli.Exec(ctx, fmt.Sprintf("systemctl restart %s", loggieRestartUnit), 8192)
	if err != nil {
		return res.Stdout, res.Stderr, err
	}
	if res.ExitCode != 0 {
		return res.Stdout, res.Stderr, fmt.Errorf("restart loggie exit=%d: %s", res.ExitCode, strings.TrimSpace(res.Stdout+res.Stderr))
	}
	return res.Stdout, res.Stderr, nil
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
