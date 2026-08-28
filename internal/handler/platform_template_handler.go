package handler

import (
	"context"
	"strconv"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service/platformtpl"

	"github.com/gin-gonic/gin"
)

type PlatformTemplateHandler struct {
	svc *platformtpl.Service
}

func NewPlatformTemplateHandler(svc *platformtpl.Service) *PlatformTemplateHandler {
	return &PlatformTemplateHandler{svc: svc}
}

func (h *PlatformTemplateHandler) List(c *gin.Context) {
	var q platformtpl.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PlatformTemplateHandler) Detail(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PlatformTemplateHandler) Create(c *gin.Context) {
	actor, _ := auth.CurrentUserFromContext(c)
	actorID := uint(0)
	if actor != nil {
		actorID = actor.ID
	}
	ServeJSON201(c, func(ctx context.Context, req platformtpl.UpsertRequest) (*platformtpl.TemplateItem, error) {
		return h.svc.Create(ctx, req, actorID)
	})
}

func (h *PlatformTemplateHandler) Update(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req platformtpl.UpsertRequest) (*platformtpl.TemplateItem, error) {
		return h.svc.Update(ctx, id, req)
	})
}

func (h *PlatformTemplateHandler) Delete(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *PlatformTemplateHandler) SaveDraft(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	actorID := uint(0)
	if actor != nil {
		actorID = actor.ID
	}
	ServeJSON(c, func(ctx context.Context, req platformtpl.SaveDraftRequest) (*platformtpl.VersionItem, error) {
		return h.svc.SaveDraft(ctx, id, req, actorID)
	})
}

func (h *PlatformTemplateHandler) Publish(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	type body struct {
		Version int `json:"version"`
	}
	var req body
	_ = c.ShouldBindJSON(&req)
	data, err := h.svc.Publish(c.Request.Context(), id, req.Version)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PlatformTemplateHandler) ListVersions(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.ListVersions(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PlatformTemplateHandler) GetVersion(c *gin.Context) {
	id, err := parsePlatformTplID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	ver, err := strconv.Atoi(c.Param("version"))
	if err != nil || ver <= 0 {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.svc.GetVersionContent(c.Request.Context(), id, ver)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *PlatformTemplateHandler) Resolve(c *gin.Context) {
	key := c.Param("template_key")
	if key == "" {
		key = c.Query("template_key")
	}
	data, err := h.svc.ResolvePublished(c.Request.Context(), key)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func parsePlatformTplID(c *gin.Context) (uint, error) {
	v, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || v == 0 {
		return 0, constants.ErrBadRequest
	}
	return uint(v), nil
}
