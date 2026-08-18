package alert

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

const defaultAckTTLMinutes = 15

type AlertAckRequest struct {
	Fingerprint string `json:"fingerprint" binding:"required,max=512"`
	TTLMinutes  int    `json:"ttl_minutes"`
}

type AlertAckActiveInfo struct {
	Fingerprint  string     `json:"fingerprint"`
	Acked        bool       `json:"acked"`
	UserID       uint       `json:"user_id,omitempty"`
	UserName     string     `json:"user_name,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
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
	ttl := req.TTLMinutes
	if ttl <= 0 {
		ttl = defaultAckTTLMinutes
	}
	if ttl > 24*60 {
		ttl = 24 * 60
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
	return row, nil
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
	uniq := make([]string, 0, len(fingerprints))
	seen := map[string]struct{}{}
	for _, fp := range fingerprints {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		if _, ok := seen[fp]; ok {
			continue
		}
		seen[fp] = struct{}{}
		uniq = append(uniq, fp)
	}
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
