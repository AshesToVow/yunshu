package inspect

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"math"
	"path"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

var reportTemplateFuncs = template.FuncMap{
	"fmtFloat": func(v float64) string {
		if math.Abs(v-math.Round(v)) < 1e-9 {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.2f", v)
	},
	"statusText": func(s string) string {
		switch s {
		case "critical":
			return "严重"
		case "warning":
			return "警告"
		default:
			return "正常"
		}
	},
	"gradeClass": func(g string) string {
		switch strings.ToUpper(strings.TrimSpace(g)) {
		case "A":
			return "grade-a"
		case "B":
			return "grade-b"
		case "C":
			return "grade-c"
		default:
			return "grade-d"
		}
	},
	"sampleNote": sampleNoteText,
	"eq":         func(a, b string) bool { return a == b },
}

func builtinTemplateFile(code string) string {
	switch strings.TrimSpace(code) {
	case "compact":
		return "templates/compact.html"
	case "executive":
		return "templates/executive.html"
	default:
		return "templates/report.html"
	}
}

func renderHTMLWithTemplate(code, body string, data ReportData) ([]byte, error) {
	name := strings.TrimSpace(code)
	if name == "" {
		name = "default"
	}
	body = strings.TrimSpace(body)
	var (
		tpl *template.Template
		err error
	)
	if body != "" {
		// 自定义正文：模板名用 code，便于报错定位
		tpl, err = template.New(name).Funcs(reportTemplateFuncs).Parse(body)
	} else {
		// embed 文件：ParseFS 以 basename 作为模板名，Execute 必须对准该名
		file := builtinTemplateFile(name)
		base := path.Base(file)
		tpl, err = template.New(base).Funcs(reportTemplateFuncs).ParseFS(reportFS, file)
	}
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Service) resolveReportTemplate(ctx context.Context, projectID, templateID uint) (*model.InspectReportTemplate, error) {
	if templateID > 0 {
		var row model.InspectReportTemplate
		err := s.db.WithContext(ctx).
			Where("id = ? AND status = 1 AND (project_id = 0 OR project_id = ?)", templateID, projectID).
			First(&row).Error
		if err == nil {
			return &row, nil
		}
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	var def model.InspectReportTemplate
	err := s.db.WithContext(ctx).
		Where("project_id = 0 AND code = ? AND status = 1", "default").
		First(&def).Error
	if err == nil {
		return &def, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	// DB 尚未 seed 时回退 embed
	return &model.InspectReportTemplate{Code: "default", Name: "标准版", Engine: "go_html", IsBuiltin: true, Status: 1}, nil
}

func builtinReportTemplateSeeds() []model.InspectReportTemplate {
	return []model.InspectReportTemplate{
		{ProjectID: 0, Code: "default", Name: "标准版", Engine: "go_html", IsBuiltin: true, Status: 1, Remark: "清晰分区，适合日常巡检归档"},
		{ProjectID: 0, Code: "compact", Name: "紧凑版", Engine: "go_html", IsBuiltin: true, Status: 1, Remark: "信息密度高，适合邮件快速浏览"},
		{ProjectID: 0, Code: "executive", Name: "摘要版", Engine: "go_html", IsBuiltin: true, Status: 1, Remark: "侧重结论与关注项，不含全量明细表"},
	}
}

// SeedReportTemplates 幂等写入全局内置报告版式。
func (s *Service) SeedReportTemplates(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	for _, want := range builtinReportTemplateSeeds() {
		var row model.InspectReportTemplate
		err := s.db.WithContext(ctx).
			Where("project_id = 0 AND code = ?", want.Code).
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
		row.Name = want.Name
		row.Remark = want.Remark
		row.IsBuiltin = true
		row.Status = 1
		row.Engine = "go_html"
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

type ReportTemplateUpsertRequest struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Body   string `json:"body"`
	Remark string `json:"remark"`
	Status *int   `json:"status"`
}

func (s *Service) ListReportTemplates(ctx context.Context, projectID uint) ([]model.InspectReportTemplate, error) {
	var rows []model.InspectReportTemplate
	err := s.db.WithContext(ctx).
		Where("(project_id = 0 OR project_id = ?) AND status = 1", projectID).
		Order("project_id ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (s *Service) CreateReportTemplate(ctx context.Context, projectID uint, req ReportTemplateUpsertRequest) (*model.InspectReportTemplate, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	body := strings.TrimSpace(req.Body)
	if code == "" || name == "" || body == "" {
		return nil, constants.ErrBadRequestWithMsg("code、name、body 必填")
	}
	if _, err := renderHTMLWithTemplate(code, body, ReportData{Project: "preview", Grade: "A"}); err != nil {
		return nil, constants.ErrBadRequestWithMsg("模板解析失败: " + err.Error())
	}
	row := model.InspectReportTemplate{
		ProjectID: projectID,
		Code:      code,
		Name:      name,
		Engine:    "go_html",
		Body:      body,
		Remark:    strings.TrimSpace(req.Remark),
		Status:    1,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) UpdateReportTemplate(ctx context.Context, projectID, tid uint, req ReportTemplateUpsertRequest) (*model.InspectReportTemplate, error) {
	var row model.InspectReportTemplate
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", tid, projectID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("模板不存在或不可修改全局内置模板")
		}
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if req.Body != "" {
		if _, err := renderHTMLWithTemplate(row.Code, req.Body, ReportData{Project: "preview", Grade: "A"}); err != nil {
			return nil, constants.ErrBadRequestWithMsg("模板解析失败: " + err.Error())
		}
		row.Body = req.Body
	}
	row.Remark = strings.TrimSpace(req.Remark)
	if req.Status != nil {
		row.Status = *req.Status
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) DeleteReportTemplate(ctx context.Context, projectID, tid uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ? AND is_builtin = ?", tid, projectID, false).
		Delete(&model.InspectReportTemplate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("模板不存在或不可删除")
	}
	return nil
}

// CopyReportTemplate 将全局/其它模板复制为项目模板。
func (s *Service) CopyReportTemplate(ctx context.Context, projectID, sourceID uint, newCode, newName string) (*model.InspectReportTemplate, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	var src model.InspectReportTemplate
	err := s.db.WithContext(ctx).
		Where("id = ? AND (project_id = 0 OR project_id = ?)", sourceID, projectID).
		First(&src).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("源模板不存在")
		}
		return nil, err
	}
	code := strings.TrimSpace(newCode)
	if code == "" {
		code = src.Code + "-copy"
	}
	name := strings.TrimSpace(newName)
	if name == "" {
		name = src.Name + "（副本）"
	}
	body := strings.TrimSpace(src.Body)
	if body == "" {
		// 从 embed 读出作为可编辑正文
		b, err := reportFS.ReadFile(builtinTemplateFile(src.Code))
		if err != nil {
			return nil, err
		}
		body = string(b)
	}
	row := model.InspectReportTemplate{
		ProjectID: projectID,
		Code:      code,
		Name:      name,
		Engine:    "go_html",
		Body:      body,
		Remark:    "复制自 " + src.Code,
		Status:    1,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

type ReportPreviewRequest struct {
	TemplateID uint   `json:"template_id"`
	Code       string `json:"code"`
	Body       string `json:"body"`
}

func (s *Service) PreviewReportTemplate(ctx context.Context, projectID uint, req ReportPreviewRequest) ([]byte, error) {
	body := strings.TrimSpace(req.Body)
	code := strings.TrimSpace(req.Code)
	if body == "" && req.TemplateID > 0 {
		tpl, err := s.resolveReportTemplate(ctx, projectID, req.TemplateID)
		if err != nil {
			return nil, err
		}
		body = tpl.Body
		code = tpl.Code
	}
	if code == "" {
		code = "default"
	}
	data := ReportData{
		Timestamp:      time.Now(),
		Project:        "预览项目",
		Datasource:     "prometheus-demo",
		InspectionUser: "preview",
		Score:          86,
		Grade:          "B",
		Summary:        "共 12 条样本：严重 1、警告 2、正常 9。健康分 86（B）。",
		ReportListMode: "abnormal_only",
		ReportListHint: "预览数据，非真实巡检结果",
		Total:          12,
		Critical:       1,
		Warning:        2,
		Normal:         9,
		Findings: []Finding{
			{Type: "基础设施层", Name: "CPU 使用率", Severity: "critical", Count: 1, Hint: "建议排查负载"},
			{Type: "数据库监控", Name: "连接数", Severity: "warning", Count: 1, Hint: "关注连接池"},
		},
		Groups: []ReportGroup{
			{
				Type:  "基础设施层",
				Stats: GroupStats{Total: 2, Critical: 1, Warning: 0, Normal: 1},
				Metrics: []MetricSample{
					{Type: "基础设施层", Name: "CPU 使用率", Instance: "10.0.0.1", Value: 92, Threshold: 85, Unit: "%", Status: "critical"},
					{Type: "基础设施层", Name: "内存使用率", Instance: "10.0.0.1", Value: 60, Threshold: 85, Unit: "%", Status: "normal"},
				},
			},
		},
	}
	return renderHTMLWithTemplate(code, body, data)
}
