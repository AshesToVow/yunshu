package cmdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/changeevent"
)

// HostProbeKind 远端只读探测类型。
type HostProbeKind string

const (
	HostProbeDisk HostProbeKind = "disk"
	HostProbeMem  HostProbeKind = "mem"
	HostProbeLoad HostProbeKind = "load"
	HostProbeAll  HostProbeKind = "all"
)

// HostProbeRequest 远端主机只读指标探测（经 SSH，命令白名单）。
type HostProbeRequest struct {
	ProjectID  uint          `json:"project_id"`
	ServerID   uint          `json:"server_id"`
	Kind       HostProbeKind `json:"kind"` // disk|mem|load|all
	Path       string        `json:"path"` // disk 路径，默认 /
	TimeoutSec int           `json:"timeout_sec"`
	Actor      *auth.CurrentUser
}

// HostProbeCommandResult 单条命令结果。
type HostProbeCommandResult struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// HostProbeResult 聚合探测结果。
type HostProbeResult struct {
	ServerID   uint                     `json:"server_id"`
	ServerName string                   `json:"server_name"`
	Host       string                   `json:"host"`
	Kind       string                   `json:"kind"`
	Note       string                   `json:"note"`
	Commands   []HostProbeCommandResult `json:"commands"`
	DurationMS int64                    `json:"duration_ms"`
}

var readonlyProbeAllowlist = map[string]string{
	"df_h":      "df -h",
	"df_path":   "df -h %s",
	"free_m":    "free -m",
	"uptime":    "uptime",
	"loadavg":   "cat /proc/loadavg",
	"uname":     "uname -a",
}

// ProbeHostMetrics 经 SSH 在远端执行只读资源探测（白名单命令）。
func (s *Service) ProbeHostMetrics(ctx context.Context, req HostProbeRequest) (*HostProbeResult, error) {
	if req.ServerID == 0 || req.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id 与 server_id 必填")
	}
	if err := s.AssertServerAccess(ctx, req.ProjectID, req.ServerID, req.Actor, "exec"); err != nil {
		return nil, err
	}
	sv, err := s.serverRepo.GetByID(ctx, req.ServerID)
	if err != nil {
		return nil, constants.ErrServerNotFound
	}
	if sv.ProjectID != req.ProjectID {
		return nil, constants.ErrServerNotInCurrentProject
	}

	kind := HostProbeKind(strings.ToLower(strings.TrimSpace(string(req.Kind))))
	if kind == "" {
		kind = HostProbeAll
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		path = "/"
	}
	if strings.ContainsAny(path, ";|&`$<>\n\r") || strings.Contains(path, "..") {
		return nil, constants.ErrBadRequestWithMsg("非法 path")
	}

	cmds := buildProbeCommands(kind, path)
	if len(cmds) == 0 {
		return nil, constants.ErrBadRequestWithMsg("不支持的 kind，请用 disk|mem|load|all")
	}

	started := time.Now()
	out := &HostProbeResult{
		ServerID:   sv.ID,
		ServerName: sv.Name,
		Host:       sv.Host,
		Kind:       string(kind),
		Note:       "SSH 只读探测远端主机；非 AI 容器本机脚本",
		Commands:   make([]HostProbeCommandResult, 0, len(cmds)),
	}

	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 20
	}
	for _, c := range cmds {
		res, err := s.ExecServerCommand(ctx, ServerExecRequest{
			ProjectID:  req.ProjectID,
			ServerID:   req.ServerID,
			Command:    c.command,
			TimeoutSec: timeout,
		})
		item := HostProbeCommandResult{Name: c.name, Command: c.command}
		if err != nil {
			item.Stderr = err.Error()
			item.ExitCode = -1
		} else {
			item.Stdout = res.Stdout
			item.Stderr = res.Stderr
			item.ExitCode = res.ExitCode
		}
		out.Commands = append(out.Commands, item)
	}
	out.DurationMS = time.Since(started).Milliseconds()

	status := model.ChangeStatusSucceeded
	for _, c := range out.Commands {
		if c.ExitCode != 0 {
			status = model.ChangeStatusFailed
			break
		}
	}
	changeevent.Record(ctx, changeevent.Input{
		ProjectID: req.ProjectID,
		Source:    model.ChangeSourceCmdb,
		Action:    "ssh_probe",
		RiskLevel: model.ChangeRiskLow,
		Status:    status,
		Summary:   fmt.Sprintf("SSH 只读探测 %s(%s) kind=%s", sv.Name, sv.Host, kind),
		Payload: map[string]any{
			"server_id": sv.ID, "kind": kind, "path": path, "commands": len(cmds),
		},
		StartedAt: &started,
	})
	return out, nil
}

type namedCmd struct {
	name, command string
}

func buildProbeCommands(kind HostProbeKind, path string) []namedCmd {
	dfPath := fmt.Sprintf(readonlyProbeAllowlist["df_path"], shellQuotePath(path))
	switch kind {
	case HostProbeDisk:
		return []namedCmd{{"df_path", dfPath}}
	case HostProbeMem:
		return []namedCmd{{"free_m", readonlyProbeAllowlist["free_m"]}}
	case HostProbeLoad:
		return []namedCmd{
			{"uptime", readonlyProbeAllowlist["uptime"]},
			{"loadavg", readonlyProbeAllowlist["loadavg"]},
		}
	case HostProbeAll:
		return []namedCmd{
			{"df_path", dfPath},
			{"free_m", readonlyProbeAllowlist["free_m"]},
			{"uptime", readonlyProbeAllowlist["uptime"]},
			{"loadavg", readonlyProbeAllowlist["loadavg"]},
		}
	default:
		return nil
	}
}

func shellQuotePath(p string) string {
	// path 已做过非法字符校验；用单引号包裹降低注入面
	return "'" + strings.ReplaceAll(p, "'", `'\''`) + "'"
}
