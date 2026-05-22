package eventforward

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type K8sEventForwardAdminService struct {
	repo interfaces.K8sEventForwardRepository
}

func NewK8sEventForwardAdminService(repo interfaces.K8sEventForwardRepository) *K8sEventForwardAdminService {
	return &K8sEventForwardAdminService{repo: repo}
}

type K8sEventForwardRuleListQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// K8sEventForwardRuleUpsertRequest 规则创建/更新请求（避免 Save 覆盖 created_at 为零值）。
type K8sEventForwardRuleUpsertRequest struct {
	Name           string `json:"name" binding:"required,max=100"`
	Description    string `json:"description"`
	ClusterIDs     string `json:"cluster_ids" binding:"required"`
	WebhookURL     string `json:"webhook_url"`
	Enabled        *bool  `json:"enabled"`
	RuleNamespaces string `json:"rule_namespaces"`
	RuleNames      string `json:"rule_names"`
	RuleReasons    string `json:"rule_reasons"`
	RuleReverse    *bool  `json:"rule_reverse"`
}

func (s *K8sEventForwardAdminService) ListRules(ctx context.Context, q K8sEventForwardRuleListQuery) (*pagination.Result[model.K8sEventForwardRule], error) {
	res, err := s.repo.ListRules(ctx, repository.K8sEventForwardRuleListFilter{Page: q.Page, PageSize: q.PageSize})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "ListRules", err)
	}
	return res, nil
}

func (s *K8sEventForwardAdminService) GetRule(ctx context.Context, id uint) (*model.K8sEventForwardRule, error) {
	rule, err := s.repo.GetRule(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "GetRule", err)
	}
	return rule, nil
}

func (s *K8sEventForwardAdminService) CreateRule(ctx context.Context, req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	rule, err := normalizeK8sEventForwardRule(req)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateRule(ctx, rule); err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "CreateRule", err)
	}
	return rule, nil
}

func (s *K8sEventForwardAdminService) UpdateRule(ctx context.Context, id uint, req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	existing, err := s.repo.GetRule(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "UpdateRule", err)
	}
	rule, err := normalizeK8sEventForwardRule(req)
	if err != nil {
		return nil, err
	}
	rule.ID = id
	rule.CreatedAt = existing.CreatedAt
	if err := s.repo.SaveRule(ctx, rule); err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "UpdateRule", err)
	}
	return s.repo.GetRule(ctx, id)
}

func (s *K8sEventForwardAdminService) DeleteRule(ctx context.Context, id uint) error {
	n, err := s.repo.DeleteRule(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "k8s.event-forward", "DeleteRule", err)
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *K8sEventForwardAdminService) GetSettings(ctx context.Context) (*model.K8sEventForwardSetting, error) {
	st, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.event-forward", "GetSettings", err)
	}
	return &st, nil
}

func (s *K8sEventForwardAdminService) UpdateSettings(ctx context.Context, st *model.K8sEventForwardSetting) error {
	if st == nil {
		return constants.ErrBadRequestWithMsg("参数无效")
	}
	if err := s.repo.SaveSettings(ctx, st); err != nil {
		return bizerrors.Pass(ctx, "k8s.event-forward", "UpdateSettings", err)
	}
	return nil
}

func normalizeK8sEventForwardRule(req K8sEventForwardRuleUpsertRequest) (*model.K8sEventForwardRule, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("规则名称不能为空")
	}
	clusterIDs := strings.TrimSpace(req.ClusterIDs)
	if clusterIDs == "" {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一个目标集群")
	}
	ns, err := normalizeJSONArrayField(req.RuleNamespaces, "rule_namespaces")
	if err != nil {
		return nil, err
	}
	names, err := normalizeJSONArrayField(req.RuleNames, "rule_names")
	if err != nil {
		return nil, err
	}
	reasons, err := normalizeJSONArrayField(req.RuleReasons, "rule_reasons")
	if err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	reverse := false
	if req.RuleReverse != nil {
		reverse = *req.RuleReverse
	}
	return &model.K8sEventForwardRule{
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		ClusterIDs:     clusterIDs,
		WebhookURL:     strings.TrimSpace(req.WebhookURL),
		Enabled:        enabled,
		RuleNamespaces: ns,
		RuleNames:      names,
		RuleReasons:    reasons,
		RuleReverse:    reverse,
	}, nil
}

func normalizeJSONArrayField(raw, field string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		s = "[]"
	}
	var arr []string
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return "", constants.ErrBadRequestWithMsg(field + " 须为 JSON 字符串数组，例如 [\"kube-system\"]")
	}
	out, err := json.Marshal(arr)
	if err != nil {
		return "", constants.ErrBadRequestWithMsg(field + " JSON 无效")
	}
	return string(out), nil
}
