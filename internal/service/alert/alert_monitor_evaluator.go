package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	"yunshu/internal/model"
	"yunshu/internal/pkg/cronutil"
	"yunshu/internal/pkg/promapi"
)

func (s *AlertService) runMonitorRuleEvaluator(ctx context.Context) {
	spec := strings.TrimSpace(s.cfg.MonitorEvalCronSpec)
	if spec == "" {
		spec = "*/5 * * * * *"
	}
	cronutil.RunWorker(ctx, spec, func() {
		_ = s.tickMonitorRules(ctx)
	}, "*/5 * * * * *")
}

func (s *AlertService) tickMonitorRules(ctx context.Context) error {
	if s.redis != nil {
		ttlSec := s.cfg.MonitorEvalLeaderLockSeconds
		if ttlSec <= 0 {
			ttlSec = 30
		}
		ok, err := s.redis.SetNX(ctx, "alert:monitor:eval:leader", "1", time.Duration(ttlSec)*time.Second).Result()
		if err != nil || !ok {
			return nil
		}
	}
	rules, err := s.monitorRuleRepo.ListEnabledWithProject(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.evaluator", "tickMonitorRules", err)
	}
	now := time.Now()
	for i := range rules {
		rule := &rules[i]
		if s.redis != nil {
			if !s.shouldEvalRuleRedis(ctx, rule.ID, rule.EvalIntervalSeconds, now) {
				continue
			}
			lockSec := min(rule.EvalIntervalSeconds, 120)
			if lockSec < 15 {
				lockSec = 15
			}
			if !s.monitorEvalLockAcquire(ctx, rule.ID, lockSec) {
				continue
			}
			func(r *repository.EvalMonitorRule) {
				defer s.monitorEvalLockRelease(ctx, r.ID)
				s.dispatchMonitorRuleEval(ctx, &r.AlertMonitorRule, r.ProjectID)
			}(rule)
			continue
		}
		if !s.shouldEvalRuleNoRedis(rule.ID, rule.EvalIntervalSeconds, now) {
			continue
		}
		s.dispatchMonitorRuleEval(ctx, &rule.AlertMonitorRule, rule.ProjectID)
	}
	return nil
}

func (s *AlertService) dispatchMonitorRuleEval(ctx context.Context, rule *model.AlertMonitorRule, projectID uint) {
	if rule == nil {
		return
	}
	switch normalizeRuleKind(rule.RuleKind) {
	case model.AlertRuleKindLog:
		s.evaluateOneLogAlertRule(ctx, rule, projectID)
	default:
		// promql / slo 均走 PromQL 评估
		s.evaluateOneMonitorRule(ctx, rule, projectID)
	}
}

func buildMonitorRuleLabels(rule *model.AlertMonitorRule, projectID uint, ds *model.AlertDatasource) map[string]string {
	labels := map[string]string{
		"alertname":       rule.Name,
		"severity":        strings.TrimSpace(rule.Severity),
		"monitor_rule_id": fmt.Sprintf("%d", rule.ID),
		"datasource_id":   fmt.Sprintf("%d", rule.DatasourceID),
		"project_id":      fmt.Sprintf("%d", projectID),
		"source":          "prometheus_monitor",
	}
	if ds != nil {
		if n := strings.TrimSpace(ds.Name); n != "" {
			labels["datasource_name"] = n
		}
		typ := strings.TrimSpace(ds.Type)
		if typ == "" {
			typ = "prometheus"
		}
		if typ == "victoriametrics" {
			typ = "victoria"
		}
		labels["datasource_type"] = typ
	}
	if strings.TrimSpace(rule.Severity) == "" {
		labels["severity"] = "warning"
	}
	raw := strings.TrimSpace(rule.LabelsJSON)
	if raw != "" && raw != "{}" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for k, v := range obj {
				labels[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return labels
}

func renderRuleAnnotationTemplate(tpl string, labels map[string]string, value string, rule *model.AlertMonitorRule) string {
	out := strings.TrimSpace(tpl)
	if out == "" {
		return ""
	}
	re := regexp.MustCompile(`\{\{\s*\$labels\.([a-zA-Z0-9_]+)\s*\}\}`)
	out = re.ReplaceAllStringFunc(out, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if len(sub) != 2 {
			return ""
		}
		return strings.TrimSpace(labels[sub[1]])
	})
	out = strings.ReplaceAll(out, "{{$value}}", strings.TrimSpace(value))
	if rule != nil {
		out = strings.ReplaceAll(out, "{{.RuleName}}", strings.TrimSpace(rule.Name))
		out = strings.ReplaceAll(out, "{{.Expr}}", strings.TrimSpace(rule.Expr))
	}
	return out
}

func buildMonitorRuleAnnotations(rule *model.AlertMonitorRule, labels map[string]string, value string) map[string]string {
	defaultSummary := fmt.Sprintf("监控规则 %s 触发", rule.Name)
	defaultDescription := fmt.Sprintf("PromQL: %s", rule.Expr)
	ann := map[string]string{
		"summary":     defaultSummary,
		"description": defaultDescription,
	}
	raw := strings.TrimSpace(rule.AnnotationsJSON)
	if raw != "" && raw != "{}" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for k, v := range obj {
				ann[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	ann["summary"] = renderRuleAnnotationTemplate(ann["summary"], labels, value, rule)
	ann["description"] = renderRuleAnnotationTemplate(ann["description"], labels, value, rule)
	if strings.TrimSpace(ann["summary"]) == "" {
		ann["summary"] = renderRuleAnnotationTemplate(defaultSummary, labels, value, rule)
	}
	if strings.TrimSpace(ann["description"]) == "" {
		ann["description"] = renderRuleAnnotationTemplate(defaultDescription, labels, value, rule)
	}
	return ann
}

func parsePromVectorSamples(body []byte) []struct {
	Metric map[string]string
	Value  string
} {
	var wrap struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil
	}
	if wrap.Status != "success" || wrap.Data.ResultType != "vector" || len(wrap.Data.Result) == 0 {
		return nil
	}
	out := make([]struct {
		Metric map[string]string
		Value  string
	}, 0, len(wrap.Data.Result))
	for _, row := range wrap.Data.Result {
		value := ""
		if len(row.Value) >= 2 {
			value = strings.TrimSpace(fmt.Sprintf("%v", row.Value[1]))
		}
		metric := row.Metric
		if metric == nil {
			metric = map[string]string{}
		}
		out = append(out, struct {
			Metric map[string]string
			Value  string
		}{Metric: metric, Value: value})
	}
	return out
}

func parsePromFirstSample(body []byte) (map[string]string, string) {
	samples := parsePromVectorSamples(body)
	if len(samples) == 0 {
		return map[string]string{}, ""
	}
	return samples[0].Metric, samples[0].Value
}

func (s *AlertService) evaluateOneMonitorRule(ctx context.Context, rule *model.AlertMonitorRule, projectID uint) {
	if rule == nil {
		return
	}
	if normalizeRuleKind(rule.RuleKind) == model.AlertRuleKindLog {
		return
	}
	ds, err := s.datasourceRepo.GetByID(ctx, rule.DatasourceID)
	if err != nil {
		return
	}
	if !ds.Enabled || !isPromCompatibleDatasourceType(ds.Type) {
		return
	}
	if IsDatasourceHealthBlocking(ds, time.Now()) {
		slog.Default().With(
			"component", "alert.monitor_eval",
			"rule_id", rule.ID,
			"datasource_id", ds.ID,
		).Info("skip evaluate: datasource health down")
		return
	}
	cli := &promapi.Client{
		BaseURL:       strings.TrimRight(strings.TrimSpace(ds.BaseURL), "/"),
		BearerToken:   ds.BearerToken,
		BasicUser:     ds.BasicUser,
		BasicPassword: ds.BasicPassword,
		SkipTLSVerify: ds.SkipTLSVerify,
	}
	qctx, cancel := context.WithTimeout(ctx, time.Duration(maxInt(3, s.cfg.PromQueryTimeout))*time.Second)
	defer cancel()
	body, _, err := cli.QueryInstant(qctx, strings.TrimSpace(rule.Expr), "")
	if err != nil {
		return
	}
	if projectID == 0 {
		projectID = ds.ProjectID
	}
	now := time.Now()
	if s.redis != nil {
		s.redisTouchLastEval(ctx, rule.ID, now)
	}

	samples := parsePromVectorSamples(body)
	current := make(map[string]struct{}, len(samples))
	for _, sample := range samples {
		labels := buildMonitorRuleLabels(rule, projectID, ds)
		for k, v := range sample.Metric {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}
		annotations := buildMonitorRuleAnnotations(rule, labels, sample.Value)
		if v := strings.TrimSpace(sample.Value); v != "" {
			annotations["value"] = v
		}
		fp := monitorSeriesFingerprint(rule.ID, labels)
		current[fp] = struct{}{}
		if s.redis == nil {
			s.evaluateMonitorRuleNoRedis(ctx, rule, true, labels, annotations, fp, now)
			continue
		}
		s.evaluateMonitorRuleWithRedis(ctx, rule, true, labels, annotations, fp, now)
	}

	if s.redis != nil {
		for _, fp := range s.listTrackedMonitorSeries(ctx, rule.ID) {
			if _, ok := current[fp]; ok {
				continue
			}
			labels, annotations := s.loadMonitorSeriesPayload(ctx, rule.ID, fp)
			if len(labels) == 0 {
				labels = buildMonitorRuleLabels(rule, projectID, ds)
			}
			if len(annotations) == 0 {
				annotations = buildMonitorRuleAnnotations(rule, labels, "")
			}
			s.evaluateMonitorRuleWithRedis(ctx, rule, false, labels, annotations, fp, now)
		}
		return
	}

	// 无 Redis：对本次未出现的序列做 resolve（进程内 map）。
	s.monitorEvalMu.Lock()
	tracked := make([]string, 0)
	for fp := range s.monitorNoRedisActive {
		if strings.HasPrefix(fp, fmt.Sprintf("mr%d_", rule.ID)) {
			tracked = append(tracked, fp)
		}
	}
	s.monitorEvalMu.Unlock()
	for _, fp := range tracked {
		if _, ok := current[fp]; ok {
			continue
		}
		labels := buildMonitorRuleLabels(rule, projectID, ds)
		annotations := buildMonitorRuleAnnotations(rule, labels, "")
		s.evaluateMonitorRuleNoRedis(ctx, rule, false, labels, annotations, fp, now)
	}
}
