package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gopkg.in/yaml.v3"
)

// ImportPrometheusRulesRequest 从 Prometheus rule YAML 导入平台监控规则。
type ImportPrometheusRulesRequest struct {
	DatasourceID uint   `json:"datasource_id" binding:"required"`
	ProjectID    *uint  `json:"project_id"`
	YAML         string `json:"yaml" binding:"required"`
	Enabled      *bool  `json:"enabled"`
	// DryRun 仅解析返回预览，不写库
	DryRun bool `json:"dry_run"`
}

type ImportPrometheusRulePreview struct {
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	Expr      string `json:"expr"`
	ForSeconds int   `json:"for_seconds"`
	Severity  string `json:"severity"`
}

type ImportPrometheusRulesResult struct {
	Created int                          `json:"created"`
	Skipped int                          `json:"skipped"`
	Preview []ImportPrometheusRulePreview `json:"preview,omitempty"`
	Errors  []string                     `json:"errors,omitempty"`
}

type promRuleFile struct {
	Groups []promRuleGroup `yaml:"groups"`
}

type promRuleGroup struct {
	Name  string     `yaml:"name"`
	Rules []promRule `yaml:"rules"`
}

type promRule struct {
	Alert       string            `yaml:"alert"`
	Record      string            `yaml:"record"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

func parsePrometheusForDuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	sec := int(d.Seconds())
	if sec < 0 {
		return 0
	}
	return sec
}

func (s *AlertMonitorRuleService) ImportPrometheusYAML(ctx context.Context, req ImportPrometheusRulesRequest) (*ImportPrometheusRulesResult, error) {
	ds, err := s.dsRepo.GetByID(ctx, req.DatasourceID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.rule", "ImportPrometheusYAML.ds", err)
	}
	var file promRuleFile
	if err := yaml.Unmarshal([]byte(req.YAML), &file); err != nil {
		return nil, constants.ErrBadRequestWithMsg("YAML 解析失败: " + err.Error())
	}
	if len(file.Groups) == 0 {
		return nil, constants.ErrBadRequestWithMsg("YAML 中无 groups")
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	_ = ds

	out := &ImportPrometheusRulesResult{Preview: make([]ImportPrometheusRulePreview, 0)}
	for _, g := range file.Groups {
		for _, r := range g.Rules {
			name := strings.TrimSpace(r.Alert)
			if name == "" {
				out.Skipped++
				continue // recording rules
			}
			expr := strings.TrimSpace(r.Expr)
			if expr == "" {
				out.Errors = append(out.Errors, fmt.Sprintf("%s/%s: expr 为空", g.Name, name))
				out.Skipped++
				continue
			}
			sev := "warning"
			if r.Labels != nil {
				if v := strings.TrimSpace(r.Labels["severity"]); v != "" {
					sev = v
				}
			}
			forSec := parsePrometheusForDuration(r.For)
			prev := ImportPrometheusRulePreview{
				GroupName:  g.Name,
				Name:       name,
				Expr:       expr,
				ForSeconds: forSec,
				Severity:   sev,
			}
			out.Preview = append(out.Preview, prev)
			if req.DryRun {
				continue
			}
			labels := map[string]string{}
			for k, v := range r.Labels {
				labels[k] = v
			}
			if g.Name != "" {
				labels["rule_group"] = g.Name
			}
			if req.ProjectID != nil && *req.ProjectID > 0 {
				labels["project_id"] = fmt.Sprintf("%d", *req.ProjectID)
			}
			labelsJSON, _ := json.Marshal(labels)
			annJSON, _ := json.Marshal(r.Annotations)
			row := &model.AlertMonitorRule{
				DatasourceID:        req.DatasourceID,
				Name:                name,
				Expr:                expr,
				ForSeconds:          forSec,
				EvalIntervalSeconds: 30,
				Severity:            sev,
				LabelsJSON:          string(labelsJSON),
				AnnotationsJSON:     string(annJSON),
				Enabled:             enabled,
			}
			if err := s.ruleRepo.Create(ctx, row); err != nil {
				out.Errors = append(out.Errors, fmt.Sprintf("%s/%s: %v", g.Name, name, err))
				out.Skipped++
				continue
			}
			out.Created++
		}
	}
	return out, nil
}
