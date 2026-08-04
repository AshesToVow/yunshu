package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// AlertRuleTemplate 内置告警规则模板（分组目录，非 DB 表）。
type AlertRuleTemplate struct {
	ID              string            `json:"id"`
	Group           string            `json:"group"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	ExprTemplate    string            `json:"expr_template"`
	ForSeconds      int               `json:"for_seconds"`
	EvalIntervalSec int               `json:"eval_interval_seconds"`
	Severity        string            `json:"severity"`
	ThresholdUnit   string            `json:"threshold_unit"`
	DefaultParams   map[string]string `json:"default_params"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
}

type CreateFromTemplateRequest struct {
	TemplateID   string            `json:"template_id" binding:"required"`
	DatasourceID uint              `json:"datasource_id" binding:"required"`
	Name         string            `json:"name"`
	Params       map[string]string `json:"params"`
	Enabled      *bool             `json:"enabled"`
}

// BuiltinAlertRuleTemplates 按 cpu/disk/memory/availability 分组的规则包。
func BuiltinAlertRuleTemplates() []AlertRuleTemplate {
	return []AlertRuleTemplate{
		{
			ID: "cpu-high", Group: "cpu", Name: "CPU 使用率过高",
			Description: "节点 CPU 使用率超过阈值（默认 85%）持续 5 分钟",
			ExprTemplate: `100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > {{threshold}}`,
			ForSeconds: 300, EvalIntervalSec: 30, Severity: "warning", ThresholdUnit: "percent",
			DefaultParams: map[string]string{"threshold": "85"},
			Labels:        map[string]string{"category": "cpu"},
			Annotations:   map[string]string{"summary": "CPU 使用率过高", "description": "instance {{ $labels.instance }} CPU > {{threshold}}%"},
		},
		{
			ID: "memory-high", Group: "memory", Name: "内存使用率过高",
			Description: "节点内存使用率超过阈值（默认 90%）",
			ExprTemplate: `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > {{threshold}}`,
			ForSeconds: 300, EvalIntervalSec: 30, Severity: "warning", ThresholdUnit: "percent",
			DefaultParams: map[string]string{"threshold": "90"},
			Labels:        map[string]string{"category": "memory"},
			Annotations:   map[string]string{"summary": "内存使用率过高"},
		},
		{
			ID: "disk-high", Group: "disk", Name: "磁盘使用率过高",
			Description: "根分区或其他挂载点磁盘使用率超过阈值（默认 85%）",
			ExprTemplate: `(1 - (node_filesystem_avail_bytes{fstype!~"tmpfs|overlay"} / node_filesystem_size_bytes{fstype!~"tmpfs|overlay"})) * 100 > {{threshold}}`,
			ForSeconds: 600, EvalIntervalSec: 60, Severity: "warning", ThresholdUnit: "percent",
			DefaultParams: map[string]string{"threshold": "85"},
			Labels:        map[string]string{"category": "disk"},
			Annotations:   map[string]string{"summary": "磁盘使用率过高"},
		},
		{
			ID: "disk-inode-high", Group: "disk", Name: "磁盘 inode 使用率过高",
			Description: "文件系统 inode 使用率超过阈值（默认 90%）",
			ExprTemplate: `(1 - (node_filesystem_files_free{fstype!~"tmpfs|overlay"} / node_filesystem_files{fstype!~"tmpfs|overlay"})) * 100 > {{threshold}}`,
			ForSeconds: 600, EvalIntervalSec: 60, Severity: "warning", ThresholdUnit: "percent",
			DefaultParams: map[string]string{"threshold": "90"},
			Labels:        map[string]string{"category": "disk"},
			Annotations:   map[string]string{"summary": "磁盘 inode 使用率过高"},
		},
		{
			ID: "instance-down", Group: "availability", Name: "实例不可达",
			Description: "up==0 持续 2 分钟",
			ExprTemplate: `up == 0`,
			ForSeconds: 120, EvalIntervalSec: 30, Severity: "critical", ThresholdUnit: "raw",
			DefaultParams: map[string]string{},
			Labels:        map[string]string{"category": "availability"},
			Annotations:   map[string]string{"summary": "实例不可达"},
		},
		{
			ID: "http-error-rate", Group: "availability", Name: "HTTP 5xx 错误率过高",
			Description: "5xx 占比超过阈值（需 http 请求指标）",
			ExprTemplate: `sum(rate(http_requests_total{code=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100 > {{threshold}}`,
			ForSeconds: 300, EvalIntervalSec: 30, Severity: "warning", ThresholdUnit: "percent",
			DefaultParams: map[string]string{"threshold": "5"},
			Labels:        map[string]string{"category": "availability"},
			Annotations:   map[string]string{"summary": "HTTP 5xx 错误率过高"},
		},
	}
}

func FindAlertRuleTemplate(id string) *AlertRuleTemplate {
	id = strings.TrimSpace(id)
	for _, t := range BuiltinAlertRuleTemplates() {
		if t.ID == id {
			cp := t
			return &cp
		}
	}
	return nil
}

func ListAlertRuleTemplates(group string) []AlertRuleTemplate {
	group = strings.ToLower(strings.TrimSpace(group))
	all := BuiltinAlertRuleTemplates()
	if group == "" {
		return all
	}
	out := make([]AlertRuleTemplate, 0, len(all))
	for _, t := range all {
		if strings.EqualFold(t.Group, group) {
			out = append(out, t)
		}
	}
	return out
}

func renderTemplateParams(tpl string, params map[string]string) string {
	out := tpl
	for k, v := range params {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	return out
}

func (s *AlertMonitorRuleService) CreateFromTemplate(ctx context.Context, req CreateFromTemplateRequest) (*model.AlertMonitorRule, error) {
	tpl := FindAlertRuleTemplate(req.TemplateID)
	if tpl == nil {
		return nil, constants.ErrBadRequestWithMsg("未知规则模板: " + req.TemplateID)
	}
	params := map[string]string{}
	for k, v := range tpl.DefaultParams {
		params[k] = v
	}
	for k, v := range req.Params {
		if strings.TrimSpace(v) != "" {
			params[k] = strings.TrimSpace(v)
		}
	}
	expr := renderTemplateParams(tpl.ExprTemplate, params)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = tpl.Name
		if th, ok := params["threshold"]; ok {
			name = fmt.Sprintf("%s (>%s)", tpl.Name, th)
		}
	}
	labels := map[string]string{}
	for k, v := range tpl.Labels {
		labels[k] = v
	}
	labels["template_id"] = tpl.ID
	ann := map[string]string{}
	for k, v := range tpl.Annotations {
		ann[k] = renderTemplateParams(v, params)
	}
	labelsJSON, _ := json.Marshal(labels)
	annJSON, _ := json.Marshal(ann)
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	upsert := AlertMonitorRuleUpsertRequest{
		DatasourceID:        req.DatasourceID,
		Name:                name,
		Expr:                expr,
		ForSeconds:          tpl.ForSeconds,
		EvalIntervalSeconds: tpl.EvalIntervalSec,
		Severity:            tpl.Severity,
		ThresholdUnit:       tpl.ThresholdUnit,
		LabelsJSON:          string(labelsJSON),
		AnnotationsJSON:     string(annJSON),
		Enabled:             &enabled,
	}
	return s.Create(ctx, upsert)
}
