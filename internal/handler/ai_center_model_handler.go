package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

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
