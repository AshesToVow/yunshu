package handler

import (
	"encoding/json"
	"fmt"

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
	response.Success(c, h.svc.Status(auth.RequestContext(c)))
}

func (h *AIHandler) Ping(c *gin.Context) {
	var req aisvc.PingRequest
	_ = c.ShouldBindJSON(&req)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.Ping(auth.RequestContext(c), uid, req)
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
	res, err := h.svc.Chat(auth.RequestContext(c), uid, user, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ChatStream(c *gin.Context) {
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
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)
	c.Writer.Flush()

	writeEvent := func(ev aisvc.ChatEvent) {
		raw, err := json.Marshal(ev)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", raw)
		c.Writer.Flush()
	}

	_, err := h.svc.ChatStream(auth.RequestContext(c), uid, user, req, writeEvent)
	if err != nil {
		writeEvent(aisvc.ChatEvent{Type: "error", Error: err.Error(), Message: err.Error()})
	}
}

func (h *AIHandler) StartInvestigation(c *gin.Context) {
	var req aisvc.StartInvestigationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.StartInvestigation(auth.RequestContext(c), uid, user, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) ListInvestigations(c *gin.Context) {
	var q aisvc.InvestigationListQuery
	_ = c.ShouldBindQuery(&q)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.ListInvestigations(auth.RequestContext(c), uid, q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) GetInvestigation(c *gin.Context) {
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
	res, err := h.svc.GetInvestigation(auth.RequestContext(c), uid, uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

func (h *AIHandler) EmbedKnowledge(c *gin.Context) {
	rep, err := h.svc.SyncEmbeddings(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rep)
}

func (h *AIHandler) ListSessions(c *gin.Context) {
	var q aisvc.SessionListQuery
	_ = c.ShouldBindQuery(&q)
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	res, err := h.svc.ListSessions(auth.RequestContext(c), uid, q)
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
	res, err := h.svc.CreateSession(auth.RequestContext(c), uid, req)
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
	res, err := h.svc.GetSession(auth.RequestContext(c), uid, uri.ID)
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
	res, err := h.svc.UpdateSession(auth.RequestContext(c), uid, uri.ID, req)
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
	if err := h.svc.DeleteSession(auth.RequestContext(c), uid, uri.ID); err != nil {
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
	if err := h.svc.ClearSessionMessages(auth.RequestContext(c), uid, uri.ID); err != nil {
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

func (h *AIHandler) CenterOverview(c *gin.Context) {
	response.Success(c, h.svc.CenterOverview(auth.RequestContext(c)))
}

func (h *AIHandler) ReseedCenter(c *gin.Context) {
	rep, err := h.svc.ReseedCenter(auth.RequestContext(c))
	if err != nil {
		// 仍返回报告，便于前端展示「目录不存在」等诊断
		if rep != nil {
			response.Success(c, gin.H{"ok": false, "error": err.Error(), "report": rep})
			return
		}
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true, "report": rep})
}

func (h *AIHandler) ListPrompts(c *gin.Context) {
	rows, err := h.svc.ListPrompts(auth.RequestContext(c))
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
	rows, err := h.svc.ListPromptVersions(auth.RequestContext(c), uri.ID)
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
	ver, err := h.svc.PublishPromptVersion(auth.RequestContext(c), uri.ID, uid, req)
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
	if err := h.svc.RollbackPromptVersion(auth.RequestContext(c), uri.ID, uri.VersionID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListLLMModels(c *gin.Context) {
	rows, err := h.svc.ListLLMModels(auth.RequestContext(c))
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
	row, err := h.svc.CreateLLMModel(auth.RequestContext(c), req)
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
	row, err := h.svc.UpdateLLMModel(auth.RequestContext(c), uri.ID, req)
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
	if err := h.svc.DeleteLLMModel(auth.RequestContext(c), uri.ID); err != nil {
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
	if err := h.svc.SetDefaultLLMModel(auth.RequestContext(c), uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListCenterTools(c *gin.Context) {
	rows, err := h.svc.ListTools(auth.RequestContext(c))
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
	if err := h.svc.UpdateToolEnabled(auth.RequestContext(c), uri.ID, req.Enabled); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *AIHandler) ListCases(c *gin.Context) {
	rows, err := h.svc.ListIncidentCases(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListSOPs(c *gin.Context) {
	rows, err := h.svc.ListSOPs(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListKBs(c *gin.Context) {
	rows, err := h.svc.ListKnowledgeBases(auth.RequestContext(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": rows})
}

func (h *AIHandler) ListEvalCases(c *gin.Context) {
	rows, err := h.svc.ListEvalCases(auth.RequestContext(c))
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
	run, err := h.svc.RunEvalSuite(auth.RequestContext(c), user, req.Live)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}
