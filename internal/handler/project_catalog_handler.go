package handler

import (
	"context"
	"strconv"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
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

func (h *ProjectCatalogHandler) IncidentContext(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.IncidentContextQuery) (*service.IncidentContext, error) {
		q.ProjectID = projectID
		return h.changes.IncidentContext(ctx, q)
	})
}

func (h *ProjectCatalogHandler) ListChangeEvents(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.ChangeEventListQuery) (*pagination.Result[model.ChangeEvent], error) {
		q.ProjectID = projectID
		return h.changes.List(ctx, q)
	})
}

func (h *ProjectCatalogHandler) ListFreezeWindows(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.FreezeWindowListQuery) (*pagination.Result[model.ChangeFreezeWindow], error) {
		q.ProjectID = projectID
		return h.changes.ListFreezeWindows(ctx, q)
	})
}

func (h *ProjectCatalogHandler) UpsertFreezeWindow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.FreezeWindowUpsertRequest) (*model.ChangeFreezeWindow, error) {
		req.ProjectID = projectID
		if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
			req.CreatedBy = u.ID
		}
		return h.changes.UpsertFreezeWindow(ctx, req)
	})
}

func (h *ProjectCatalogHandler) DeleteFreezeWindow(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	freezeID, err := parseUintParam(c, "freezeId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.changes.DeleteFreezeWindow(c.Request.Context(), projectID, freezeID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *ProjectCatalogHandler) ConflictCheck(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.ConflictCheckQuery) (*service.ConflictCheckResult, error) {
		q.ProjectID = projectID
		return h.changes.ConflictCheck(ctx, q)
	})
}

func (h *ProjectCatalogHandler) ListIncidents(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.IncidentListQuery) (*pagination.Result[model.Incident], error) {
		q.ProjectID = projectID
		return h.changes.ListIncidents(ctx, q)
	})
}

func (h *ProjectCatalogHandler) OpenIncident(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.IncidentOpenRequest) (*model.Incident, error) {
		req.ProjectID = projectID
		if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
			id := u.ID
			req.OpenedBy = &id
		}
		return h.changes.OpenIncident(ctx, req)
	})
}

func (h *ProjectCatalogHandler) UpdateIncident(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	incidentID, err := parseUintParam(c, "incidentId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.IncidentUpdateRequest) (*model.Incident, error) {
		req.ProjectID = projectID
		req.ID = incidentID
		return h.changes.UpdateIncident(ctx, req)
	})
}

func (h *ProjectCatalogHandler) GetIncidentTimeline(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	incidentID, err := parseUintParam(c, "incidentId")
	if err != nil {
		response.Error(c, err)
		return
	}
	window := 60
	if v := c.Query("window_minutes"); v != "" {
		if parsed, e := strconv.Atoi(v); e == nil && parsed > 0 {
			window = parsed
		}
	}
	timeline, err := h.changes.GetIncidentTimeline(c.Request.Context(), projectID, incidentID, window)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, timeline)
}

func (h *ProjectCatalogHandler) AddIncidentNote(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	incidentID, err := parseUintParam(c, "incidentId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req struct {
		Body string `json:"body" binding:"required"`
	}) (*model.IncidentNote, error) {
		var author uint
		if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
			author = u.ID
		}
		return h.changes.AddIncidentNote(ctx, projectID, incidentID, author, req.Body)
	})
}
