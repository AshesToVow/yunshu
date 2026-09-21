package handler

import (
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	aisvc "yunshu/internal/service/ai"

	"github.com/gin-gonic/gin"
)

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

func (h *AIHandler) GetPrompt(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.GetPrompt(auth.RequestContext(c), uri.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) CreatePrompt(c *gin.Context) {
	var req aisvc.PromptUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	user, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if user != nil {
		uid = user.ID
	}
	row, err := h.svc.CreatePrompt(auth.RequestContext(c), uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) UpdatePrompt(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	var req aisvc.PromptUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdatePrompt(auth.RequestContext(c), uri.ID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *AIHandler) DeletePrompt(c *gin.Context) {
	var uri struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&uri); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeletePrompt(auth.RequestContext(c), uri.ID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
