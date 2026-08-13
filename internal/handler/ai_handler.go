package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	svc *aisvc.Service
}

func NewAIHandler(svc *aisvc.Service) *AIHandler {
	return &AIHandler{svc: svc}
}

func (h *AIHandler) Status(c *gin.Context) {
	response.Success(c, h.svc.Status(c.Request.Context()))
}

func (h *AIHandler) Ping(c *gin.Context) {
	var req aisvc.PingRequest
	_ = c.ShouldBindJSON(&req)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.Ping(c.Request.Context(), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) Chat(c *gin.Context) {
	var req aisvc.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.Chat(c.Request.Context(), uid, user, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ListSessions(c *gin.Context) {
	var q aisvc.SessionListQuery
	_ = c.ShouldBindQuery(&q)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.ListSessions(c.Request.Context(), uid, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) CreateSession(c *gin.Context) {
	var req aisvc.SessionCreateRequest
	_ = c.ShouldBindJSON(&req)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.CreateSession(c.Request.Context(), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) GetSession(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.GetSession(c.Request.Context(), uid, uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) UpdateSession(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.SessionUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.UpdateSession(c.Request.Context(), uid, uri.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) DeleteSession(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	if err := h.svc.DeleteSession(c.Request.Context(), uid, uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ClearSession(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	if err := h.svc.ClearSessionMessages(c.Request.Context(), uid, uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
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
	res, err := h.svc.AnalyzePodDiagnose(c.Request.Context(), uid, req)
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
	res, err := h.svc.AnalyzeCicdBuildFail(c.Request.Context(), uid, user, req)
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
	res, err := h.svc.AnalyzeAlertExplain(c.Request.Context(), uid, req)
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
	res, err := h.svc.ListApprovals(c.Request.Context(), user, q)
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
	res, err := h.svc.ReviewApproval(c.Request.Context(), user, uri.ID, req)
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
	res, err := h.svc.ExecuteApproval(c.Request.Context(), user, uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) SyncKnowledge(c *gin.Context) {
	n, err := h.svc.SyncKnowledgeBase(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, map[string]any{"indexed": n})
}
