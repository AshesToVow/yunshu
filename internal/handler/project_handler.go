package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	pb "yunshu/internal/grpc/proto"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	svc             *service.ProjectMgmtService
	projectClient   pb.ProjectServerServiceClient
	logSourceClient pb.LogSourceServiceClient
}

// NewProjectHandler 创建相关逻辑。
func NewProjectHandler(svc *service.ProjectMgmtService, projectClient pb.ProjectServerServiceClient, logSourceClient pb.LogSourceServiceClient) *ProjectHandler {
	return &ProjectHandler{svc: svc, projectClient: projectClient, logSourceClient: logSourceClient}
}

// List 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) List(c *gin.Context) {
	ServeQuery(c, h.svc.ListProjects)
}

// Create 创建对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) Create(c *gin.Context) {
	u, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ProjectCreateRequest) (*service.ProjectItem, error) {
		return h.svc.CreateProject(ctx, u.ID, req)
	})
}

// Update 更新对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ProjectUpdateRequest) (*service.ProjectItem, error) {
		return h.svc.UpdateProject(ctx, id, req)
	})
}

// ApplicationTopology 返回项目应用拓扑图（服务/服务器/日志源）。
func (h *ProjectHandler) ApplicationTopology(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, _ struct{}) (*service.ApplicationTopologyGraph, error) {
		return h.svc.ApplicationTopology(ctx, id)
	})
}

// Delete 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteProject(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListServices 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListServices(c *gin.Context) {
	ServeQuery(c, h.svc.ListServices)
}

// UpsertService 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertService(c *gin.Context) {
	ServeJSON(c, h.svc.UpsertService)
}

// DeleteService 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteService(c *gin.Context) {
	id, err := parseUintParam(c, "serviceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteService(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListLogSources 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogSources(c *gin.Context) {
	ServeQuery(c, func(ctx context.Context, q service.LogSourceListQuery) (gin.H, error) {
		req := &pb.ListLogSourcesRequest{
			ProjectId: uint64(q.ProjectID),
			Page:      &pb.PageRequest{Page: int32(q.Page), PageSize: int32(q.PageSize)},
		}
		if q.ServiceID != nil {
			req.HasServiceId = true
			req.ServiceId = uint64(*q.ServiceID)
		}
		resp, err := h.logSourceClient.ListLogSources(ctx, req)
		if err != nil {
			return nil, grpcToAppError(err)
		}
		out := make([]service.LogSourceItem, 0, len(resp.GetList()))
		for _, it := range resp.GetList() {
			out = append(out, service.LogSourceItem{
				ID:            uint(it.GetId()),
				ServiceID:     uint(it.GetServiceId()),
				LogType:       it.GetLogType(),
				Path:          it.GetPath(),
				Encoding:      stringPtr(it.GetEncoding()),
				Timezone:      stringPtr(it.GetTimezone()),
				MultilineRule: stringPtr(it.GetMultilineRule()),
				IncludeRegex:  stringPtr(it.GetIncludeRegex()),
				ExcludeRegex:  stringPtr(it.GetExcludeRegex()),
				Status:        int(it.GetStatus()),
				CreatedAt:     it.GetCreatedAt(),
			})
		}
		return gin.H{
			"list":      out,
			"total":     resp.GetPage().GetTotal(),
			"page":      resp.GetPage().GetPage(),
			"page_size": resp.GetPage().GetPageSize(),
		}, nil
	})
}

// UpsertLogSource 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertLogSource(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.LogSourceUpsertRequest) (service.LogSourceItem, error) {
		r := &pb.UpsertLogSourceRequest{
			ServiceId: uint64(req.ServiceID),
			LogType:   req.LogType,
			Path:      req.Path,
			Status:    int32(req.Status),
		}
		if req.ID != nil {
			r.Id = uint64(*req.ID)
		}
		if req.Encoding != nil {
			r.Encoding = *req.Encoding
		}
		if req.Timezone != nil {
			r.Timezone = *req.Timezone
		}
		if req.MultilineRule != nil {
			r.MultilineRule = *req.MultilineRule
		}
		if req.IncludeRegex != nil {
			r.IncludeRegex = *req.IncludeRegex
		}
		if req.ExcludeRegex != nil {
			r.ExcludeRegex = *req.ExcludeRegex
		}
		it, err := h.logSourceClient.UpsertLogSource(ctx, r)
		if err != nil {
			return service.LogSourceItem{}, grpcToAppError(err)
		}
		return service.LogSourceItem{
			ID:            uint(it.GetId()),
			ServiceID:     uint(it.GetServiceId()),
			LogType:       it.GetLogType(),
			Path:          it.GetPath(),
			Encoding:      stringPtr(it.GetEncoding()),
			Timezone:      stringPtr(it.GetTimezone()),
			MultilineRule: stringPtr(it.GetMultilineRule()),
			IncludeRegex:  stringPtr(it.GetIncludeRegex()),
			ExcludeRegex:  stringPtr(it.GetExcludeRegex()),
			Status:        int(it.GetStatus()),
			CreatedAt:     it.GetCreatedAt(),
		}, nil
	})
}

// DeleteLogSource 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteLogSource(c *gin.Context) {
	id, err := parseUintParam(c, "logSourceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if _, err := h.logSourceClient.DeleteLogSource(c.Request.Context(), &pb.DeleteLogSourceRequest{Id: uint64(id)}); err != nil {
		response.Error(c, grpcToAppError(err))
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// StreamLogs 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) StreamLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	q, ok := bindQuery[service.LogStreamQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID

	if err := h.svc.ValidateLogSourceAccess(c.Request.Context(), q.ProjectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}

	streamKey := service.BuildLogStreamKey(q.ProjectID, q.ServerID, q.LogSourceID)
	var includeRe *regexp.Regexp
	if q.Include != nil && strings.TrimSpace(*q.Include) != "" {
		re, err := regexp.Compile(*q.Include)
		if err != nil {
			response.Error(c, constants.ErrIncludeRegexInvalid)
			return
		}
		includeRe = re
	}
	var excludeRe *regexp.Regexp
	if q.Exclude != nil && strings.TrimSpace(*q.Exclude) != "" {
		re, err := regexp.Compile(*q.Exclude)
		if err != nil {
			response.Error(c, constants.ErrExcludeRegexInvalid)
			return
		}
		excludeRe = re
	}
	hl := ""
	if q.Highlight != nil {
		hl = strings.TrimSpace(*q.Highlight)
	}
	targetFilePath := ""
	if q.FilePath != nil {
		targetFilePath = strings.TrimSpace(*q.FilePath)
	}
	replayLines := q.TailLines
	if replayLines <= 0 {
		replayLines = 200
	}

	ch, cancelSub := service.AgentLogBroker.Subscribe(streamKey, replayLines, q.AfterID)
	defer cancelSub()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// SSE：须在参数校验通过后再写头，避免非法请求返回 JSON 却已发送 event-stream。
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			_, _ = c.Writer.WriteString("event: ping\ndata: {}\n\n")
			c.Writer.Flush()
		case event := <-ch:
			if targetFilePath != "" && strings.TrimSpace(event.FilePath) != targetFilePath {
				continue
			}
			line := event.Line
			if includeRe != nil && !includeRe.MatchString(line) {
				continue
			}
			if excludeRe != nil && excludeRe.MatchString(line) {
				continue
			}
			if hl != "" && strings.Contains(line, hl) {
				line = strings.ReplaceAll(line, hl, "\x1b[31m"+hl+"\x1b[0m")
			}
			if len(line) > 4096 {
				line = line[:4096] + " ...<truncated>"
			}
			c.SSEvent("log", gin.H{
				"id":        event.ID,
				"line":      line,
				"file_path": strings.TrimSpace(event.FilePath),
			})
			c.Writer.Flush()
		}
	}
}

// ExportLogs 导出对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ExportLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogExportQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	data, filename, err := h.svc.ExportLogs(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", data)
}

// ListLogFiles 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogFiles(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.RemoteLogFileQuery) (gin.H, error) {
		q.ProjectID = projectID
		files, err := h.svc.ListRemoteLogFiles(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": files}, nil
	})
}

// ListLogUnits 查询列表对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ListLogUnits(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.RemoteLogUnitQuery) (gin.H, error) {
		q.ProjectID = projectID
		list, err := h.svc.ListRemoteLogUnits(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{"list": list}, nil
	})
}

// ListProjectMembers 项目成员列表。
func (h *ProjectHandler) ListProjectMembers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListProjectMembers(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// AddProjectMember 添加项目成员。
func (h *ProjectHandler) AddProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ProjectMemberAddRequest) (*service.ProjectMemberItem, error) {
		return h.svc.AddProjectMember(ctx, projectID, req)
	})
}

// UpdateProjectMember 更新成员角色。
func (h *ProjectHandler) UpdateProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	memberID, err := parseUintParam(c, "memberId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ProjectMemberUpdateRequest) (*service.ProjectMemberItem, error) {
		return h.svc.UpdateProjectMember(ctx, projectID, memberID, req)
	})
}

// RemoveProjectMember 移除项目成员。
func (h *ProjectHandler) RemoveProjectMember(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	memberID, err := parseUintParam(c, "memberId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.RemoveProjectMember(c.Request.Context(), projectID, memberID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}
