package handler

import (
	"context"
	"fmt"
	"time"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	svc       *service.ProjectMgmtService
	logSearch *service.LogSearchService
	logIntel  *service.LogIntelligenceService
}

// NewProjectHandler 创建相关逻辑。
func NewProjectHandler(svc *service.ProjectMgmtService, logSearch *service.LogSearchService, logIntel *service.LogIntelligenceService) *ProjectHandler {
	return &ProjectHandler{svc: svc, logSearch: logSearch, logIntel: logIntel}
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

// Archive 归档项目。
func (h *ProjectHandler) Archive(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.ArchiveProject(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

// Restore 恢复已归档项目。
func (h *ProjectHandler) Restore(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.RestoreProject(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
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
		res, err := h.svc.ListLogSources(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"list":      res.List,
			"total":     res.Total,
			"page":      res.Page,
			"page_size": res.PageSize,
		}, nil
	})
}

// UpsertLogSource 处理对应的 HTTP 请求并返回统一响应。
func (h *ProjectHandler) UpsertLogSource(c *gin.Context) {
	ServeJSON(c, func(ctx context.Context, req service.LogSourceUpsertRequest) (service.LogSourceItem, error) {
		it, err := h.svc.UpsertLogSource(ctx, req)
		if err != nil {
			return service.LogSourceItem{}, err
		}
		return *it, nil
	})
}

// DeleteLogSource 删除对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) DeleteLogSource(c *gin.Context) {
	id, err := parseUintParam(c, "logSourceId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteLogSource(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// SearchLogs 经 Elasticsearch 检索项目日志（Loggie 采集写入 ES）。
func (h *ProjectHandler) SearchLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogSearchQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if err := h.svc.ValidateLogSearchFilters(c.Request.Context(), projectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}
	if h.logSearch == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未配置"))
		return
	}
	res, err := h.logSearch.Search(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"list":      res.List,
		"total":     res.Total,
		"page":      res.Page,
		"page_size": res.PageSize,
	})
}

// ExportLogs 导出对应的 HTTP 接口处理逻辑。
func (h *ProjectHandler) ExportLogs(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogSearchQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if err := h.svc.ValidateLogSearchFilters(c.Request.Context(), projectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}
	if h.logSearch == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未配置"))
		return
	}
	text, err := h.logSearch.Export(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := fmt.Sprintf("project-%d-logs-page-%s.txt", projectID, time.Now().Format("20060102-150405"))
	exportutil.ServeBytes(c, filename, "text/plain; charset=utf-8", []byte(text))
}

// LogOverview 日志概览（直方图 + 级别 + 签名）。
func (h *ProjectHandler) LogOverview(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogSearchQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if err := h.svc.ValidateLogSearchFilters(c.Request.Context(), projectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}
	if h.logSearch == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未配置"))
		return
	}
	res, err := h.logSearch.Overview(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// DiscoverLogFields 字段发现（观测云风格可观察字段）。
func (h *ProjectHandler) DiscoverLogFields(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogSearchQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if err := h.svc.ValidateLogSearchFilters(c.Request.Context(), projectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}
	if h.logSearch == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未配置"))
		return
	}
	res, err := h.logSearch.DiscoverFields(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// ListLogPatterns 日志模板聚类列表。
func (h *ProjectHandler) ListLogPatterns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogPatternListQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if h.logIntel == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("日志智能服务不可用"))
		return
	}
	res, err := h.logIntel.ListPatterns(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": res.List, "total": res.Total, "page": res.Page, "page_size": res.PageSize})
}

// ListLogAnomalies 日志异常事件列表。
func (h *ProjectHandler) ListLogAnomalies(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogAnomalyListQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if h.logIntel == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("日志智能服务不可用"))
		return
	}
	res, err := h.logIntel.ListAnomalies(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": res.List, "total": res.Total, "page": res.Page, "page_size": res.PageSize})
}

// UpdateLogAnomaly 更新错误追踪问题（状态/负责人/静默）。
func (h *ProjectHandler) UpdateLogAnomaly(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	anomalyID, err := parseUintParam(c, "anomaly_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.LogAnomalyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if h.logIntel == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("日志智能服务不可用"))
		return
	}
	res, err := h.logIntel.UpdateAnomaly(c.Request.Context(), projectID, anomalyID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// LogTopN 日志排行榜。
func (h *ProjectHandler) LogTopN(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogTopNQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if err := h.svc.ValidateLogSearchFilters(c.Request.Context(), projectID, q.ServerID, q.LogSourceID); err != nil {
		response.Error(c, err)
		return
	}
	if h.logSearch == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("Elasticsearch 未配置"))
		return
	}
	res, err := h.logSearch.TopN(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// LogContext 关联上下文（变更/告警/日志）。
func (h *ProjectHandler) LogContext(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	q, ok := bindQuery[service.LogContextQuery](c)
	if !ok {
		return
	}
	q.ProjectID = projectID
	if h.logIntel == nil {
		response.Error(c, constants.ErrBadRequestWithMsg("日志智能服务不可用"))
		return
	}
	res, err := h.logIntel.GetContext(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
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
