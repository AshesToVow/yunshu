package handler

import (
	"context"
	"strconv"
	"yunshu/internal/service"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type WorkflowHandler struct {
	svc *service.WorkflowService
}

func NewWorkflowHandler(svc *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{svc: svc}
}

func (h *WorkflowHandler) GetDefinition(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, constants.ErrInternal)
		return
	}
	domain := c.Param("domain")
	projectID, err := parseWorkflowProjectParam(c, "project_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketType := c.DefaultQuery("ticket_type", model.WorkflowTicketTypeDefault)
	def, err := h.svc.GetDefinition(c.Request.Context(), service.DefinitionKey{
		Domain: domain, ProjectID: projectID, TicketType: ticketType,
	}, defaultStagesForDomain(domain))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, def)
}

func (h *WorkflowHandler) UpsertDefinition(c *gin.Context) {
	domain := c.Param("domain")
	projectID, err := parseWorkflowProjectParam(c, "project_id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ticketType := c.DefaultQuery("ticket_type", model.WorkflowTicketTypeDefault)
	ServeJSON(c, func(ctx context.Context, req service.DefinitionUpsertRequest) (*service.DefinitionResponse, error) {
		return h.svc.UpsertDefinition(ctx, service.DefinitionKey{
			Domain: domain, ProjectID: projectID, TicketType: ticketType,
		}, req)
	})
}

func (h *WorkflowHandler) ListPending(c *gin.Context) {
	var q service.PendingListQuery
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

func (h *WorkflowHandler) ListTickets(c *gin.Context) {
	var q service.WorkflowTicketListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequest)
		return
	}
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	data, err := h.svc.ListTickets(c.Request.Context(), q, actor)
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
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	data, err := h.svc.TicketDetail(c.Request.Context(), id, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func (h *WorkflowHandler) CreateTicket(c *gin.Context) {
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req service.CreateTicketRequest) (*service.TicketDetail, error) {
		// 提交人以登录态为准，忽略客户端传值（防止伪造提交人绕过禁止自审）
		req.SubmitterUserID = actor.ID
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
	ServeJSON(c, func(ctx context.Context, req service.ReviewStepRequest) (*service.TicketDetail, error) {
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

// CreateIncidentFromAlertRef POST /alerts/to-ticket：支持 alert_event_id 或 fingerprint。
func (h *WorkflowHandler) CreateIncidentFromAlertRef(c *gin.Context) {
	var req service.CreateIncidentFromAlertRefRequest
	_ = c.ShouldBindJSON(&req)
	actor, ok := auth.CurrentUserFromContext(c)
	if !ok {
		response.Error(c, constants.ErrNotLoggedIn)
		return
	}
	data, err := h.svc.CreateIncidentFromAlertRef(c.Request.Context(), req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func defaultStagesForDomain(domain string) []service.StageItem {
	switch domain {
	case model.WorkflowDomainDbmgmt:
		return service.DefaultDbmgmtStages()
	case model.WorkflowDomainCicd:
		return service.DefaultCicdStages()
	case model.WorkflowDomainIncident:
		return service.DefaultIncidentStages()
	case model.WorkflowDomainAI:
		return []service.StageItem{{
			StageKey: "ai_ops", StageName: "AI 运维审批", SortOrder: 10,
			Enabled: true, AssigneeRuleType: model.WorkflowAssigneePlatformRole,
		}}
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

// parseWorkflowProjectParam 允许 0（全局流程定义挂在 project_id=0 下）。
func parseWorkflowProjectParam(c *gin.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, constants.ErrBadRequest
	}
	return uint(v), nil
}
