package handler

import (
	"context"
	"net/http"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/model"
	inspectsvc "yunshu/internal/service/inspect"

	"github.com/gin-gonic/gin"
)

type InspectHandler struct {
	svc *inspectsvc.Service
}

func NewInspectHandler(svc *inspectsvc.Service) *InspectHandler {
	return &InspectHandler{svc: svc}
}

func (h *InspectHandler) GetPlan(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	plan, err := h.svc.GetOrCreatePlan(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, plan)
}

func (h *InspectHandler) UpdatePlan(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req inspectsvc.PlanUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	plan, err := h.svc.UpdatePlan(c.Request.Context(), projectID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, plan)
}

func (h *InspectHandler) ListItems(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	items, err := h.svc.ListItems(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, items)
}

func (h *InspectHandler) CreateItem(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req inspectsvc.ItemUpsertRequest) (*model.InspectItem, error) {
		return h.svc.CreateItem(ctx, projectID, req)
	})
}

func (h *InspectHandler) UpdateItem(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req inspectsvc.ItemUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.UpdateItem(c.Request.Context(), projectID, itemID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, item)
}

func (h *InspectHandler) DeleteItem(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	itemID, err := parseUintParam(c, "itemId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteItem(c.Request.Context(), projectID, itemID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *InspectHandler) SyncItems(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	n, err := h.svc.SyncItemsFromTemplate(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"created": n})
}

func (h *InspectHandler) ListRuns(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q inspectsvc.RunListQuery) (*pagination.Result[model.InspectRun], error) {
		return h.svc.ListRuns(ctx, projectID, q)
	})
}

func (h *InspectHandler) CreateRun(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req inspectsvc.RunCreateRequest
	_ = c.ShouldBindJSON(&req)
	actor, _ := auth.CurrentUserFromContext(c)
	var uid uint
	if actor != nil {
		uid = actor.ID
	}
	run, err := h.svc.StartManualRun(c.Request.Context(), projectID, uid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *InspectHandler) GetRun(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	run, err := h.svc.GetRun(c.Request.Context(), projectID, runID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, run)
}

func (h *InspectHandler) ReportHTML(c *gin.Context) {
	h.serveReport(c, "html")
}

func (h *InspectHandler) ReportPDF(c *gin.Context) {
	h.serveReport(c, "pdf")
}

func (h *InspectHandler) serveReport(c *gin.Context, kind string) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	body, ctype, err := h.svc.ReadReport(c.Request.Context(), projectID, runID, kind)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, ctype, body)
}

func (h *InspectHandler) ResendEmail(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	runID, err := parseUintParam(c, "runId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.ResendEmail(c.Request.Context(), projectID, runID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}
