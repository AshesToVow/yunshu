package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

func (h *AIHandler) EmbedKnowledge(c *gin.Context) {
	rep, err := h.svc.SyncEmbeddings(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rep)
}

func (h *AIHandler) PodDiagnose(c *gin.Context) {
	var req aisvc.PodDiagnoseAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.AnalyzePodDiagnose(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) GenerateK8sYAML(c *gin.Context) {
	var req aisvc.GenerateK8sYAMLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.GenerateK8sYAML(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) CicdBuildFail(c *gin.Context) {
	var req aisvc.CicdBuildFailAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.AnalyzeCicdBuildFail(auth.RequestContext(c), uid, user, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) AlertExplain(c *gin.Context) {
	var req aisvc.AlertExplainAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.AnalyzeAlertExplain(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) LogAnalyze(c *gin.Context) {
	var req aisvc.LogAnalyzeAIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.AnalyzeLogs(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) PipelineAdjust(c *gin.Context) {
	var req aisvc.PipelineAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.AdjustLoggiePipeline(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ListApprovals(c *gin.Context) {
	var q aisvc.ApprovalListQuery
	_ = c.ShouldBindQuery(&q)
	user, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.ListApprovals(auth.RequestContext(c), user, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ReviewApproval(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.ReviewApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.ReviewApproval(auth.RequestContext(c), user, uri.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ExecuteApproval(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	res, err := h.svc.ExecuteApproval(auth.RequestContext(c), user, uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) SyncKnowledge(c *gin.Context) {
	rep, err := h.svc.SyncKnowledgeBase(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rep)
}
