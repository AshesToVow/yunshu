package handler

import (
	"context"
	"strconv"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	workflowsvc "yunshu/internal/service/workflow"

	"github.com/gin-gonic/gin"
)

type WorkflowHandler struct {
	svc *workflowsvc.Service
}

func NewWorkflowHandler(svc *workflowsvc.Service) *WorkflowHandler {
	return &WorkflowHandler{svc: svc}
}

func (h *WorkflowHandler) GetDefinition(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	domain := c.Param("domain")
	projectID, _ := strconv.ParseUint(c.Param("project_id"), 10, 64)
	ticketType := c.DefaultQuery("ticket_type", model.WorkflowTicketTypeDefault)
	def, err := h.svc.GetDefinition(c.Request.Context(), workflowsvc.DefinitionKey{
		Domain: domain, ProjectID: uint(projectID), TicketType: ticketType,
	}, defaultStagesForDomain(domain))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, def)
}

func (h *WorkflowHandler) UpsertDefinition(c *gin.Context) {
	domain := c.Param("domain")
	projectID, _ := strconv.ParseUint(c.Param("project_id"), 10, 64)
	ticketType := c.DefaultQuery("ticket_type", model.WorkflowTicketTypeDefault)
	ServeJSON(c, func(ctx context.Context, req workflowsvc.DefinitionUpsertRequest) (*workflowsvc.DefinitionResponse, error) {
		return h.svc.UpsertDefinition(ctx, workflowsvc.DefinitionKey{
			Domain: domain, ProjectID: uint(projectID), TicketType: ticketType,
		}, req)
	})
}

func (h *WorkflowHandler) ListPending(c *gin.Context) {
	var q workflowsvc.PendingListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	data, err := h.svc.ListPendingForUser(c.Request.Context(), q, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkflowHandler) ReviewPendingStep(c *gin.Context) {
	stepID, err := parseWorkflowUintParam(c, "step_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketID, err := parseWorkflowUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	ServeJSON(c, func(ctx context.Context, req workflowsvc.ReviewStepRequest) (*workflowsvc.TicketDetail, error) {
		return h.svc.ReviewStep(ctx, ticketID, stepID, req, actor)
	})
}

func (h *WorkflowHandler) ListTickets(c *gin.Context) {
	var q workflowsvc.TicketListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	data, err := h.svc.ListTickets(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkflowHandler) TicketDetail(c *gin.Context) {
	id, err := parseWorkflowUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.TicketDetail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkflowHandler) CreateTicket(c *gin.Context) {
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON201(c, func(ctx context.Context, req workflowsvc.CreateTicketRequest) (*workflowsvc.TicketDetail, error) {
		return h.svc.CreateTicket(ctx, req, actor)
	})
}

func (h *WorkflowHandler) ReviewStep(c *gin.Context) {
	ticketID, err := parseWorkflowUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	stepID, err := parseWorkflowUintParam(c, "step_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	ServeJSON(c, func(ctx context.Context, req workflowsvc.ReviewStepRequest) (*workflowsvc.TicketDetail, error) {
		return h.svc.ReviewStep(ctx, ticketID, stepID, req, actor)
	})
}

func (h *WorkflowHandler) CreateIncidentFromAlert(c *gin.Context) {
	alertID, err := parseWorkflowUintParam(c, "alert_event_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	type body struct {
		Title string `json:"title"`
	}
	var req body
	_ = c.ShouldBindJSON(&req)
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	data, err := h.svc.CreateIncidentFromAlert(c.Request.Context(), alertID, req.Title, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func defaultStagesForDomain(domain string) []workflowsvc.StageItem {
	switch domain {
	case model.WorkflowDomainDbmgmt:
		return workflowsvc.DefaultDbmgmtStages()
	case model.WorkflowDomainCicd:
		return workflowsvc.DefaultCicdStages()
	case model.WorkflowDomainIncident:
		return workflowsvc.DefaultIncidentStages()
	default:
		return nil
	}
}

func parseWorkflowUintParam(c *gin.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || v == 0 {
		return 0, constants.ErrBadRequest
	}
	return uint(v), nil
}
