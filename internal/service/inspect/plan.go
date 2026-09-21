package inspect

// 巡检计划（inspect_plan）读写：定时表达式、数据源绑定、报告版式与收件人。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"

	"gorm.io/gorm"
)

type PlanUpsertRequest struct {
	Enabled          *bool    `json:"enabled"`
	CronSpec         string   `json:"cron_spec"`
	DatasourceID     uint     `json:"datasource_id"`
	ReportListMode   string   `json:"report_list_mode"`
	ReportTemplateID *uint    `json:"report_template_id"`
	RetainDays       *int     `json:"retain_days"`
	Recipients       []string `json:"recipients"`
}

func (s *Service) GetOrCreatePlan(ctx context.Context, projectID uint) (*model.InspectPlan, error) {
	if projectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id required")
	}
	plan, err := s.repo.GetPlanByProject(ctx, projectID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		plan = &model.InspectPlan{
			ProjectID:      projectID,
			Enabled:        false,
			CronSpec:       "0 0 9 * * *",
			ReportListMode: "abnormal_only",
			RetainDays:     90,
			RecipientsJSON: "[]",
		}
		if err := s.repo.CreatePlan(ctx, plan); err != nil {
			return nil, err
		}
	}
	_ = s.ensurePlanDefaults(ctx, plan)
	return plan, nil
}

// ensurePlanDefaults 首次进入自动绑定数据源、同步模板巡检项，减少手工配置。
func (s *Service) ensurePlanDefaults(ctx context.Context, plan *model.InspectPlan) error {
	if plan == nil {
		return nil
	}
	changed := false
	if plan.DatasourceID == 0 {
		ds, err := s.repo.FirstEnabledPrometheusDS(ctx, plan.ProjectID)
		if err == nil && ds != nil {
			plan.DatasourceID = ds.ID
			changed = true
		}
	}
	if changed {
		if err := s.repo.SavePlan(ctx, plan); err != nil {
			return err
		}
	}
	n, _ := s.repo.CountItemsByProject(ctx, plan.ProjectID)
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
		ds, err := s.repo.GetDatasource(ctx, req.DatasourceID)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("数据源不存在")
		}
		if ds.ProjectID != projectID {
			return nil, constants.ErrBadRequestWithMsg("数据源不属于当前项目")
		}
		if !strings.EqualFold(strings.TrimSpace(ds.Type), "prometheus") {
			return nil, constants.ErrBadRequestWithMsg("巡检仅支持 Prometheus 数据源")
		}
		if !ds.Enabled {
			return nil, constants.ErrBadRequestWithMsg("数据源未启用")
		}
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
	if req.ReportTemplateID != nil {
		tid := *req.ReportTemplateID
		if tid > 0 {
			if _, err := s.resolveReportTemplate(ctx, projectID, tid); err != nil {
				return nil, constants.ErrBadRequestWithMsg("报告模板不存在或未启用")
			}
		}
		plan.ReportTemplateID = tid
	}
	if req.RetainDays != nil {
		if *req.RetainDays < 0 {
			return nil, constants.ErrBadRequestWithMsg("retain_days 不能为负")
		}
		plan.RetainDays = *req.RetainDays
	}
	if req.Recipients != nil {
		b, _ := json.Marshal(uniqEmails(req.Recipients))
		plan.RecipientsJSON = string(b)
	}
	if plan.Enabled && plan.DatasourceID == 0 {
		return nil, constants.ErrBadRequestWithMsg("启用定时巡检前请先配置 datasource_id")
	}
	if err := s.repo.SavePlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *Service) getPlanByRun(ctx context.Context, run *model.InspectRun) (*model.InspectPlan, error) {
	if s == nil || s.repo == nil || run == nil || run.PlanID == 0 {
		return nil, constants.ErrBadRequestWithMsg("invalid run")
	}
	return s.repo.GetPlanByID(ctx, run.PlanID)
}
