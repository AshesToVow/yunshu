package dbmgmt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

type ColumnMaskRuleUpsertRequest struct {
	SchemaName string `json:"schema_name"`
	TableName  string `json:"table_name" binding:"required,max=128"`
	ColumnName string `json:"column_name" binding:"required,max=128"`
	MaskType   string `json:"mask_type" binding:"required"`
	Pattern    string `json:"pattern"`
}

func (s *Service) ListColumnMaskRules(ctx context.Context, instanceID uint) ([]model.DbColumnMaskRule, error) {
	var list []model.DbColumnMaskRule
	err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).Order("id ASC").Find(&list).Error
	return list, bizerrors.Pass(ctx, "dbmgmt.mask", "List", err)
}

func (s *Service) UpsertColumnMaskRule(ctx context.Context, instanceID uint, req ColumnMaskRuleUpsertRequest) (*model.DbColumnMaskRule, error) {
	mt := strings.TrimSpace(strings.ToLower(req.MaskType))
	if mt == "" {
		mt = "partial"
	}
	var row model.DbColumnMaskRule
	err := s.db.WithContext(ctx).Where(
		"instance_id = ? AND schema_name = ? AND table_name = ? AND column_name = ?",
		instanceID, strings.TrimSpace(req.SchemaName), strings.TrimSpace(req.TableName), strings.TrimSpace(req.ColumnName),
	).First(&row).Error
	if err != nil {
		row = model.DbColumnMaskRule{
			InstanceID: instanceID,
			SchemaName: strings.TrimSpace(req.SchemaName),
			MaskTable:  strings.TrimSpace(req.TableName),
			ColumnName: strings.TrimSpace(req.ColumnName),
		}
	}
	row.MaskType = mt
	row.Pattern = strings.TrimSpace(req.Pattern)
	if row.ID == 0 {
		err = s.db.WithContext(ctx).Create(&row).Error
	} else {
		err = s.db.WithContext(ctx).Save(&row).Error
	}
	if err != nil {
		return nil, bizerrors.Pass(ctx, "dbmgmt.mask", "Upsert", err)
	}
	return &row, nil
}

func (s *Service) DeleteColumnMaskRule(ctx context.Context, instanceID, id uint) error {
	res := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).Delete(&model.DbColumnMaskRule{}, id)
	if res.Error != nil {
		return bizerrors.Pass(ctx, "dbmgmt.mask", "Delete", res.Error)
	}
	if res.RowsAffected == 0 {
		return constants.ErrNotFound
	}
	return nil
}

func (s *Service) applyColumnMasks(ctx context.Context, instanceID uint, database string, cols []string, rows [][]any) {
	if s == nil || s.db == nil || instanceID == 0 || len(cols) == 0 {
		return
	}
	var rules []model.DbColumnMaskRule
	_ = s.db.WithContext(ctx).Where("instance_id = ?", instanceID).Find(&rules).Error
	if len(rules) == 0 {
		return
	}
	colIndex := map[string]int{}
	for i, c := range cols {
		colIndex[strings.ToLower(strings.TrimSpace(c))] = i
	}
	for _, rule := range rules {
		if db := strings.TrimSpace(database); db != "" && rule.SchemaName != "" && !strings.EqualFold(rule.SchemaName, db) {
			continue
		}
		idx, ok := colIndex[strings.ToLower(rule.ColumnName)]
		if !ok {
			continue
		}
		for ri := range rows {
			if idx >= len(rows[ri]) {
				continue
			}
			rows[ri][idx] = maskCellValue(rows[ri][idx], rule)
		}
	}
}

func maskCellValue(v any, rule model.DbColumnMaskRule) any {
	if v == nil {
		return v
	}
	s := fmt.Sprint(v)
	switch strings.ToLower(strings.TrimSpace(rule.MaskType)) {
	case "hash":
		sum := sha256.Sum256([]byte(s))
		return "sha256:" + hex.EncodeToString(sum[:8])
	case "redact":
		return "***"
	default: // partial
		parts := strings.Split(strings.TrimSpace(rule.Pattern), ",")
		keepHead, keepTail := 3, 4
		if len(parts) >= 1 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && n >= 0 {
				keepHead = n
			}
		}
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && n >= 0 {
				keepTail = n
			}
		}
		runes := []rune(s)
		if len(runes) <= keepHead+keepTail {
			return strings.Repeat("*", len(runes))
		}
		return string(runes[:keepHead]) + strings.Repeat("*", len(runes)-keepHead-keepTail) + string(runes[len(runes)-keepTail:])
	}
}
