package alert

import (
	"context"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type AlertReceiverGroupService struct {
	repo  interfaces.AlertReceiverGroupRepository
	cache *ReceiverGroupCache
}

func NewAlertReceiverGroupService(repo interfaces.AlertReceiverGroupRepository, cache *ReceiverGroupCache) *AlertReceiverGroupService {
	return &AlertReceiverGroupService{repo: repo, cache: cache}
}

type AlertReceiverGroupListQuery struct {
	ProjectID uint   `form:"project_id"`
	Keyword   string `form:"keyword"`
	Enabled   *bool  `form:"enabled"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (s *AlertReceiverGroupService) List(ctx context.Context, q AlertReceiverGroupListQuery) ([]model.AlertReceiverGroup, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.repo.List(ctx, repository.AlertReceiverGroupListFilter{
		ProjectID: q.ProjectID,
		Keyword:   q.Keyword,
		Enabled:   q.Enabled,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.receiver", "List", err)
	}
	for i := range list {
		hydrateReceiverGroup(&list[i])
	}
	return list, total, page, pageSize, nil
}

type AlertReceiverGroupUpsertRequest struct {
	ProjectID   uint   `json:"project_id"` // 0=全局接收组
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description"`

	ChannelIDsJSON      string  `json:"channel_ids_json"`
	EmailRecipientsJSON string  `json:"email_recipients_json"`
	ActiveTimeStart     *string `json:"active_time_start"`
	ActiveTimeEnd       *string `json:"active_time_end"`
	WeekdaysJSON           string `json:"weekdays_json"`
	EscalationLevel        int    `json:"escalation_level"`
	EscalationDelaySeconds int    `json:"escalation_delay_seconds"`
	Enabled                *bool  `json:"enabled"`
}

func (s *AlertReceiverGroupService) Create(ctx context.Context, req AlertReceiverGroupUpsertRequest) (*model.AlertReceiverGroup, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg65788b1abed6)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &model.AlertReceiverGroup{
		ProjectID:              req.ProjectID,
		Name:                   strings.TrimSpace(req.Name),
		Description:            strings.TrimSpace(req.Description),
		ChannelIDsJSON:         strings.TrimSpace(req.ChannelIDsJSON),
		EmailRecipientsJSON:    strings.TrimSpace(req.EmailRecipientsJSON),
		ActiveTimeStart:        req.ActiveTimeStart,
		ActiveTimeEnd:          req.ActiveTimeEnd,
		WeekdaysJSON:           strings.TrimSpace(req.WeekdaysJSON),
		EscalationLevel:        normalizeEscalationLevel(req.EscalationLevel),
		EscalationDelaySeconds: normalizeEscalationDelaySeconds(req.EscalationDelaySeconds),
		Enabled:                enabled,
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.receiver", "Create", err)
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	hydrateReceiverGroup(row)
	return row, nil
}

func (s *AlertReceiverGroupService) Update(ctx context.Context, id uint, req AlertReceiverGroupUpsertRequest) (*model.AlertReceiverGroup, error) {
	row, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsg7628a50dd0ab)
		}
		return nil, bizerrors.Pass(ctx, "alert.receiver", "Update", err)
	}
	if req.ProjectID > 0 && req.ProjectID != row.ProjectID {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgd165b6af9d52)
	}
	if strings.TrimSpace(req.Name) != "" {
		row.Name = strings.TrimSpace(req.Name)
	}
	row.Description = strings.TrimSpace(req.Description)
	row.ChannelIDsJSON = strings.TrimSpace(req.ChannelIDsJSON)
	row.EmailRecipientsJSON = strings.TrimSpace(req.EmailRecipientsJSON)
	row.ActiveTimeStart = req.ActiveTimeStart
	row.ActiveTimeEnd = req.ActiveTimeEnd
	row.WeekdaysJSON = strings.TrimSpace(req.WeekdaysJSON)
	row.EscalationLevel = normalizeEscalationLevel(req.EscalationLevel)
	row.EscalationDelaySeconds = normalizeEscalationDelaySeconds(req.EscalationDelaySeconds)
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if err := s.repo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.receiver", "Update", err)
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	hydrateReceiverGroup(row)
	return row, nil
}

func (s *AlertReceiverGroupService) Delete(ctx context.Context, id uint) error {
	n, err := s.repo.Delete(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.receiver", "Delete", err)
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg(constants.ErrMsg7628a50dd0ab)
	}
	if s.cache != nil {
		s.cache.Invalidate()
	}
	return nil
}

func hydrateReceiverGroup(it *model.AlertReceiverGroup) {
	if it == nil {
		return
	}
	it.ChannelIDs = parseUintSliceJSON(it.ChannelIDsJSON)
	it.EmailRecipients = parseStringSliceJSON(it.EmailRecipientsJSON)
	it.Weekdays = parseIntSliceJSON(it.WeekdaysJSON)
}

const (
	maxEscalationLevel         = 10
	maxEscalationDelaySeconds  = 7 * 24 * 3600
	defaultEscalationDelaySecs = 900
)

func normalizeEscalationLevel(v int) int {
	if v < 0 {
		return 0
	}
	if v > maxEscalationLevel {
		return maxEscalationLevel
	}
	return v
}

func normalizeEscalationDelaySeconds(v int) int {
	if v < 0 {
		return 0
	}
	if v > maxEscalationDelaySeconds {
		return maxEscalationDelaySeconds
	}
	return v
}

// effectiveEscalationDelaySeconds level>=1 组的等待秒数；未配置时默认 15 分钟。
func effectiveEscalationDelaySeconds(level, configured int) int {
	if level <= 0 {
		return 0
	}
	if configured > 0 {
		return normalizeEscalationDelaySeconds(configured)
	}
	return defaultEscalationDelaySecs
}
