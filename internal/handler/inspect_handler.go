package handler

import (
	"context"
	"net/http"
	"strings"

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

func (h *InspectHandler) GetStorageInfo(c *gin.Context) {
	if _, err := parseUintParam(c, "id"); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, h.svc.ReportStorageInfo(c.Request.Context()))
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

func (h *InspectHandler) ResetItems(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	n, err := h.svc.ResetItemsFromTemplate(c.Request.Context(), projectID)
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
	operator := ""
	if actor != nil {
		uid = actor.ID
		operator = strings.TrimSpace(actor.Nickname)
		if operator == "" {
			operator = strings.TrimSpace(actor.Username)
		}
	}
	run, err := h.svc.StartManualRun(c.Request.Context(), projectID, uid, operator, req)
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

func (h *InspectHandler) ReportExcel(c *gin.Context) {
	h.serveReport(c, "excel")
}

func (h *InspectHandler) ReportPrint(c *gin.Context) {
	h.serveReport(c, "print")
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
	if kind == "excel" {
		c.Header("Content-Disposition", `attachment; filename="inspect-run-`+c.Param("runId")+`.xlsx"`)
	}
	if kind == "pdf" && strings.HasPrefix(ctype, "application/pdf") {
		c.Header("Content-Disposition", `attachment; filename="inspect-run-`+c.Param("runId")+`.pdf"`)
	}
	c.Data(http.StatusOK, ctype, body)
}

func (h *InspectHandler) ListReportTemplates(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	list, err := h.svc.ListReportTemplates(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, list)
}

func (h *InspectHandler) CreateReportTemplate(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON201(c, func(ctx context.Context, req inspectsvc.ReportTemplateUpsertRequest) (*model.InspectReportTemplate, error) {
		return h.svc.CreateReportTemplate(ctx, projectID, req)
	})
}

func (h *InspectHandler) UpdateReportTemplate(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	tid, err := parseUintParam(c, "templateId")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req inspectsvc.ReportTemplateUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.UpdateReportTemplate(c.Request.Context(), projectID, tid, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *InspectHandler) DeleteReportTemplate(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	tid, err := parseUintParam(c, "templateId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteReportTemplate(c.Request.Context(), projectID, tid); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type copyReportTemplateReq struct {
	SourceID uint   `json:"source_id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
}

func (h *InspectHandler) CopyReportTemplate(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req copyReportTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	row, err := h.svc.CopyReportTemplate(c.Request.Context(), projectID, req.SourceID, req.Code, req.Name)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, row)
}

func (h *InspectHandler) PreviewReportTemplate(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	var req inspectsvc.ReportPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	body, err := h.svc.PreviewReportTemplate(c.Request.Context(), projectID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", body)
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
