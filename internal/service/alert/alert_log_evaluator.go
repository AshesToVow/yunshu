package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/service/logplatform"
)

// LogAlertConfig 日志告警规则配置（存于 AlertMonitorRule.Expr JSON）。
type LogAlertConfig struct {
	Mode           string `json:"mode"` // error_count | error_spike | pattern_rate
	Level          string `json:"level"`
	Keyword        string `json:"keyword"`
	ServiceName    string `json:"service_name"`
	Namespace      string `json:"namespace"`
	WindowMinutes  int    `json:"window_minutes"`
	Threshold      int64  `json:"threshold"`
	BaselineMinutes int   `json:"baseline_minutes"` // error_spike: 基线窗口
	SpikeRatio     float64 `json:"spike_ratio"`     // error_spike: 当前/基线倍率，默认 3
}

func normalizeRuleKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case model.AlertRuleKindLog:
		return model.AlertRuleKindLog
	case model.AlertRuleKindSLO:
		return model.AlertRuleKindSLO
	default:
		return model.AlertRuleKindPromQL
	}
}

func parseLogAlertConfig(expr string) (LogAlertConfig, error) {
	var cfg LogAlertConfig
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return cfg, fmt.Errorf("log alert config empty")
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, err
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = "error_count"
	}
	if cfg.WindowMinutes <= 0 {
		cfg.WindowMinutes = 5
	}
	if cfg.WindowMinutes > 120 {
		cfg.WindowMinutes = 120
	}
	if cfg.Level == "" && (cfg.Mode == "error_count" || cfg.Mode == "error_spike") {
		cfg.Level = "ERROR"
	}
	if cfg.Threshold <= 0 {
		cfg.Threshold = 10
	}
	if cfg.BaselineMinutes <= 0 {
		cfg.BaselineMinutes = cfg.WindowMinutes * 6
	}
	if cfg.SpikeRatio <= 0 {
		cfg.SpikeRatio = 3
	}
	return cfg, nil
}

func (s *AlertService) evaluateOneLogAlertRule(ctx context.Context, rule *model.AlertMonitorRule, projectID uint) {
	if s == nil || rule == nil || s.logSearch == nil || projectID == 0 {
		return
	}
	cfg, err := parseLogAlertConfig(rule.Expr)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	curFrom := now.Add(-time.Duration(cfg.WindowMinutes) * time.Minute)
	curQ := logplatform.LogSearchQuery{
		ProjectID:   projectID,
		Level:       cfg.Level,
		Keyword:     cfg.Keyword,
		ServiceName: cfg.ServiceName,
		Namespace:   cfg.Namespace,
		From:        curFrom.Format(time.RFC3339),
		To:          now.Format(time.RFC3339),
		Page:        1,
		PageSize:    1,
	}
	ov, err := s.logSearch.Overview(ctx, curQ)
	if err != nil || ov == nil {
		return
	}
	curCount := ov.Total
	firing := false
	value := strconv.FormatInt(curCount, 10)
	switch cfg.Mode {
	case "error_spike":
		baseFrom := curFrom.Add(-time.Duration(cfg.BaselineMinutes) * time.Minute)
		baseQ := curQ
		baseQ.From = baseFrom.Format(time.RFC3339)
		baseQ.To = curFrom.Format(time.RFC3339)
		baseOv, err := s.logSearch.Overview(ctx, baseQ)
		baseCount := int64(0)
		if err == nil && baseOv != nil {
			baseCount = baseOv.Total
		}
		ratio := cfg.SpikeRatio
		if baseCount <= 0 {
			firing = curCount >= cfg.Threshold
		} else {
			firing = float64(curCount) >= float64(baseCount)*ratio && curCount >= cfg.Threshold
		}
		value = fmt.Sprintf("cur=%d base=%d", curCount, baseCount)
	case "pattern_rate", "error_count":
		fallthrough
	default:
		firing = curCount >= cfg.Threshold
	}

	labels := buildMonitorRuleLabels(rule, projectID, nil)
	labels["rule_kind"] = model.AlertRuleKindLog
	labels["log_mode"] = cfg.Mode
	if cfg.ServiceName != "" {
		labels["service_name"] = cfg.ServiceName
	}
	if cfg.Namespace != "" {
		labels["namespace"] = cfg.Namespace
	}
	annotations := buildMonitorRuleAnnotations(rule, labels, value)
	annotations["log_count"] = strconv.FormatInt(curCount, 10)
	annotations["log_threshold"] = strconv.FormatInt(cfg.Threshold, 10)
	fp := monitorSeriesFingerprint(rule.ID, labels)

	if s.redis == nil {
		s.evaluateMonitorRuleNoRedis(ctx, rule, firing, labels, annotations, fp, now)
		return
	}
	s.evaluateMonitorRuleWithRedis(ctx, rule, firing, labels, annotations, fp, now)
}
