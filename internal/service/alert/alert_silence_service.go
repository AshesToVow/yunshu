package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/service/changeevent"

	"gorm.io/gorm"
)

// SilenceMatcher 单条 matcher，语义参考 Alertmanager。
type SilenceMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"is_regex"`
}

type AlertSilenceListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type AlertSilenceUpsertRequest struct {
	ProjectID    uint      `json:"project_id"`
	Name         string    `json:"name" binding:"required,max=128"`
	MatchersJSON string    `json:"matchers_json" binding:"required"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	Comment      string    `json:"comment" binding:"omitempty,max=512"`
	Enabled      *bool     `json:"enabled"`
}

type AlertSilenceBatchItem struct {
	ProjectID    uint      `json:"project_id"` // 可选；非 0 时覆盖请求级 project_id
	Name         string    `json:"name" binding:"required,max=128"`
	MatchersJSON string    `json:"matchers_json" binding:"required"`
	StartsAt     time.Time `json:"starts_at" binding:"required"`
	EndsAt       time.Time `json:"ends_at" binding:"required"`
	Comment      string    `json:"comment" binding:"omitempty,max=512"`
	Enabled      *bool     `json:"enabled"`
}

type AlertSilenceBatchRequest struct {
	ProjectID uint                     `json:"project_id"` // 应用到所有 items；单项可覆盖
	Items     []AlertSilenceBatchItem `json:"items" binding:"required,min=1"`
}

type AlertSilenceService struct {
	repo       interfaces.AlertSilenceRepository
	memberRepo interfaces.ProjectMemberRepository
}

func NewAlertSilenceService(repo interfaces.AlertSilenceRepository, memberRepo interfaces.ProjectMemberRepository) *AlertSilenceService {
	return &AlertSilenceService{repo: repo, memberRepo: memberRepo}
}

func (s *AlertSilenceService) assertSilenceProjectAccess(ctx context.Context, actor *auth.CurrentUser, projectID uint) error {
	if projectID == 0 {
		// 全局静默仅超管
		if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
			return nil
		}
		return constants.ErrForbiddenWithMsg("全局静默（project_id=0）仅超级管理员可操作")
	}
	if actor != nil && auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil || actor == nil || actor.ID == 0 {
		return constants.ErrForbidden
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, projectID, actor.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrForbiddenWithMsg("非项目成员，无法操作该项目静默")
		}
		return err
	}
	_ = m
	return nil
}

// disableExpiredSilences 将已过期但仍启用的静默自动停用，避免 UI 显示“启用”造成误解。
// 这是轻量级的“读时修正”：不引入定时任务，也不影响未过期静默的正常流程。
func (s *AlertSilenceService) disableExpiredSilences(ctx context.Context, now time.Time) {
	if s == nil || s.repo == nil {
		return
	}
	_ = s.repo.DisableExpired(ctx, now)
}

func ParseSilenceMatchersJSON(raw string) ([]SilenceMatcher, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, constants.ErrBadRequestWithMsg("静默 matchers 不能为空")
	}
	var ms []SilenceMatcher
	if err := json.Unmarshal([]byte(raw), &ms); err != nil {
		return nil, bizerrors.Pass(context.Background(), "alert.silence", "ParseSilenceMatchersJSON", err)
	}
	if len(ms) == 0 {
		return nil, constants.ErrBadRequestWithMsg("静默 matchers 不能为空")
	}
	for _, m := range ms {
		if strings.TrimSpace(m.Name) == "" {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg3f8f3c7674a1)
		}
	}
	return ms, nil
}

func normalizeSilenceMatchersKey(ms []SilenceMatcher) string {
	if len(ms) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		name := strings.TrimSpace(m.Name)
		value := strings.TrimSpace(m.Value)
		regex := "0"
		if m.IsRegex {
			regex = "1"
		}
		parts = append(parts, name+"|"+value+"|"+regex)
	}
	sort.Strings(parts)
	return strings.Join(parts, ";")
}

func (s *AlertSilenceService) hasEnabledUnexpiredDuplicate(ctx context.Context, projectID uint, matchersJSON string, now time.Time) (bool, error) {
	targetMatchers, err := ParseSilenceMatchersJSON(matchersJSON)
	if err != nil {
		return false, bizerrors.Pass(ctx, "alert.silence", "hasEnabledUnexpiredDuplicate", err)
	}
	targetKey := normalizeSilenceMatchersKey(targetMatchers)
	if targetKey == "" {
		return false, nil
	}
	list, err := s.repo.ListEnabledUnexpiredByProject(ctx, projectID, now)
	if err != nil {
		return false, bizerrors.Pass(ctx, "alert.silence", "hasEnabledUnexpiredDuplicate", err)
	}
	for _, row := range list {
		ms, err := ParseSilenceMatchersJSON(row.MatchersJSON)
		if err != nil {
			continue
		}
		if normalizeSilenceMatchersKey(ms) == targetKey {
			return true, nil
		}
	}
	return false, nil
}

// LabelsMatchSilenceMatchers 全部 matcher 命中 labels 时返回 true；matchers 为空不匹配任何告警。
func LabelsMatchSilenceMatchers(ms []SilenceMatcher, labels map[string]string) bool {
	if len(ms) == 0 {
		return false
	}
	get := func(k string) string {
		if labels == nil {
			return ""
		}
		return strings.TrimSpace(labels[k])
	}
	for _, m := range ms {
		name := strings.TrimSpace(m.Name)
		want := m.Value
		got := get(name)
		if m.IsRegex {
			re, err := regexp.Compile("^(?:" + want + ")$")
			if err != nil || !re.MatchString(got) {
				return false
			}
		} else {
			if got != strings.TrimSpace(want) {
				return false
			}
		}
	}
	return true
}

func (s *AlertSilenceService) List(ctx context.Context, q AlertSilenceListQuery) ([]model.AlertSilence, int64, int, int, error) {
	s.disableExpiredSilences(ctx, time.Now())
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.ListPaged(ctx, q.ProjectID, q.Keyword, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.silence", "List", err)
	}
	return list, total, page, pageSize, nil
}

// ListActiveAt 返回在 t 时刻生效的静默（用于 Webhook 入口）。
func (s *AlertSilenceService) ListActiveAt(ctx context.Context, t time.Time) ([]model.AlertSilence, error) {
	list, err := s.repo.ListActiveAt(ctx, t)
	return list, bizerrors.Pass(ctx, "alert.silence", "ListActiveAt", err)
}

// silenceProjectScopeOK：ProjectID==0 为全局静默，可匹配任意告警；>0 时要求 labels["project_id"] 等于该项目。
func silenceProjectScopeOK(projectID uint, labels map[string]string) bool {
	if projectID == 0 {
		return true
	}
	if labels == nil {
		return false
	}
	return parseLabelUintOrZero(labels["project_id"]) == projectID
}

func (s *AlertSilenceService) FirstMatchingSilenceID(ctx context.Context, labels map[string]string, t time.Time) (uint, bool, error) {
	list, err := s.ListActiveAt(ctx, t)
	if err != nil {
		return 0, false, bizerrors.Pass(ctx, "alert.silence", "FirstMatchingSilenceID", err)
	}
	for _, sil := range list {
		ms, err := ParseSilenceMatchersJSON(sil.MatchersJSON)
		if err != nil {
			continue
		}
		if LabelsMatchSilenceMatchers(ms, labels) && silenceProjectScopeOK(sil.ProjectID, labels) {
			return sil.ID, true, nil
		}
	}
	return 0, false, nil
}

func (s *AlertSilenceService) Create(ctx context.Context, actor *auth.CurrentUser, req AlertSilenceUpsertRequest) (*model.AlertSilence, error) {
	if err := s.assertSilenceProjectAccess(ctx, actor, req.ProjectID); err != nil {
		return nil, err
	}
	userID := uint(0)
	if actor != nil {
		userID = actor.ID
	}
	ms, err := ParseSilenceMatchersJSON(req.MatchersJSON)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.silence", "Create", err)
	}
	if len(ms) == 0 {
		return nil, constants.ErrBadRequestWithMsg("静默 matchers 不能为空")
	}
	if !req.EndsAt.After(req.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	dup, err := s.hasEnabledUnexpiredDuplicate(ctx, req.ProjectID, req.MatchersJSON, time.Now())
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.silence", "Create", err)
	}
	if dup {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg612f94e277ef)
	}
	row := model.AlertSilence{
		Name:         strings.TrimSpace(req.Name),
		MatchersJSON: strings.TrimSpace(req.MatchersJSON),
		StartsAt:     req.StartsAt,
		EndsAt:       req.EndsAt,
		Comment:      strings.TrimSpace(req.Comment),
		CreatedBy:    userID,
		ProjectID:    req.ProjectID,
		Enabled:      req.Enabled == nil || *req.Enabled,
	}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.silence", "Create", err)
	}
	if row.ProjectID > 0 {
		changeevent.Record(ctx, changeevent.Input{
			ProjectID: row.ProjectID,
			Source:    model.ChangeSourceAlert,
			Action:    "silence_create",
			RiskLevel: model.ChangeRiskLow,
			Status:    model.ChangeStatusSucceeded,
			Summary:   fmt.Sprintf("创建告警静默「%s」至 %s", row.Name, row.EndsAt.Format(time.RFC3339)),
			Payload: map[string]any{
				"silence_id": row.ID, "name": row.Name, "ends_at": row.EndsAt,
			},
		})
	}
	return &row, nil
}

func (s *AlertSilenceService) Update(ctx context.Context, id uint, actor *auth.CurrentUser, req AlertSilenceUpsertRequest) (*model.AlertSilence, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrAlertSilenceNotFound
		}
		return nil, bizerrors.Pass(ctx, "alert.silence", "Update", err)
	}
	projectID := row.ProjectID
	if req.ProjectID > 0 {
		projectID = req.ProjectID
	}
	if err := s.assertSilenceProjectAccess(ctx, actor, projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.MatchersJSON) != "" {
		ms, err := ParseSilenceMatchersJSON(req.MatchersJSON)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.silence", "Update", err)
		}
		if len(ms) == 0 {
			return nil, constants.ErrBadRequestWithMsg("静默 matchers 不能为空")
		}
		row.MatchersJSON = strings.TrimSpace(req.MatchersJSON)
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	if !req.StartsAt.IsZero() {
		row.StartsAt = req.StartsAt
	}
	if !req.EndsAt.IsZero() {
		row.EndsAt = req.EndsAt
	}
	if row.EndsAt.Before(row.StartsAt) || row.EndsAt.Equal(row.StartsAt) {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgc1f741f96c03)
	}
	row.Comment = strings.TrimSpace(req.Comment)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.ProjectID > 0 {
		row.ProjectID = req.ProjectID
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.silence", "Update", err)
	}
	return row, nil
}

func (s *AlertSilenceService) GetByID(ctx context.Context, id uint) (*model.AlertSilence, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrAlertSilenceNotFound
		}
		return nil, bizerrors.Pass(ctx, "alert.silence", "GetByID", err)
	}
	return row, nil
}

func (s *AlertSilenceService) Delete(ctx context.Context, id uint, actor *auth.CurrentUser) error {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrAlertSilenceNotFound
		}
		return bizerrors.Pass(ctx, "alert.silence", "Delete", err)
	}
	if err := s.assertSilenceProjectAccess(ctx, actor, row.ProjectID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return bizerrors.Pass(ctx, "alert.silence", "Delete", err)
	}
	return nil
}

func (s *AlertSilenceService) CreateBatch(ctx context.Context, actor *auth.CurrentUser, req AlertSilenceBatchRequest) (int, error) {
	n := 0
	userID := uint(0)
	if actor != nil {
		userID = actor.ID
	}
	for _, it := range req.Items {
		ms, err := ParseSilenceMatchersJSON(it.MatchersJSON)
		if err != nil {
			return n, bizerrors.Pass(ctx, "alert.silence", "CreateBatch", err)
		}
		if len(ms) == 0 {
			return n, constants.ErrBadRequestWithMsg("静默 matchers 不能为空")
		}
		if !it.EndsAt.After(it.StartsAt) {
			return n, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmtAlertSilenceBatchEndsAt, it.Name))
		}
		projectID := req.ProjectID
		if it.ProjectID > 0 {
			projectID = it.ProjectID
		}
		if err := s.assertSilenceProjectAccess(ctx, actor, projectID); err != nil {
			return n, err
		}
		dup, err := s.hasEnabledUnexpiredDuplicate(ctx, projectID, it.MatchersJSON, time.Now())
		if err != nil {
			return n, bizerrors.Pass(ctx, "alert.silence", "CreateBatch", err)
		}
		if dup {
			return n, constants.ErrBadRequestWithMsg(constants.ErrMsg76b99177ec58)
		}
		row := model.AlertSilence{
			Name:         strings.TrimSpace(it.Name),
			MatchersJSON: strings.TrimSpace(it.MatchersJSON),
			StartsAt:     it.StartsAt,
			EndsAt:       it.EndsAt,
			Comment:      strings.TrimSpace(it.Comment),
			CreatedBy:    userID,
			ProjectID:    projectID,
			Enabled:      it.Enabled == nil || *it.Enabled,
		}
		if err := s.repo.Create(ctx, &row); err != nil {
			return n, bizerrors.Pass(ctx, "alert.silence", "CreateBatch", err)
		}
		n++
	}
	return n, nil
}
