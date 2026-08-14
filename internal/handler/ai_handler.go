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

func (h *AIHandler) CenterOverview(c *gin.Context) {
	response.Success(c, h.svc.CenterOverview(c.Request.Context()))
}

func (h *AIHandler) ReseedCenter(c *gin.Context) {
	if err := h.svc.ReseedCenter(c.Request.Context()); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListPrompts(c *gin.Context) {
	rows, err := h.svc.ListPrompts(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListPromptVersions(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	rows, err := h.svc.ListPromptVersions(c.Request.Context(), uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) PublishPrompt(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.PromptPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	ver, err := h.svc.PublishPromptVersion(c.Request.Context(), uri.ID, uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, ver)
}

func (h *AIHandler) RollbackPrompt(c *gin.Context) {
	var uri struct {
		ID        uint `uri:"id" binding:"required"`
		VersionID uint `uri:"vid" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.RollbackPromptVersion(c.Request.Context(), uri.ID, uri.VersionID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListLLMModels(c *gin.Context) {
	rows, err := h.svc.ListLLMModels(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) CreateLLMModel(c *gin.Context) {
	var req aisvc.LLMModelUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.CreateLLMModel(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) UpdateLLMModel(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.LLMModelUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdateLLMModel(c.Request.Context(), uri.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) DeleteLLMModel(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteLLMModel(c.Request.Context(), uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) SetDefaultLLMModel(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.SetDefaultLLMModel(c.Request.Context(), uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListCenterTools(c *gin.Context) {
	rows, err := h.svc.ListTools(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) UpdateToolEnabled(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.UpdateToolEnabled(c.Request.Context(), uri.ID, req.Enabled); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListCases(c *gin.Context) {
	rows, err := h.svc.ListIncidentCases(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListSOPs(c *gin.Context) {
	rows, err := h.svc.ListSOPs(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListKBs(c *gin.Context) {
	rows, err := h.svc.ListKnowledgeBases(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListEvalCases(c *gin.Context) {
	rows, err := h.svc.ListEvalCases(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) RunEval(c *gin.Context) {
	var req struct {
		Live bool `json:"live"`
	}
	_ = c.ShouldBindJSON(&req)
	user, _ := auth.CurrentUserFromContext(c)
	run, err := h.svc.RunEvalSuite(c.Request.Context(), user, req.Live)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}
