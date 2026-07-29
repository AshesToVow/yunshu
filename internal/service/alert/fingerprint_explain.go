package alert

import (
	"context"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

// FingerprintDeliveryExplain 按 fingerprint 汇总投递/跳过原因，便于排障「为何没通知」。
type FingerprintDeliveryExplain struct {
	Fingerprint           string                         `json:"fingerprint"`
	FiringDelivered       bool                           `json:"firing_delivered"`
	FiringDeliveredSource string                         `json:"firing_delivered_source,omitempty"` // redis|db|none
	Events                []FingerprintDeliveryEventItem `json:"events"`
	SkipSummary           []FingerprintSkipBucket        `json:"skip_summary"`
}

type FingerprintDeliveryEventItem struct {
	ID              uint   `json:"id"`
	CreatedAt       string `json:"created_at"`
	Status          string `json:"status"`
	Title           string `json:"title"`
	ChannelName     string `json:"channel_name"`
	Success         bool   `json:"success"`
	ErrorMessage    string `json:"error_message"`
	Category        string `json:"category"`
	ReasonHint      string `json:"reason_hint"`
	ResponseSnippet string `json:"response_snippet,omitempty"`
}

type FingerprintSkipBucket struct {
	ErrorMessage string `json:"error_message"`
	Category     string `json:"category"`
	Count        int    `json:"count"`
	Hint         string `json:"hint"`
}

// ExplainFingerprintDelivery 查询某 fingerprint 的投递留痕与 firing_delivered 状态。
func (s *AlertService) ExplainFingerprintDelivery(ctx context.Context, fingerprint string) (*FingerprintDeliveryExplain, error) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil, constants.ErrBadRequestWithMsg("fingerprint required")
	}
	out := &FingerprintDeliveryExplain{Fingerprint: fp}

	if s.redis != nil {
		v, err := s.redis.Get(ctx, firingDeliveredRedisKey(fp)).Result()
		if err == nil && strings.TrimSpace(v) == "1" {
			out.FiringDelivered = true
			out.FiringDeliveredSource = "redis"
		}
	}
	if !out.FiringDelivered && s.firingDeliveryRepo != nil {
		ok, err := s.firingDeliveryRepo.Exists(ctx, fp)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert", "ExplainFingerprintDelivery", err)
		}
		if ok {
			out.FiringDelivered = true
			out.FiringDeliveredSource = "db"
		}
	}
	if out.FiringDeliveredSource == "" {
		out.FiringDeliveredSource = "none"
	}

	list, _, err := s.eventRepo.List(ctx, repository.AlertEventListFilter{Fingerprint: fp}, 0, 200)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert", "ExplainFingerprintDelivery", err)
	}
	buckets := map[string]*FingerprintSkipBucket{}
	out.Events = make([]FingerprintDeliveryEventItem, 0, len(list))
	cfg := s.cfg
	for i := range list {
		ev := list[i]
		hydrateAlertEvent(&ev)
		code := strings.TrimSpace(ev.ErrorMessage)
		cat := classifyAlertEventCategory(ev.Success, code, ev.ChannelID)
		item := FingerprintDeliveryEventItem{
			ID:              ev.ID,
			CreatedAt:       ev.CreatedAt.Format("2006-01-02 15:04:05"),
			Status:          ev.Status,
			Title:           ev.Title,
			ChannelName:     ev.ChannelName,
			Success:         ev.Success,
			ErrorMessage:    code,
			Category:        cat,
			ReasonHint:      humanReasonForErrorMessage(code, cfg),
			ResponseSnippet: truncateText(ev.ResponsePayload, 240),
		}
		out.Events = append(out.Events, item)
		if code == "" || cat == AlertEventCategoryDelivery {
			continue
		}
		b := buckets[code]
		if b == nil {
			b = &FingerprintSkipBucket{
				ErrorMessage: code,
				Category:     cat,
				Hint:         item.ReasonHint,
			}
			buckets[code] = b
		}
		b.Count++
	}
	for _, b := range buckets {
		out.SkipSummary = append(out.SkipSummary, *b)
	}
	return out, nil
}

func classifyAlertEventCategory(success bool, errorMessage string, channelID uint) string {
	code := strings.TrimSpace(errorMessage)
	if strings.HasPrefix(code, "inhibition_suppressed:") {
		return AlertEventCategoryInhibition
	}
	switch code {
	case "silence_suppressed", "subscription_suppressed":
		return AlertEventCategorySilence
	case "group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled":
		return AlertEventCategoryTiming
	case "resolved_aggregate_suppressed", "resolved_no_prior_firing_delivery":
		return AlertEventCategoryResolved
	case "no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription":
		return AlertEventCategoryRouting
	case "all_channel_delivery_failed":
		return AlertEventCategoryFailure
	}
	if !success {
		return AlertEventCategoryFailure
	}
	if channelID > 0 && code == "" {
		return AlertEventCategoryDelivery
	}
	if code != "" {
		return AlertEventCategoryOther
	}
	return AlertEventCategoryDelivery
}

func humanReasonForErrorMessage(code string, cfg config.AlertConfig) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "通道外发成功或无错误码"
	}
	if strings.HasPrefix(code, "inhibition_suppressed:") {
		return "命中抑制规则，本轮未外发"
	}
	switch code {
	case "silence_suppressed":
		return "命中平台静默"
	case "subscription_suppressed":
		return "命中订阅树静默窗口"
	case "group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled":
		return humanReadableGroupTimingSuppression(code, cfg)
	case "resolved_aggregate_suppressed":
		return "同组恢复通知已合并，本轮未重复推送"
	case "resolved_no_prior_firing_delivery":
		return "此前无成功 firing 投递，已抑制恢复外发（以 DB/Redis firing_delivered 为准）"
	case "no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription":
		return "未匹配到可投递通道/订阅"
	case "all_channel_delivery_failed":
		return "所有通道投递均失败"
	default:
		return code
	}
}
