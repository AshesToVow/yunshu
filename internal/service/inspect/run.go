package inspect

// 巡检执行（inspect_run）：记录查询、手动触发入队、worker 内采集与报告生成。

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type RunListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (s *Service) ListRuns(ctx context.Context, projectID uint, q RunListQuery) (*pagination.Result[model.InspectRun], error) {
	page, size := pagination.Normalize(q.Page, q.PageSize)
	var total int64
	db := s.db.WithContext(ctx).Model(&model.InspectRun{}).Where("project_id = ?", projectID)
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.InspectRun
	err := db.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.InspectRun]{List: rows, Total: total, Page: page, PageSize: size}, nil
}

func (s *Service) GetRun(ctx context.Context, projectID, runID uint) (*model.InspectRun, error) {
	var run model.InspectRun
	err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&run).Error
	if err == gorm.ErrRecordNotFound {
		return nil, constants.ErrNotFoundWithMsg("巡检记录不存在")
	}
	return &run, err
}

type RunCreateRequest struct {
	DatasourceID uint `json:"datasource_id"`
}

func (s *Service) StartManualRun(ctx context.Context, projectID, userID uint, operatorName string, req RunCreateRequest) (*model.InspectRun, error) {
	plan, err := s.GetOrCreatePlan(ctx, projectID)
	if err != nil {
		return nil, err
	}
	dsID := req.DatasourceID
	if dsID == 0 {
		dsID = plan.DatasourceID
	}
	if dsID == 0 {
		return nil, constants.ErrBadRequestWithMsg("请指定 datasource_id 或在计划中配置数据源")
	}
	return s.enqueueNewRun(ctx, plan, dsID, "manual", userID, operatorName)
}

// enqueueNewRun 创建 pending 记录并入队异步执行，立即返回（不阻塞 HTTP）。
func (s *Service) enqueueNewRun(ctx context.Context, plan *model.InspectPlan, datasourceID uint, trigger string, userID uint, operatorName string) (*model.InspectRun, error) {
	if plan == nil || plan.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("invalid plan")
	}
	_, ds, err := s.dsSvc.PrometheusClient(ctx, datasourceID)
	if err != nil {
		return nil, err
	}
	if ds.ProjectID != plan.ProjectID {
		return nil, constants.ErrBadRequestWithMsg("数据源不属于当前项目")
	}

	run := model.InspectRun{
		ProjectID:      plan.ProjectID,
		PlanID:         plan.ID,
		Status:         "pending",
		Trigger:        trigger,
		DatasourceID:   datasourceID,
		DatasourceName: ds.Name,
		CreatedBy:      userID,
		OperatorName:   strings.TrimSpace(operatorName),
	}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}
	s.enqueueRun(run.ID)
	return &run, nil
}

func (s *Service) executeRun(ctx context.Context, plan *model.InspectPlan, datasourceID uint, trigger string, userID uint, operatorName string) (*model.InspectRun, error) {
	return s.enqueueNewRun(ctx, plan, datasourceID, trigger, userID, operatorName)
}

// performRun 在 worker 中执行采集与报告生成。状态落库用独立于 ctx 的后台 context，
// 避免请求取消/采集超时导致记录永远停在「执行中」；但该 context 仍带 inspectDBCtxTimeout，
// 不能是裸 Background——否则 DB/MinIO/SMTP 挂死时 worker goroutine 会永久泄漏。
func (s *Service) performRun(ctx context.Context, plan *model.InspectPlan, run *model.InspectRun) (*model.InspectRun, error) {
	if plan == nil || run == nil || plan.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("invalid plan")
	}
	dbCtx, dbCancel := context.WithTimeout(context.Background(), inspectDBCtxTimeout)
	defer dbCancel()
	projectName := fmt.Sprintf("project-%d", plan.ProjectID)
	if s.projects != nil {
		if p, err := s.projects.GetByID(dbCtx, plan.ProjectID); err == nil && p != nil {
			projectName = p.Name
		}
	}
	cli, ds, err := s.dsSvc.PrometheusClient(ctx, run.DatasourceID)
	if err != nil {
		return s.failRun(dbCtx, run, err)
	}
	if ds.ProjectID != plan.ProjectID {
		return s.failRun(dbCtx, run, constants.ErrBadRequestWithMsg("数据源不属于当前项目"))
	}
	healthNote := ""
	if health, herr := s.dsSvc.CheckHealth(ctx, run.DatasourceID); herr == nil && health != nil {
		switch health.Status {
		case model.DatasourceHealthDown:
			return s.failRun(dbCtx, run, constants.ErrBadRequestWithMsg("数据源不可用，已中止巡检："+health.Message))
		case model.DatasourceHealthDegraded:
			healthNote = "采集健康降级：" + health.Message + "。"
		}
	}

	storInfo := resolveReportStorageInfo(dbCtx, s.db, s.reportDir)
	if storInfo.RequireMinIO && !storInfo.MinioReady {
		reason := strings.TrimSpace(storInfo.MinioReason)
		if reason == "" {
			reason = "请配置数据字典 MinIO"
		}
		return s.failRun(dbCtx, run, constants.ErrBadRequestWithMsg("巡检报告须写入 MinIO（inspect_report.require_minio=true）："+reason))
	}

	items, err := s.effectiveItems(dbCtx, plan.ProjectID)
	if err != nil {
		return s.failRun(dbCtx, run, err)
	}
	collected := collectItems(ctx, cli, items, 8)
	operator := strings.TrimSpace(run.OperatorName)
	if operator == "" && run.Trigger == "cron" {
		operator = "系统定时"
	}
	data := buildReportData(projectName, ds.Name, operator, plan.ReportListMode, collected)
	if healthNote != "" {
		data.Summary = healthNote + data.Summary
	}

	// 风险台账期次对比：回填「新增/持续/已恢复」与上期的责任人、整改期限，
	// 再落库作为下一期的比对基线。落库失败只告警，不阻断报告生成。
	data.Ledger, data.Diff = s.applyPeriodDiff(dbCtx, plan.ProjectID, run.ID, data.Timestamp, data.Ledger)
	if err := s.saveFindings(dbCtx, plan.ProjectID, run.ID, data.Timestamp, data.Ledger, data.Diff); err != nil {
		slog.Default().With("component", "inspect.run", "run_id", run.ID, "project_id", plan.ProjectID).
			Warn("inspect findings persist failed", "error", err)
	}

	tpl, err := s.resolveReportTemplate(dbCtx, plan.ProjectID, plan.ReportTemplateID)
	if err != nil {
		return s.failRun(dbCtx, run, err)
	}
	htmlBytes, err := s.renderReportHTML(dbCtx, tpl.Code, tpl.Body, data)
	if err != nil {
		slog.Default().With("component", "inspect.report", "template", tpl.Code, "error", err).
			Warn("report template render failed, using builtin default")
		data.Summary = "（报告模板渲染失败，已使用内置默认模板）" + data.Summary
		htmlBytes, err = renderHTML(data)
		if err != nil {
			return s.failRun(dbCtx, run, err)
		}
	}
	printBytes, err := s.renderReportHTML(dbCtx, "print", "", data)
	if err != nil || len(printBytes) == 0 {
		printBytes = htmlBytes
	}
	// PDF 优先用可打印 HTML（与 print.html 一致），保证版式与浏览器预览接近
	pdfBytes := renderBinaryPDF(data, printBytes)
	excelBytes, excelErr := renderExcel(data)

	store := s.store(dbCtx)
	htmlKey := reportObjectKey(plan.ProjectID, run.ID, "html")
	printKey := reportObjectKey(plan.ProjectID, run.ID, "print.html")
	pdfKey := reportObjectKey(plan.ProjectID, run.ID, "pdf")
	excelKey := reportObjectKey(plan.ProjectID, run.ID, "xlsx")

	if err := store.Put(dbCtx, htmlKey, htmlBytes, "text/html; charset=utf-8"); err != nil {
		return s.failRun(dbCtx, run, err)
	}
	if err := store.Put(dbCtx, printKey, printBytes, "text/html; charset=utf-8"); err != nil {
		return s.failRun(dbCtx, run, err)
	}

	var writeWarnings []string
	if len(pdfBytes) >= 4 && string(pdfBytes[:4]) == "%PDF" {
		if err := store.Put(dbCtx, pdfKey, pdfBytes, "application/pdf"); err != nil {
			slog.Default().With("component", "inspect.run", "run_id", run.ID, "project_id", plan.ProjectID).
				Warn("inspect pdf report store failed", "error", err, "key", pdfKey)
			writeWarnings = append(writeWarnings, "PDF: "+err.Error())
		} else {
			run.ReportPDFPath = pdfKey
		}
	} else {
		run.ReportPDFPath = ""
	}
	if excelErr != nil {
		slog.Default().With("component", "inspect.run", "run_id", run.ID, "project_id", plan.ProjectID).
			Warn("inspect excel report render failed", "error", excelErr)
		writeWarnings = append(writeWarnings, "Excel 渲染: "+excelErr.Error())
	} else if len(excelBytes) > 0 {
		if err := store.Put(dbCtx, excelKey, excelBytes, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"); err != nil {
			slog.Default().With("component", "inspect.run", "run_id", run.ID, "project_id", plan.ProjectID).
				Warn("inspect excel report store failed", "error", err, "key", excelKey)
			writeWarnings = append(writeWarnings, "Excel: "+err.Error())
		} else {
			run.ReportExcelPath = excelKey
		}
	}

	finished := time.Now()
	run.Status = "success"
	run.Score = data.Score
	run.Grade = data.Grade
	run.Summary = data.Summary
	if len(writeWarnings) > 0 {
		warn := "部分附件写入失败：" + strings.Join(writeWarnings, "; ")
		if strings.TrimSpace(run.Summary) != "" {
			run.Summary += "\n" + warn
		} else {
			run.Summary = warn
		}
	}
	run.ErrorMessage = ""
	run.TotalCount = collected.Total
	run.CriticalCount = collected.Critical
	run.WarningCount = collected.Warning
	run.NormalCount = collected.Normal
	run.Storage = store.Backend()
	run.ReportHTMLPath = htmlKey
	run.ReportTemplateID = tpl.ID
	run.ReportTemplateCode = tpl.Code
	run.FinishedAt = &finished
	if err := s.db.WithContext(dbCtx).Save(run).Error; err != nil {
		return nil, err
	}
	plan.LastRunAt = &finished
	_ = s.db.WithContext(dbCtx).Model(plan).Update("last_run_at", finished).Error

	_ = s.sendRunEmail(dbCtx, plan, run, data, htmlBytes, pdfBytes)
	if run.CriticalCount > 0 || run.Status == "failed" {
		_ = s.sendInspectAnomalyEmail(dbCtx, plan, run, data)
	}
	return run, nil
}

func (s *Service) failRun(ctx context.Context, run *model.InspectRun, err error) (*model.InspectRun, error) {
	if run == nil {
		return nil, err
	}
	finished := time.Now()
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	// 刻意不复用入参 ctx：调用方 ctx 往往已因超时/取消失效，失败状态必须落库。
	// 但仍加超时上限，避免 DB/SMTP 挂死时 worker 卡住。
	dbCtx, cancel := context.WithTimeout(context.Background(), inspectFailRunTimeout)
	defer cancel()
	_ = s.db.WithContext(dbCtx).Model(&model.InspectRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"status":        "failed",
		"error_message": msg,
		"finished_at":   finished,
	}).Error
	run.Status = "failed"
	run.ErrorMessage = msg
	run.FinishedAt = &finished
	if plan, perr := s.getPlanByRun(dbCtx, run); perr == nil && plan != nil {
		data := ReportData{Project: fmt.Sprintf("project-%d", run.ProjectID), Score: run.Score, Grade: run.Grade, Summary: msg, Timestamp: finished}
		_ = s.sendInspectAnomalyEmail(dbCtx, plan, run, data)
	}
	return run, err
}
