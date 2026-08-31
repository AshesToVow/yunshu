package ai

import (
	"context"
	"fmt"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/llm"
	"yunshu/internal/repository"
	cmdbsvc "yunshu/internal/service/cmdb"
	dbmgmtsvc "yunshu/internal/service/dbmgmt"
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
			return s.cmdbSvc.GetServer(ctx, sid)
		}
		if s.serverRepo == nil {
			return nil, fmt.Errorf("CMDB 服务不可用")
		}
		sv, err := s.serverRepo.GetByID(ctx, sid)
		if err != nil {
			return nil, err
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
