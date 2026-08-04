package handler

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type ProjectCatalogHandler struct {
	catalog *service.ServiceCatalogService
	changes *service.ChangeEventService
}

func NewProjectCatalogHandler(
	catalog *service.ServiceCatalogService,
	changes *service.ChangeEventService,
) *ProjectCatalogHandler {
	return &ProjectCatalogHandler{catalog: catalog, changes: changes}
}

func (h *ProjectCatalogHandler) ListCatalog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.ServiceCatalogListQuery) (*pagination.Result[service.ServiceCatalogItem], error) {
		q.ProjectID = projectID
		return h.catalog.List(ctx, q)
	})
}

func (h *ProjectCatalogHandler) UpsertCatalog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServiceCatalogUpsertRequest) (*service.ServiceCatalogItem, error) {
		req.ProjectID = projectID
		return h.catalog.Upsert(ctx, req)
	})
}

func (h *ProjectCatalogHandler) DeleteCatalog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	catalogID, err := parseUintParam(c, "catalogId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.catalog.Delete(c.Request.Context(), projectID, catalogID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectCatalogHandler) AddLink(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	catalogID, err := parseUintParam(c, "catalogId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServiceLinkRequest) (*model.ServiceLink, error) {
		req.ProjectID = projectID
		req.ServiceID = catalogID
		return h.catalog.AddLink(ctx, req)
	})
}

func (h *ProjectCatalogHandler) DeleteLink(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	catalogID, err := parseUintParam(c, "catalogId")
	if err != nil {
		response.Error(c, err)
		return
	}
	linkID, err := parseUintParam(c, "linkId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.catalog.DeleteLink(c.Request.Context(), projectID, catalogID, linkID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectCatalogHandler) GetCatalog(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	catalogID, err := parseUintParam(c, "catalogId")
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.catalog.Get(c.Request.Context(), projectID, catalogID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ProjectCatalogHandler) GetPortrait(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	catalogID, err := parseUintParam(c, "catalogId")
	if err != nil {
		response.Error(c, err)
		return
	}
	sid := catalogID
	changes, err := h.changes.List(c.Request.Context(), service.ChangeEventListQuery{
		ProjectID: projectID,
		ServiceID: &sid,
		Page:      1,
		PageSize:  20,
	})
	var recent []model.ChangeEvent
	if err == nil && changes != nil {
		recent = changes.List
	}
	portrait, err := h.catalog.Portrait(c.Request.Context(), projectID, catalogID, recent)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, portrait)
}
