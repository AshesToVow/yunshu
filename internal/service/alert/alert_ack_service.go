package alert

import (
	"context"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

const (
	dictTypeAlertAckTTLMinutes = "alert_ack_ttl_minutes"
	fallbackAckTTLMinutes      = 15
	maxAckTTLMinutes           = 24 * 60
)

type AlertAckRequest struct {
	Fingerprint string `json:"fingerprint" binding:"required,max=512"`
	TTLMinutes  int    `json:"ttl_minutes"`
}

type AlertAckActiveInfo struct {
	Fingerprint string     `json:"fingerprint"`
	Acked       bool       `json:"acked"`
	UserID      uint       `json:"user_id,omitempty"`
	UserName    string     `json:"user_name,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// AcknowledgeAlert 认领告警：TTL 内同指纹 firing 通知抑制。
func (s *AlertService) AcknowledgeAlert(ctx context.Context, userID uint, userName string, req AlertAckRequest) (*model.AlertAck, error) {
	if s == nil || s.db == nil {
		return nil, constants.ErrInternal
	}
	fp := strings.TrimSpace(req.Fingerprint)
	if fp == "" {
		return nil, constants.ErrBadRequestWithMsg("fingerprint required")
	}
	ttl, err := parseAckTTLMinutes(s.loadAckTTLAllowed(ctx), req.TTLMinutes)
	if err != nil {
		return nil, err
	}
	row := &model.AlertAck{
		Fingerprint: fp,
		UserID:      userID,
		UserName:    strings.TrimSpace(userName),
		ExpiresAt:   time.Now().UTC().Add(time.Duration(ttl) * time.Minute),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.ack", "AcknowledgeAlert", err)
	}
	s.clearEscalationState(ctx, fp)
	return row, nil
}

func (s *AlertService) loadAckTTLAllowed(ctx context.Context) []int {
	if s == nil || s.db == nil {
		return nil
	}
	var rows []model.DictEntry
	err := s.db.WithContext(ctx).
		Where("dict_type = ? AND status = ?", dictTypeAlertAckTTLMinutes, 1).
		Order("sort ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil
	}
	return parseAckTTLDictValues(rows)
}

func parseAckTTLDictValues(rows []model.DictEntry) []int {
	seen := map[int]struct{}{}
	out := []int{}
	for _, row := range rows {
		n, err := strconv.Atoi(strings.TrimSpace(row.Value))
		if err != nil || n <= 0 {
			continue
		}
		if n > maxAckTTLMinutes {
			n = maxAckTTLMinutes
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func parseAckTTLMinutes(allowed []int, requested int) (int, error) {
	if requested > maxAckTTLMinutes {
		requested = maxAckTTLMinutes
	}
	if len(allowed) == 0 {
		if requested <= 0 {
			return fallbackAckTTLMinutes, nil
		}
		return requested, nil
	}
	if requested <= 0 {
		return allowed[0], nil
	}
	for _, m := range allowed {
		if m == requested {
			return m, nil
		}
	}
	return 0, constants.ErrBadRequestWithMsg("认领时长须为数据字典 alert_ack_ttl_minutes 中的启用项")
}

// ClearAlertAck 提前结束认领（将未过期记录的 expires_at 置为现在）。
func (s *AlertService) ClearAlertAck(ctx context.Context, fingerprint string) error {
	if s == nil || s.db == nil {
		return nil
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return constants.ErrBadRequestWithMsg("fingerprint required")
	}
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Model(&model.AlertAck{}).
		Where("fingerprint = ? AND expires_at > ?", fp, now).
		Update("expires_at", now).Error
	return bizerrors.Pass(ctx, "alert.ack", "ClearAlertAck", err)
}

// IsAckActive 当前指纹是否仍在认领有效期内。
func (s *AlertService) IsAckActive(ctx context.Context, fingerprint string) bool {
	info, _ := s.GetActiveAck(ctx, fingerprint)
	return info != nil && info.Acked
}

func (s *AlertService) GetActiveAck(ctx context.Context, fingerprint string) (*AlertAckActiveInfo, error) {
	fp := strings.TrimSpace(fingerprint)
	out := &AlertAckActiveInfo{Fingerprint: fp}
	if s == nil || s.db == nil || fp == "" {
		return out, nil
	}
	var row model.AlertAck
	err := s.db.WithContext(ctx).
		Where("fingerprint = ? AND expires_at > ?", fp, time.Now().UTC()).
		Order("expires_at desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return out, nil
	}
	if err != nil {
		return out, bizerrors.Pass(ctx, "alert.ack", "GetActiveAck", err)
	}
	exp := row.ExpiresAt
	out.Acked = true
	out.UserID = row.UserID
	out.UserName = row.UserName
	out.ExpiresAt = &exp
	return out, nil
}

// ListActiveAcksByFingerprints 批量查询页内指纹的认领状态。
func (s *AlertService) ListActiveAcksByFingerprints(ctx context.Context, fingerprints []string) (map[string]AlertAckActiveInfo, error) {
	out := make(map[string]AlertAckActiveInfo)
	if s == nil || s.db == nil || len(fingerprints) == 0 {
		return out, nil
	}
	uniq := uniqueNonEmptyStrings(fingerprints)
	if len(uniq) == 0 {
		return out, nil
	}
	var rows []model.AlertAck
	err := s.db.WithContext(ctx).
		Where("fingerprint IN ? AND expires_at > ?", uniq, time.Now().UTC()).
		Order("expires_at desc").
		Find(&rows).Error
	if err != nil {
		return out, bizerrors.Pass(ctx, "alert.ack", "ListActiveAcksByFingerprints", err)
	}
	for _, row := range rows {
		if _, ok := out[row.Fingerprint]; ok {
			continue
		}
		exp := row.ExpiresAt
		out[row.Fingerprint] = AlertAckActiveInfo{
			Fingerprint: row.Fingerprint,
			Acked:       true,
			UserID:      row.UserID,
			UserName:    row.UserName,
			ExpiresAt:   &exp,
		}
	}
	return out, nil
}

func uniqueNonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
