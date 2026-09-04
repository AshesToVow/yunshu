package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ClusterLogHandler struct {
	svc *service.ClusterLogService
}

func NewClusterLogHandler(svc *service.ClusterLogService) *ClusterLogHandler {
	return &ClusterLogHandler{svc: svc}
}

func actorUserID(c *gin.Context) uint {
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		return u.ID
	}
	return 0
}

func (h *ClusterLogHandler) ListRules(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	clusterID := parseOptionalUintQuery(c, "cluster_id")
	list, err := h.svc.ListRules(c.Request.Context(), projectID, clusterID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *ClusterLogHandler) CreateRule(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.ClusterLogRuleUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.CreateRule(c.Request.Context(), projectID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) UpdateRule(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ruleID, err := parseUintParam(c, "rule_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.ClusterLogRuleUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdateRule(c.Request.Context(), projectID, ruleID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) DeleteRule(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ruleID, err := parseUintParam(c, "rule_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteRule(c.Request.Context(), projectID, ruleID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *ClusterLogHandler) ListAgents(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListAgents(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *ClusterLogHandler) PreviewPipelines(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	clusterID, err := parseUintQuery(c, "cluster_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.PreviewPipelines(c.Request.Context(), projectID, clusterID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *ClusterLogHandler) SavePipelines(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.ClusterPipelinesUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.SavePipelines(c.Request.Context(), projectID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

type clusterLogDeployReq struct {
	ClusterID    uint   `json:"cluster_id"`
	Namespace    string `json:"namespace"`
	RateLimitQPS int    `json:"rate_limit_qps"`
}

func (h *ClusterLogHandler) Deploy(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req clusterLogDeployReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	agent, err := h.svc.DeployOrSync(c.Request.Context(), projectID, req.ClusterID, req.Namespace, req.RateLimitQPS)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *ClusterLogHandler) RefreshStatus(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	clusterID, err := parseUintQuery(c, "cluster_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	agent, err := h.svc.RefreshStatus(c.Request.Context(), projectID, clusterID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, agent)
}

func (h *ClusterLogHandler) ListPipelinesRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListLogPipelines(c.Request.Context(), projectID, c.Query("kind"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *ClusterLogHandler) GetPipelineRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	pid, err := parseUintParam(c, "pipeline_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.GetLogPipeline(c.Request.Context(), projectID, pid)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) CreatePipelineRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.LogPipelineUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpsertLogPipeline(c.Request.Context(), projectID, 0, actorUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) UpdatePipelineRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	pid, err := parseUintParam(c, "pipeline_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.LogPipelineUpsert
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpsertLogPipeline(c.Request.Context(), projectID, pid, actorUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) DeletePipelineRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	pid, err := parseUintParam(c, "pipeline_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteLogPipeline(c.Request.Context(), projectID, pid); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ClusterLogHandler) SyncPipelineRepoFromCluster(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		ClusterID uint `json:"cluster_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.SyncLogPipelinesFromCluster(c.Request.Context(), projectID, req.ClusterID, actorUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) ApplyPipelineRepo(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	pid, err := parseUintParam(c, "pipeline_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req service.LogPipelineApplyRequest
	_ = c.ShouldBindJSON(&req)
	row, err := h.svc.ApplyLogPipeline(c.Request.Context(), projectID, pid, actorUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *ClusterLogHandler) ListParseProfiles(c *gin.Context) {
	response.Success(c, gin.H{"list": h.svc.ListParseProfiles()})
}
