package handler

import (
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	esmgmtsvc "yunshu/internal/service/esmgmt"

	"github.com/gin-gonic/gin"
)

func (h *EsmgmtHandler) SearchDocs(c *gin.Context) {
	var req esmgmtsvc.SearchDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if req.ConnectionID == 0 {
		req.ConnectionID = parseOptionalUintQuery(c, "connection_id")
	}
	out, err := h.svc.SearchDocs(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) GetDoc(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	index := strings.TrimSpace(c.Query("index"))
	id := strings.TrimSpace(c.Query("id"))
	out, err := h.svc.GetDoc(c.Request.Context(), connID, index, id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"source": out})
}

func (h *EsmgmtHandler) UpsertDoc(c *gin.Context) {
	var req esmgmtsvc.UpsertDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.UpsertDoc(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) DeleteDoc(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connection_id"`
		Index        string `json:"index"`
		ID           string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteDoc(c.Request.Context(), req.ConnectionID, req.Index, req.ID, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) ListTemplates(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	kind := strings.TrimSpace(c.Query("kind"))
	out, err := h.svc.ListTemplates(c.Request.Context(), connID, kind)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) GetTemplate(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	kind := strings.TrimSpace(c.Query("kind"))
	name := strings.TrimSpace(c.Query("name"))
	out, err := h.svc.GetTemplate(c.Request.Context(), connID, kind, name)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) PutTemplate(c *gin.Context) {
	var req esmgmtsvc.PutTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.PutTemplate(c.Request.Context(), req, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) DeleteTemplate(c *gin.Context) {
	var req struct {
		ConnectionID uint   `json:"connection_id"`
		Kind         string `json:"kind"`
		Name         string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteTemplate(c.Request.Context(), req.ConnectionID, req.Kind, req.Name, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *EsmgmtHandler) CreateReindex(c *gin.Context) {
	var req esmgmtsvc.ReindexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	job, err := h.svc.CreateReindex(c.Request.Context(), req, actorFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, job)
}

func (h *EsmgmtHandler) ListReindexJobs(c *gin.Context) {
	connID := parseOptionalUintQuery(c, "connection_id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	out, err := h.svc.ListReindexJobs(c.Request.Context(), connID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) GetReindexJob(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.GetReindexJob(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, out)
}

func (h *EsmgmtHandler) CancelReindex(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg("任务 ID 无效"))
		return
	}
	if err := h.svc.CancelReindex(c.Request.Context(), id, actorFrom(c)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
