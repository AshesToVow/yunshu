package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/alert"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	redis    *redis.Client
	dsSvc    *alert.AlertDatasourceService
	projects interfaces.ProjectRepository
	mailer   mailer.Sender
	appName  string
	reportDir string
}

func NewService(
	db *gorm.DB,
	redisClient *redis.Client,
	dsSvc *alert.AlertDatasourceService,
	projects interfaces.ProjectRepository,
	sender mailer.Sender,
	appName string,
) *Service {
	dir := filepath.Join("logs", "inspect-reports")
	_ = os.MkdirAll(dir, 0o755)
	return &Service{
		db:        db,
		redis:     redisClient,
		dsSvc:     dsSvc,
		projects:  projects,
		mailer:    sender,
		appName:   strings.TrimSpace(appName),
		reportDir: dir,
	}
}

// SeedGlobalTemplates 幂等写入/刷新全局巡检模板项（按 type+name upsert）。
func (s *Service) SeedGlobalTemplates(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	for _, want := range defaultTemplateItems() {
		var row model.InspectItem
		err := s.db.WithContext(ctx).
			Where("project_id = 0 AND type = ? AND name = ?", want.Type, want.Name).
			First(&row).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.WithContext(ctx).Create(&want).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		row.Description = want.Description
		row.Query = want.Query
		row.Threshold = want.Threshold
		row.ThresholdType = want.ThresholdType
		row.Unit = want.Unit
		row.SortOrder = want.SortOrder
		// 不覆盖管理员已改的 Enabled
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type PlanUpsertRequest struct {
	Enabled        *bool  `json:"enabled"`
	CronSpec       string `json:"cron_spec"`
	DatasourceID   uint   `json:"datasource_id"`
	ReportListMode string `json:"report_list_mode"`
	Recipients     []string `json:"recipients"`
}

func (s *Service) GetOrCreatePlan(ctx context.Context, projectID uint) (*model.InspectPlan, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	var plan model.InspectPlan
	err := s.db.WithContext(ctx).Where("project_id = ?", projectID).First(&plan).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		plan = model.InspectPlan{
			ProjectID:      projectID,
			Enabled:        false,
			CronSpec:       "0 0 9 * * *",
			ReportListMode: "abnormal_only",
			RecipientsJSON: "[]",
		}
		if err := s.db.WithContext(ctx).Create(&plan).Error; err != nil {
			return nil, err
		}
	}
	_ = s.ensurePlanDefaults(ctx, &plan)
	return &plan, nil
}

// ensurePlanDefaults 首次进入自动绑定数据源、同步模板巡检项，减少手工配置。
func (s *Service) ensurePlanDefaults(ctx context.Context, plan *model.InspectPlan) error {
	if plan == nil {
		return nil
	}
	changed := false
	if plan.DatasourceID == 0 {
		var ds model.AlertDatasource
		err := s.db.WithContext(ctx).
			Where("project_id = ? AND enabled = ? AND type = ?", plan.ProjectID, true, "prometheus").
			Order("id ASC").
			First(&ds).Error
		if err == nil {
			plan.DatasourceID = ds.ID
			changed = true
		}
	}
	if changed {
		if err := s.db.WithContext(ctx).Save(plan).Error; err != nil {
			return err
		}
	}
	var n int64
	_ = s.db.WithContext(ctx).Model(&model.InspectItem{}).Where("project_id = ?", plan.ProjectID).Count(&n).Error
	if n == 0 {
		_, _ = s.SyncItemsFromTemplate(ctx, plan.ProjectID)
	}
	return nil
}

func (s *Service) UpdatePlan(ctx context.Context, projectID uint, req PlanUpsertRequest) (*model.InspectPlan, error) {
	plan, err := s.GetOrCreatePlan(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if req.Enabled != nil {
		plan.Enabled = *req.Enabled
	}
	if strings.TrimSpace(req.CronSpec) != "" {
		if err := cronutil.ValidateSpec(req.CronSpec, "cron_spec"); err != nil {
			return nil, err
		}
		plan.CronSpec = strings.TrimSpace(req.CronSpec)
	}
	if req.DatasourceID > 0 {
		plan.DatasourceID = req.DatasourceID
	}
	mode := strings.TrimSpace(req.ReportListMode)
	if mode != "" {
		switch mode {
		case "abnormal_only", "summary", "all":
			plan.ReportListMode = mode
		default:
			return nil, constants.ErrBadRequestWithMsg("invalid report_list_mode")
		}
	}
	if req.Recipients != nil {
		b, _ := json.Marshal(uniqEmails(req.Recipients))
		plan.RecipientsJSON = string(b)
	}
	if plan.Enabled && plan.DatasourceID == 0 {
		return nil, constants.ErrBadRequestWithMsg("启用定时巡检前请先配置 datasource_id")
	}
	if err := s.db.WithContext(ctx).Save(plan).Error; err != nil {
		return nil, err
	}
	return plan, nil
}

type ItemUpsertRequest struct {
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Query         string  `json:"query"`
	Threshold     float64 `json:"threshold"`
	ThresholdType string  `json:"threshold_type"`
	Unit          string  `json:"unit"`
	LabelsJSON    string  `json:"labels_json"`
	Enabled       *bool   `json:"enabled"`
	SortOrder     *int    `json:"sort_order"`
}

func (s *Service) ListItems(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var projectItems []model.InspectItem
	if err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("sort_order ASC, id ASC").
		Find(&projectItems).Error; err != nil {
		return nil, err
	}
	if len(projectItems) > 0 {
		return projectItems, nil
	}
	var globals []model.InspectItem
	err := s.db.WithContext(ctx).
		Where("project_id = 0").
		Order("sort_order ASC, id ASC").
		Find(&globals).Error
	return globals, err
}

func (s *Service) CreateItem(ctx context.Context, projectID uint, req ItemUpsertRequest) (*model.InspectItem, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	name := strings.TrimSpace(req.Name)
	query := strings.TrimSpace(req.Query)
	if name == "" || query == "" {
		return nil, constants.ErrBadRequestWithMsg("name and query required")
	}
	tt := strings.TrimSpace(req.ThresholdType)
	if tt == "" {
		tt = "greater"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sortOrder := 1000
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}
	item := model.InspectItem{
		ProjectID:     projectID,
		Type:          strings.TrimSpace(req.Type),
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		Query:         query,
		Threshold:     req.Threshold,
		ThresholdType: tt,
		Unit:          strings.TrimSpace(req.Unit),
		LabelsJSON:    strings.TrimSpace(req.LabelsJSON),
		Enabled:       enabled,
		SortOrder:     sortOrder,
	}
	if item.Type == "" {
		item.Type = "自定义"
	}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) UpdateItem(ctx context.Context, projectID, itemID uint, req ItemUpsertRequest) (*model.InspectItem, error) {
	var item model.InspectItem
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("巡检项不存在或不可修改全局模板（请先同步到项目）")
		}
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		item.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Type) != "" {
		item.Type = strings.TrimSpace(req.Type)
	}
	if strings.TrimSpace(req.Query) != "" {
		item.Query = strings.TrimSpace(req.Query)
	}
	item.Description = strings.TrimSpace(req.Description)
	item.Threshold = req.Threshold
	if strings.TrimSpace(req.ThresholdType) != "" {
		item.ThresholdType = strings.TrimSpace(req.ThresholdType)
	}
	item.Unit = strings.TrimSpace(req.Unit)
	if req.LabelsJSON != "" {
		item.LabelsJSON = strings.TrimSpace(req.LabelsJSON)
	}
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		item.SortOrder = *req.SortOrder
	}
	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) DeleteItem(ctx context.Context, projectID, itemID uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).Delete(&model.InspectItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("巡检项不存在")
	}
	return nil
}

// SyncItemsFromTemplate 将全局模板复制为项目项（已存在同名则跳过；含默认关闭项，便于按需启用）。
func (s *Service) SyncItemsFromTemplate(ctx context.Context, projectID uint) (int, error) {
	if projectID == 0 {
		return 0, constants.ErrBadRequestWithMsg("project_id required")
	}
	var globals []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = 0").Order("sort_order ASC, id ASC").Find(&globals).Error; err != nil {
		return 0, err
	}
	var existing []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&existing).Error; err != nil {
		return 0, err
	}
	have := map[string]bool{}
	for _, e := range existing {
		have[e.Type+"|"+e.Name] = true
	}
	created := 0
	for _, g := range globals {
		key := g.Type + "|" + g.Name
		if have[key] {
			continue
		}
		cp := g
		cp.ID = 0
		cp.ProjectID = projectID
		cp.CreatedAt = time.Time{}
		cp.UpdatedAt = time.Time{}
		if err := s.db.WithContext(ctx).Create(&cp).Error; err != nil {
			return created, err
		}
		created++
		have[key] = true
	}
	return created, nil
}

// ResetItemsFromTemplate 删除项目自有巡检项后，重新从全局模板全量同步（用于切换 Telegraf 模板等）。
func (s *Service) ResetItemsFromTemplate(ctx context.Context, projectID uint) (int, error) {
	if projectID == 0 {
		return 0, constants.ErrBadRequestWithMsg("project_id required")
	}
	if err := s.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&model.InspectItem{}).Error; err != nil {
		return 0, err
	}
	return s.SyncItemsFromTemplate(ctx, projectID)
}

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

func (s *Service) StartManualRun(ctx context.Context, projectID, userID uint, req RunCreateRequest) (*model.InspectRun, error) {
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
	return s.executeRun(ctx, plan, dsID, "manual", userID)
}

func (s *Service) executeRun(ctx context.Context, plan *model.InspectPlan, datasourceID uint, trigger string, userID uint) (*model.InspectRun, error) {
	if plan == nil || plan.ProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("invalid plan")
	}
	projectName := fmt.Sprintf("project-%d", plan.ProjectID)
	if s.projects != nil {
		if p, err := s.projects.GetByID(ctx, plan.ProjectID); err == nil && p != nil {
			projectName = p.Name
		}
	}
	cli, ds, err := s.dsSvc.PrometheusClient(ctx, datasourceID)
	if err != nil {
		return nil, err
	}
	if ds.ProjectID != plan.ProjectID {
		return nil, constants.ErrBadRequestWithMsg("数据源不属于当前项目")
	}

	now := time.Now()
	run := model.InspectRun{
		ProjectID:      plan.ProjectID,
		PlanID:         plan.ID,
		Status:         "running",
		Trigger:        trigger,
		DatasourceID:   datasourceID,
		DatasourceName: ds.Name,
		StartedAt:      &now,
		CreatedBy:      userID,
	}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}

	items, err := s.effectiveItems(ctx, plan.ProjectID)
	if err != nil {
		return s.failRun(ctx, &run, err)
	}
	collected := collectItems(ctx, cli, items, 8)
	user := ""
	if userID > 0 {
		user = fmt.Sprintf("user#%d", userID)
	}
	data := buildReportData(projectName, ds.Name, user, plan.ReportListMode, collected)
	htmlBytes, err := renderHTML(data)
	if err != nil {
		return s.failRun(ctx, &run, err)
	}
	// PDF 通道存放可打印 HTML（含中文）；浏览器「打印 → 另存为 PDF」即可，避免 Helvetica 中文乱码。
	printBytes := htmlBytes

	htmlPath, err := s.writeReportFile(plan.ProjectID, run.ID, "html", htmlBytes)
	if err != nil {
		return s.failRun(ctx, &run, err)
	}
	pdfPath, err := s.writeReportFile(plan.ProjectID, run.ID, "print.html", printBytes)
	if err != nil {
		return s.failRun(ctx, &run, err)
	}

	finished := time.Now()
	run.Status = "success"
	run.Score = data.Score
	run.Grade = data.Grade
	run.Summary = data.Summary
	run.TotalCount = collected.Total
	run.CriticalCount = collected.Critical
	run.WarningCount = collected.Warning
	run.NormalCount = collected.Normal
	run.ReportHTMLPath = htmlPath
	run.ReportPDFPath = pdfPath
	run.FinishedAt = &finished
	if err := s.db.WithContext(ctx).Save(&run).Error; err != nil {
		return nil, err
	}
	plan.LastRunAt = &finished
	_ = s.db.WithContext(ctx).Model(plan).Update("last_run_at", finished).Error

	_ = s.sendRunEmail(ctx, plan, &run, data, htmlBytes, nil)
	return &run, nil
}

func (s *Service) effectiveItems(ctx context.Context, projectID uint) ([]model.InspectItem, error) {
	var projectItems []model.InspectItem
	if err := s.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).
		Order("sort_order ASC, id ASC").Find(&projectItems).Error; err != nil {
		return nil, err
	}
	if len(projectItems) > 0 {
		return projectItems, nil
	}
	var globals []model.InspectItem
	err := s.db.WithContext(ctx).Where("project_id = 0 AND enabled = ?", true).
		Order("sort_order ASC, id ASC").Find(&globals).Error
	return globals, err
}

func (s *Service) failRun(ctx context.Context, run *model.InspectRun, err error) (*model.InspectRun, error) {
	finished := time.Now()
	run.Status = "failed"
	run.ErrorMessage = err.Error()
	run.FinishedAt = &finished
	_ = s.db.WithContext(ctx).Save(run).Error
	return run, err
}

func (s *Service) writeReportFile(projectID, runID uint, ext string, body []byte) (string, error) {
	dir := filepath.Join(s.reportDir, fmt.Sprintf("%d", projectID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("run-%d.%s", runID, ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) ReadReport(ctx context.Context, projectID, runID uint, kind string) ([]byte, string, error) {
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return nil, "", err
	}
	path := run.ReportHTMLPath
	ctype := "text/html; charset=utf-8"
	if kind == "pdf" {
		path = run.ReportPDFPath
		if path == "" {
			path = run.ReportHTMLPath
		}
		// 打印版实为 HTML（支持中文）；前端可引导浏览器另存 PDF
		ctype = "text/html; charset=utf-8"
	}
	if strings.TrimSpace(path) == "" {
		return nil, "", constants.ErrNotFoundWithMsg("报告文件不存在")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", constants.ErrNotFoundWithMsg("报告文件不存在")
	}
	return b, ctype, nil
}

func (s *Service) ResendEmail(ctx context.Context, projectID, runID uint) error {
	run, err := s.GetRun(ctx, projectID, runID)
	if err != nil {
		return err
	}
	plan, err := s.GetOrCreatePlan(ctx, projectID)
	if err != nil {
		return err
	}
	htmlBytes, _, err := s.ReadReport(ctx, projectID, runID, "html")
	if err != nil {
		return err
	}
	data := ReportData{
		Project:    fmt.Sprintf("project-%d", projectID),
		Datasource: run.DatasourceName,
		Score:      run.Score,
		Grade:      run.Grade,
		Summary:    run.Summary,
		Timestamp:  time.Now(),
	}
	return s.sendRunEmail(ctx, plan, run, data, htmlBytes, nil)
}

func (s *Service) sendRunEmail(ctx context.Context, plan *model.InspectPlan, run *model.InspectRun, data ReportData, html, pdf []byte) error {
	if s.mailer == nil || !s.mailer.Enabled() || plan == nil {
		return nil
	}
	recipients := parseRecipients(plan.RecipientsJSON)
	if len(recipients) == 0 {
		return nil
	}
	subject := fmt.Sprintf("[%s] 巡检报告 %s 分数%.0f", s.appNameOrDefault(), data.Project, data.Score)
	text := data.Summary
	htmlBody := fmt.Sprintf("<p>%s</p><p>严重 %d / 警告 %d / 正常 %d</p>", data.Summary, run.CriticalCount, run.WarningCount, run.NormalCount)
	atts := []mailer.Attachment{
		{Filename: fmt.Sprintf("inspect-run-%d.html", run.ID), ContentType: "text/html; charset=utf-8", Content: html},
	}
	if len(pdf) > 0 {
		atts = append(atts, mailer.Attachment{Filename: fmt.Sprintf("inspect-run-%d.pdf", run.ID), ContentType: "application/pdf", Content: pdf})
	}
	var lastErr error
	for _, to := range recipients {
		if err := s.mailer.SendWithAttachments(ctx, to, subject, text, htmlBody, atts); err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(run).Update("email_sent_at", now).Error
	}
	return lastErr
}

func (s *Service) appNameOrDefault() string {
	if s.appName != "" {
		return s.appName
	}
	return "yunshu"
}

func parseRecipients(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		parts := strings.Split(raw, ",")
		list = parts
	}
	return uniqEmails(list)
}

func uniqEmails(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	return out
}
