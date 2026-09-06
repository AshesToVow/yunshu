package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/repository"
	"yunshu/internal/service/alert"
	cmdbsvc "yunshu/internal/service/cmdb"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
	projectsvc "yunshu/internal/service/project"
)

func (s *Service) platformToolDefinitions() []llm.ToolDefinition {
	return []llm.ToolDefinition{
		llm.NewFunctionTool("list_servers",
			"列出项目 CMDB 服务器（只读）。需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "必填"},
					"keyword":    map[string]any{"type": "string"},
					"page_size":  map[string]any{"type": "integer", "description": "默认 20，最大 50"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("get_server",
			"获取单台服务器详情（凭据仅摘要，无明文）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"server_id": map[string]any{"type": "integer"},
				},
				"required": []string{"server_id"},
			}),
		llm.NewFunctionTool("test_server_connectivity",
			"探测 CMDB 服务器连通性（TCP/云实例状态，只读）。主机宕机/网络不通时先 list_servers 再对本机探测；不是 SSH 执行命令。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"server_id":  map[string]any{"type": "integer", "description": "单机探测"},
					"project_id": map[string]any{"type": "integer", "description": "与 server_ids 组合做批量探测"},
					"server_ids": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "可选，批量；空则探测项目下部分主机"},
				},
			}),
		llm.NewFunctionTool("probe_server_metrics",
			"经 SSH 在远端 CMDB 服务器上只读探测磁盘/内存/负载（白名单命令）。需要 project_id+server_id 与 exec 权限。查业务机磁盘用此工具，不要用 linux.disk.check（那是 AI 容器本机）。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"server_id":  map[string]any{"type": "integer"},
					"kind":       map[string]any{"type": "string", "description": "disk|mem|load|all，默认 all"},
					"path":       map[string]any{"type": "string", "description": "disk 路径，默认 /"},
				},
				"required": []string{"server_id"},
			}),
		llm.NewFunctionTool("list_change_events",
			"查询项目变更时间线（发布/K8s/DB/静默/SSH 探测等）。排查告警前建议看告警前后窗口变更。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"source":     map[string]any{"type": "string", "description": "cicd|k8s|dbmgmt|cmdb|alert"},
					"keyword":    map[string]any{"type": "string"},
					"from":       map[string]any{"type": "string", "description": "RFC3339"},
					"to":         map[string]any{"type": "string", "description": "RFC3339"},
					"page_size":  map[string]any{"type": "integer"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("list_db_instances",
			"列出项目数据库实例（只读，无密码）。需要 project_id。",
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer"},
					"keyword":    map[string]any{"type": "string"},
					"env":        map[string]any{"type": "string"},
					"page_size":  map[string]any{"type": "integer"},
				},
				"required": []string{"project_id"},
			}),
		llm.NewFunctionTool("list_es_connections",
			"列出 ES 管理连接（只读，不含密码）。",
			map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}),
	}
}

func (s *Service) executePlatformTool(
	ctx context.Context,
	name string,
	args map[string]any,
	getUint func(string, uint) uint,
	getStr func(string) string,
	projectID uint,
	actor *auth.CurrentUser,
	requireProject, requireActor func() error,
) (any, error) {
	switch name {
	case "list_servers":
		if err := requireProject(); err != nil {
			return nil, err
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		if s.cmdbSvc != nil {
			return s.cmdbSvc.ListServers(ctx, cmdbsvc.ServerListQuery{
				ProjectID: projectID,
				Keyword:   getStr("keyword"),
				Page:      1,
				PageSize:  ps,
				Actor:     actor,
			})
		}
		return s.listServersViaRepo(ctx, projectID, getStr("keyword"), ps)
	case "get_server":
		if err := requireActor(); err != nil {
			return nil, err
		}
		sid := getUint("server_id", 0)
		if sid == 0 {
			return nil, fmt.Errorf("server_id 必填")
		}
		if s.cmdbSvc != nil {
			sv, err := s.cmdbSvc.GetServer(ctx, sid)
			if err != nil {
				return nil, err
			}
			if sv != nil && sv.ProjectID > 0 {
				if err := s.assertProjectMember(ctx, actor, sv.ProjectID); err != nil {
					return nil, err
				}
			}
			return sv, nil
		}
		if s.serverRepo == nil {
			return nil, fmt.Errorf("CMDB 服务不可用")
		}
		sv, err := s.serverRepo.GetByID(ctx, sid)
		if err != nil {
			return nil, err
		}
		if sv.ProjectID > 0 {
			if err := s.assertProjectMember(ctx, actor, sv.ProjectID); err != nil {
				return nil, err
			}
		}
		return map[string]any{
			"id":         sv.ID,
			"project_id": sv.ProjectID,
			"name":       sv.Name,
			"host":       sv.Host,
			"port":       sv.Port,
			"os_type":    sv.OSType,
			"status":     sv.Status,
			"tags":       sv.Tags,
		}, nil
	case "test_server_connectivity":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.cmdbSvc == nil {
			return nil, fmt.Errorf("CMDB 服务不可用")
		}
		sid := getUint("server_id", 0)
		if sid > 0 {
			sv, err := s.cmdbSvc.GetServer(ctx, sid)
			if err != nil {
				return nil, err
			}
			if sv != nil && sv.ProjectID > 0 {
				if err := s.assertProjectMember(ctx, actor, sv.ProjectID); err != nil {
					return nil, err
				}
			}
			return s.cmdbSvc.TestServerConnectivity(ctx, cmdbsvc.ServerTestRequest{ServerID: sid})
		}
		pid := getUint("project_id", projectID)
		if pid == 0 {
			return nil, fmt.Errorf("请提供 server_id，或 project_id（可带 server_ids）")
		}
		if err := s.assertProjectMember(ctx, actor, pid); err != nil {
			return nil, err
		}
		return s.cmdbSvc.BatchTestServerConnectivity(ctx, cmdbsvc.BatchServerTestRequest{
			ProjectID: pid,
			ServerIDs: parseUintSlice(args["server_ids"]),
			Parallel:  5,
		})
	case "probe_server_metrics":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.cmdbSvc == nil {
			return nil, fmt.Errorf("CMDB 服务不可用")
		}
		sid := getUint("server_id", 0)
		if sid == 0 {
			return nil, fmt.Errorf("server_id 必填")
		}
		pid := getUint("project_id", projectID)
		if pid == 0 {
			sv, err := s.cmdbSvc.GetServer(ctx, sid)
			if err != nil {
				return nil, err
			}
			if sv != nil {
				pid = sv.ProjectID
			}
		}
		if pid == 0 {
			return nil, fmt.Errorf("project_id 必填")
		}
		if err := s.assertProjectMember(ctx, actor, pid); err != nil {
			return nil, err
		}
		kind := getStr("kind")
		if kind == "" {
			kind = "all"
		}
		return s.cmdbSvc.ProbeHostMetrics(ctx, cmdbsvc.HostProbeRequest{
			ProjectID: pid,
			ServerID:  sid,
			Kind:      cmdbsvc.HostProbeKind(kind),
			Path:      getStr("path"),
			Actor:     actor,
		})
	case "list_change_events":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.changeEventSvc == nil {
			return nil, fmt.Errorf("变更时间线服务不可用")
		}
		ps := int(getUint("page_size", 30))
		if ps <= 0 || ps > 100 {
			ps = 30
		}
		return s.changeEventSvc.List(ctx, projectsvc.ChangeEventListQuery{
			ProjectID: projectID,
			Source:    getStr("source"),
			Keyword:   getStr("keyword"),
			From:      getStr("from"),
			To:        getStr("to"),
			Page:      1,
			PageSize:  ps,
		})
	case "list_db_instances":
		if err := requireProject(); err != nil {
			return nil, err
		}
		if s.dbmgmtSvc == nil {
			return nil, fmt.Errorf("数据库管理服务不可用")
		}
		ps := int(getUint("page_size", 20))
		if ps <= 0 || ps > 50 {
			ps = 20
		}
		return s.dbmgmtSvc.ListInstances(ctx, dbmgmtsvc.InstanceListQuery{
			ProjectID: projectID,
			Env:       getStr("env"),
			Keyword:   getStr("keyword"),
			Page:      1,
			PageSize:  ps,
		})
	case "list_es_connections":
		if err := requireActor(); err != nil {
			return nil, err
		}
		if s.esmgmtSvc == nil {
			return nil, fmt.Errorf("ES 管理服务不可用")
		}
		list, err := s.esmgmtSvc.ListConnectionsForSelect(ctx)
		if err != nil {
			return nil, err
		}
		type item struct {
			ID          uint   `json:"id"`
			Name        string `json:"name"`
			Addresses   string `json:"addresses"`
			Username    string `json:"username"`
			HasPassword bool   `json:"has_password"`
			TimeoutSec  int    `json:"timeout_sec"`
			IsDefault   bool   `json:"is_default"`
			Remark      string `json:"remark"`
		}
		out := make([]item, 0, len(list))
		for _, c := range list {
			out = append(out, item{
				ID:          c.ID,
				Name:        c.Name,
				Addresses:   c.Addresses,
				Username:    c.Username,
				HasPassword: c.HasPassword,
				TimeoutSec:  c.TimeoutSec,
				IsDefault:   c.IsDefault,
				Remark:      c.Remark,
			})
		}
		return map[string]any{"total": len(out), "list": out}, nil
	default:
		return nil, fmt.Errorf("未知平台工具: %s", name)
	}
}

func parseUintSlice(v any) []uint {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	out := make([]uint, 0, len(arr))
	for _, it := range arr {
		switch n := it.(type) {
		case float64:
			if n > 0 {
				out = append(out, uint(n))
			}
		case int:
			if n > 0 {
				out = append(out, uint(n))
			}
		case int64:
			if n > 0 {
				out = append(out, uint(n))
			}
		case json.Number:
			i, err := n.Int64()
			if err == nil && i > 0 {
				out = append(out, uint(i))
			}
		}
	}
	return out
}

func (s *Service) listServersViaRepo(ctx context.Context, projectID uint, keyword string, pageSize int) (any, error) {
	if s.serverRepo == nil {
		return nil, fmt.Errorf("CMDB 服务不可用")
	}
	list, total, err := s.serverRepo.List(ctx, repository.ServerListParams{
		ProjectID: projectID,
		Keyword:   keyword,
		Page:      1,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	type item struct {
		ID     uint   `json:"id"`
		Name   string `json:"name"`
		Host   string `json:"host"`
		Port   int    `json:"port"`
		OSType string `json:"os_type"`
		Status int    `json:"status"`
		Tags   string `json:"tags"`
	}
	out := make([]item, 0, len(list))
	for _, sv := range list {
		out = append(out, item{
			ID: sv.ID, Name: sv.Name, Host: sv.Host, Port: sv.Port,
			OSType: sv.OSType, Status: sv.Status, Tags: sv.Tags,
		})
	}
	return map[string]any{"total": total, "list": out, "page": 1, "page_size": pageSize}, nil
}

func (s *Service) executeCreateAlertSilence(
	ctx context.Context,
	actor *auth.CurrentUser,
	projectID uint,
	getUint func(string, uint) uint,
	getStr func(string) string,
) (any, error) {
	if s.silenceSvc == nil {
		return nil, fmt.Errorf("静默服务不可用")
	}
	fp := strings.TrimSpace(getStr("fingerprint"))
	if fp == "" {
		return nil, fmt.Errorf("fingerprint 必填")
	}
	pid := getUint("project_id", projectID)
	if pid == 0 {
		return nil, fmt.Errorf("project_id 必填")
	}
	if err := s.assertProjectMember(ctx, actor, pid); err != nil {
		return nil, err
	}
	hours := int(getUint("hours", 2))
	if hours <= 0 {
		hours = 2
	}
	if hours > 72 {
		hours = 72
	}
	alertname := strings.TrimSpace(getStr("alertname"))
	if alertname == "" {
		alertname = fp
	}
	comment := strings.TrimSpace(getStr("comment"))
	if comment == "" {
		comment = "AI 助手告警闭环静默"
	}
	matchers := []map[string]any{{
		"name": "fingerprint", "value": fp, "is_regex": false,
	}}
	raw, _ := json.Marshal(matchers)
	now := time.Now()
	ends := now.Add(time.Duration(hours) * time.Hour)
	en := true
	uid := uint(0)
	if actor != nil {
		uid = actor.ID
	}
	row, err := s.silenceSvc.Create(ctx, uid, alert.AlertSilenceUpsertRequest{
		ProjectID:    pid,
		Name:         fmt.Sprintf("AI静默 %s（%dh）", truncateStr(alertname, 64), hours),
		MatchersJSON: string(raw),
		StartsAt:     now,
		EndsAt:       ends,
		Comment:      comment,
		Enabled:      &en,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"ok": true, "silence_id": row.ID, "ends_at": ends, "hours": hours,
		"note": "静默已生效；告警监控 → 降噪·静默 可管理",
	}, nil
}
