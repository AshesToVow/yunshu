package alert

import (
	"context"
	"strings"
	"unicode/utf8"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

const maxAlertNoteRunes = 2000

type AlertNoteCreateRequest struct {
	Fingerprint string `json:"fingerprint" binding:"required,max=512"`
	Content     string `json:"content" binding:"required"`
}

func normalizeAlertNoteContent(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", constants.ErrBadRequestWithMsg("进展内容不能为空")
	}
	if utf8.RuneCountInString(s) > maxAlertNoteRunes {
		return "", constants.ErrBadRequestWithMsg("进展内容最多 2000 字")
	}
	return s, nil
}

func (s *AlertService) CreateAlertNote(ctx context.Context, userID uint, userName string, req AlertNoteCreateRequest) (*model.AlertProgressNote, error) {
	if s == nil || s.db == nil {
		return nil, constants.ErrInternal
	}
	fp := strings.TrimSpace(req.Fingerprint)
	if fp == "" {
		return nil, constants.ErrBadRequestWithMsg("fingerprint required")
	}
	content, err := normalizeAlertNoteContent(req.Content)
	if err != nil {
		return nil, err
	}
	row := &model.AlertProgressNote{
		Fingerprint: fp,
		UserID:      userID,
		UserName:    strings.TrimSpace(userName),
		Content:     content,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, bizerrors.Pass(ctx, "alert.note", "CreateAlertNote", err)
	}
	return row, nil
}

func (s *AlertService) ListAlertNotes(ctx context.Context, fingerprint string) ([]model.AlertProgressNote, error) {
	out := []model.AlertProgressNote{}
	if s == nil || s.db == nil {
		return out, nil
	}
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return out, constants.ErrBadRequestWithMsg("fingerprint required")
	}
	err := s.db.WithContext(ctx).
		Where("fingerprint = ?", fp).
		Order("id ASC").
		Limit(200).
		Find(&out).Error
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.note", "ListAlertNotes", err)
	}
	return out, nil
}

func (s *AlertService) ListLatestNotesByFingerprints(ctx context.Context, fingerprints []string) (map[string]model.AlertProgressNote, error) {
	out := map[string]model.AlertProgressNote{}
	if s == nil || s.db == nil {
		return out, nil
	}
	uniq := uniqueNonEmptyStrings(fingerprints)
	if len(uniq) == 0 {
		return out, nil
	}
	var rows []model.AlertProgressNote
	err := s.db.WithContext(ctx).
		Where("fingerprint IN ?", uniq).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return out, bizerrors.Pass(ctx, "alert.note", "ListLatestNotesByFingerprints", err)
	}
	for _, row := range rows {
		if _, ok := out[row.Fingerprint]; ok {
			continue
		}
		out[row.Fingerprint] = row
	}
	return out, nil
}
