package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
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
	if kind == "html" || kind == "print" {
		uploadURL := fmt.Sprintf("/api/v1/projects/%d/inspect/runs/%d/report.pdf", projectID, runID)
		body = inspectsvc.EnhanceReportHTML(body, "/api/v1/inspect/pdf-libs", uploadURL)
	}
	downloadName := h.svc.ReportDownloadFilename(c.Request.Context(), projectID, runID, kind)
	if kind == "excel" {
		c.Header("Content-Disposition", exportutil.ContentDispositionAttachment(downloadName))
	}
	if kind == "pdf" && strings.HasPrefix(ctype, "application/pdf") {
		c.Header("Content-Disposition", exportutil.ContentDispositionAttachment(downloadName))
	}
	c.Data(http.StatusOK, ctype, body)
}

func (h *InspectHandler) SaveReportPDF(c *gin.Context) {
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
	fh, err := c.FormFile("pdf")
	if err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg("缺少 PDF 文件"))
		return
	}
	if fh.Size > 30<<20 {
		response.Error(c, constants.ErrBadRequestWithMsg("PDF 文件过大"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.Error(c, err)
		return
	}
	defer f.Close()
	pdf, err := io.ReadAll(f)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.SaveReportPDF(c.Request.Context(), projectID, runID, pdf); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *InspectHandler) CheckReportPDF(c *gin.Context) {
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
	st, err := h.svc.CheckReportPDF(c.Request.Context(), projectID, runID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, st)
}

func (h *InspectHandler) ServePDFLib(c *gin.Context) {
	name := c.Param("name")
	body, err := inspectsvc.ReadPDFLib(name)
	if err != nil {
		response.Error(c, constants.ErrNotFoundWithMsg("资源不存在"))
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "application/javascript; charset=utf-8", body)
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

func (h *InspectHandler) ListRunTrends(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	limit := 30
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, e := strconv.Atoi(v); e == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.svc.ListRunTrends(c.Request.Context(), projectID, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func (h *InspectHandler) MigrateReportsToMinIO(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	n, err := h.svc.MigrateReportsToMinIO(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"migrated": n})
}
