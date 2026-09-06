package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// PromoteAlertRequest 巡检项 → 持续告警规则。
type PromoteAlertRequest struct {
	DatasourceID        uint   `json:"datasource_id"`
	ForSeconds          int    `json:"for_seconds"`
	EvalIntervalSeconds int    `json:"eval_interval_seconds"`
	Severity            string `json:"severity"`
	Enabled             *bool  `json:"enabled"`
}

// PromoteItemToAlert 将项目巡检项转为 alert_monitor_rules（幂等：已关联则更新 Expr）。
func (s *Service) PromoteItemToAlert(ctx context.Context, projectID, itemID uint, req PromoteAlertRequest) (*model.AlertMonitorRule, error) {
	if projectID == 0 || itemID == 0 {
		return nil, constants.ErrBadRequestWithMsg("project_id and item_id required")
	}
	var item model.InspectItem
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", itemID, projectID).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("巡检项不存在（请先同步模板到项目）")
		}
		return nil, err
	}
	dsID := req.DatasourceID
	if dsID == 0 {
		plan, err := s.GetOrCreatePlan(ctx, projectID)
		if err != nil {
			return nil, err
		}
		dsID = plan.DatasourceID
	}
	if dsID == 0 {
		return nil, constants.ErrBadRequestWithMsg("请指定 datasource_id 或在巡检计划中配置数据源")
	}
	if _, ds, err := s.dsSvc.PrometheusClient(ctx, dsID); err != nil {
		return nil, err
	} else if ds.ProjectID != projectID {
		return nil, constants.ErrBadRequestWithMsg("数据源不属于当前项目")
	}

	expr, err := buildInspectAlertExpr(item.Query, item.Threshold, item.ThresholdType)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	sev := strings.TrimSpace(req.Severity)
	if sev == "" {
		sev = "warning"
	}
	ev := req.EvalIntervalSeconds
	if ev <= 0 {
		ev = 60
	}
	if ev < 5 {
		ev = 5
	}
	forSec := req.ForSeconds
	if forSec < 0 {
		forSec = 0
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	unit := strings.TrimSpace(item.Unit)
	if unit == "" {
		unit = "raw"
	}
	labelsJSON := mergeInspectPromoteLabels(item.LabelsJSON, item.ID)
	annJSON := fmt.Sprintf(
		`{"summary":"巡检项转告警: %s","description":"阈值 %s %g；来源巡检项 #%d"}`,
		escapeJSONString(item.Name),
		escapeJSONString(item.ThresholdType),
		item.Threshold,
		item.ID,
	)

	var rule model.AlertMonitorRule
	if item.LinkedRuleID > 0 {
		if err := s.db.WithContext(ctx).First(&rule, item.LinkedRuleID).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}
			item.LinkedRuleID = 0
		}
	}
	if item.LinkedRuleID > 0 {
		rule.DatasourceID = dsID
		rule.ProjectID = projectID
		rule.Name = promoteRuleName(item.Name)
		rule.RuleKind = model.AlertRuleKindPromQL
		rule.Expr = expr
		rule.ForSeconds = forSec
		rule.EvalIntervalSeconds = ev
		rule.Severity = sev
		rule.ThresholdUnit = unit
		rule.LabelsJSON = labelsJSON
		rule.AnnotationsJSON = annJSON
		rule.Enabled = enabled
		rule.Origin = model.AlertRuleOriginInspect
		rule.OriginInspectItemID = item.ID
		if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
			return nil, err
		}
		return &rule, nil
	}

	rule = model.AlertMonitorRule{
		DatasourceID:         dsID,
		ProjectID:            projectID,
		Name:                 promoteRuleName(item.Name),
		RuleKind:             model.AlertRuleKindPromQL,
		Expr:                 expr,
		ForSeconds:           forSec,
		EvalIntervalSeconds:  ev,
		Severity:             sev,
		ThresholdUnit:        unit,
		LabelsJSON:           labelsJSON,
		AnnotationsJSON:      annJSON,
		Enabled:              enabled,
		Origin:               model.AlertRuleOriginInspect,
		OriginInspectItemID:  item.ID,
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&rule).Error; err != nil {
			return err
		}
		return tx.Model(&model.InspectItem{}).Where("id = ?", item.ID).
			Update("linked_rule_id", rule.ID).Error
	}); err != nil {
		return nil, err
	}
	item.LinkedRuleID = rule.ID
	return &rule, nil
}

func promoteRuleName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "[巡检] unnamed"
	}
	if strings.HasPrefix(name, "[巡检]") {
		return name
	}
	return "[巡检] " + name
}

func buildInspectAlertExpr(query string, threshold float64, thresholdType string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", fmt.Errorf("empty query")
	}
	// 已是比较表达式则原样使用
	if strings.ContainsAny(q, "<>=") && !strings.HasPrefix(strings.ToLower(q), "count(") {
		// 粗略：含比较符时仍包一层阈值，避免双重；若用户已写比较则直接返回
		low := strings.ToLower(q)
		if strings.Contains(low, " > ") || strings.Contains(low, " < ") ||
			strings.Contains(low, ">=") || strings.Contains(low, "<=") ||
			strings.Contains(low, "==") || strings.Contains(low, "!=") {
			return q, nil
		}
	}
	th := strconv.FormatFloat(threshold, 'f', -1, 64)
	inner := "(" + q + ")"
	tt := strings.ToLower(strings.TrimSpace(thresholdType))
	switch tt {
	case "", "greater", ">":
		return inner + " > " + th, nil
	case "greater_equal", ">=":
		return inner + " >= " + th, nil
	case "less", "<":
		return inner + " < " + th, nil
	case "less_equal", "<=":
		return inner + " <= " + th, nil
	case "equal", "=":
		return inner + " == " + th, nil
	case "not_equal", "!=":
		return inner + " != " + th, nil
	default:
		return "", fmt.Errorf("不支持的 threshold_type: %s", thresholdType)
	}
}

func mergeInspectPromoteLabels(raw string, itemID uint) string {
	m := map[string]string{
		"origin":           model.AlertRuleOriginInspect,
		"inspect_item_id":  strconv.FormatUint(uint64(itemID), 10),
	}
	raw = strings.TrimSpace(raw)
	if raw != "" && raw != "{}" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(raw), &obj); err == nil {
			for k, v := range obj {
				m[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return `{"origin":"inspect"}`
	}
	return string(b)
}

func escapeJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	// Marshal 带引号，去掉首尾
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}
