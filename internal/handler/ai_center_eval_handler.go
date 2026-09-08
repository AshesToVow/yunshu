package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

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

func (h *AIHandler) GetEvalCase(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.GetEvalCase(auth.RequestContext(c), uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) CreateEvalCase(c *gin.Context) {
	var req aisvc.EvalCaseUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.CreateEvalCase(auth.RequestContext(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) UpdateEvalCase(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.EvalCaseUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdateEvalCase(auth.RequestContext(c), uri.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) DeleteEvalCase(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteEvalCase(auth.RequestContext(c), uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
