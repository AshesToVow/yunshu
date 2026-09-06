package logplatform

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// LogDropRuleUpsert 创建/更新黑名单。
type LogDropRuleUpsert struct {
	Name     string `json:"name"`
	Enabled  *bool  `json:"enabled"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Remark   string `json:"remark"`
}

// LogDropRuleService 日志黑名单 CRUD。
type LogDropRuleService struct {
	db *gorm.DB
}

func NewLogDropRuleService(db *gorm.DB) *LogDropRuleService {
	return &LogDropRuleService{db: db}
}

func (s *ClusterLogService) DropRules() *LogDropRuleService {
	return NewLogDropRuleService(s.db)
}

func (s *LogDropRuleService) List(ctx context.Context, projectID uint) ([]model.LogDropRule, error) {
	if projectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	var list []model.LogDropRule
	err := s.db.WithContext(ctx).Where("project_id = ?", projectID).
		Order("enabled DESC, id DESC").Find(&list).Error
	return list, err
}

func (s *LogDropRuleService) Create(ctx context.Context, projectID, userID uint, req LogDropRuleUpsert) (*model.LogDropRule, error) {
	row, err := normalizeDropRule(projectID, userID, req)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (s *LogDropRuleService) Update(ctx context.Context, projectID, ruleID uint, req LogDropRuleUpsert) (*model.LogDropRule, error) {
	var row model.LogDropRule
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).First(&row).Error; err != nil {
		return nil, constants.ErrNotFoundWithMsg("黑名单规则不存在")
	}
	norm, err := normalizeDropRule(projectID, row.CreatedBy, req)
	if err != nil {
		return nil, err
	}
	row.Name = norm.Name
	row.Field = norm.Field
	row.Operator = norm.Operator
	row.Value = norm.Value
	row.Remark = norm.Remark
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *LogDropRuleService) Delete(ctx context.Context, projectID, ruleID uint) error {
	res := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", ruleID, projectID).Delete(&model.LogDropRule{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFoundWithMsg("黑名单规则不存在")
	}
	return nil
}

func (s *LogDropRuleService) ListEnabled(ctx context.Context, projectID uint) ([]model.LogDropRule, error) {
	if projectID == 0 || s.db == nil {
		return nil, nil
	}
	var list []model.LogDropRule
	err := s.db.WithContext(ctx).Where("project_id = ? AND enabled = ?", projectID, true).Find(&list).Error
	return list, err
}

func normalizeDropRule(projectID, userID uint, req LogDropRuleUpsert) (*model.LogDropRule, error) {
	if projectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	name := strings.TrimSpace(req.Name)
	field := strings.TrimSpace(req.Field)
	value := strings.TrimSpace(req.Value)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("名称不能为空")
	}
	if field == "" || value == "" {
		return nil, constants.ErrBadRequestWithMsg("field/value 必填")
	}
	op := strings.ToLower(strings.TrimSpace(req.Operator))
	if op == "" {
		op = model.LogDropOpEq
	}
	if op != model.LogDropOpEq && op != model.LogDropOpContains {
		return nil, constants.ErrBadRequestWithMsg("operator 仅支持 eq|contains")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	return &model.LogDropRule{
		ProjectID: projectID,
		Name:      name,
		Enabled:   enabled,
		Field:     field,
		Operator:  op,
		Value:     value,
		Remark:    strings.TrimSpace(req.Remark),
		CreatedBy: userID,
	}, nil
}

func dropRulesToMustNot(rules []model.LogDropRule) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, r := range rules {
		if clause := dropRuleClause(r); clause != nil {
			out = append(out, clause)
		}
	}
	return out
}

func dropRuleClause(r model.LogDropRule) map[string]any {
	field := strings.TrimSpace(r.Field)
	value := strings.TrimSpace(r.Value)
	if field == "" || value == "" {
		return nil
	}
	candidates := dropFieldCandidates(field)
	op := strings.ToLower(strings.TrimSpace(r.Operator))
	if op == model.LogDropOpContains {
		should := make([]map[string]any, 0, len(candidates))
		for _, f := range candidates {
			should = append(should, map[string]any{
				"wildcard": map[string]any{
					f: map[string]any{"value": "*" + value + "*", "case_insensitive": true},
				},
			})
		}
		return map[string]any{"bool": map[string]any{"should": should, "minimum_should_match": 1}}
	}
	return multiFieldTermFilter(candidates, value)
}

func dropFieldCandidates(field string) []string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "level", "status":
		return []string{"level", "level.keyword", "fields.level", "fields.level.keyword"}
	case "service", "service_name":
		return []string{"service_name", "service_name.keyword", "fields.service_name"}
	case "host", "hostname", "server_host":
		return []string{"host", "host.keyword", "server_host", "hostname"}
	case "pod", "podname":
		return []string{"pod", "podname", "pod.keyword", "podname.keyword"}
	case "namespace", "ns":
		return []string{"namespace", "namespace.keyword"}
	case "container", "containername":
		return []string{"container", "containername", "container.keyword"}
	case "message", "msg":
		return []string{"message", "msg", "log"}
	case "signature":
		return []string{"signature", "signature.keyword"}
	default:
		f := strings.TrimSpace(field)
		return []string{f, f + ".keyword", "fields." + f, "fields." + f + ".keyword"}
	}
}
