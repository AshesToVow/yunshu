package dbmgmt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)

type ColumnMaskRuleUpsertRequest struct {
	SchemaName string `json:"schema_name"`
	TableName  string `json:"table_name" binding:"required,max=128"`
	ColumnName string `json:"column_name" binding:"required,max=128"`
	MaskType   string `json:"mask_type" binding:"required"`
	Pattern    string `json:"pattern"`
}

func (s *Service) ListColumnMaskRules(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) ([]model.DbColumnMaskRule, error) {
	if _, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID); err != nil {
		return nil, err
	}
	if err := s.requireInstanceManage(ctx, projectID, instanceID, actor); err != nil {
		return nil, err
	}
	list, err := s.repo.ListColumnMaskRules(ctx, instanceID)
	return list, bizerrors.Pass(ctx, "dbmgmt.mask", "List", err)
}

func (s *Service) UpsertColumnMaskRule(ctx context.Context, projectID, instanceID uint, req ColumnMaskRuleUpsertRequest, actor *auth.CurrentUser) (*model.DbColumnMaskRule, error) {
	if _, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID); err != nil {
		return nil, err
	}
	if err := s.requireInstanceManage(ctx, projectID, instanceID, actor); err != nil {
		return nil, err
	}
	mt := strings.TrimSpace(strings.ToLower(req.MaskType))
	if mt == "" {
		mt = "partial"
	}
	schema := strings.TrimSpace(req.SchemaName)
	table := strings.TrimSpace(req.TableName)
	column := strings.TrimSpace(req.ColumnName)
	if table == "" || column == "" {
		return nil, constants.ErrBadRequestWithMsg("table_name 与 column_name 必填")
	}
	row, err := s.repo.FindColumnMaskRule(ctx, instanceID, schema, table, column)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, bizerrors.Pass(ctx, "dbmgmt.mask", "Find", err)
		}
		row = &model.DbColumnMaskRule{
			InstanceID: instanceID,
			SchemaName: schema,
			MaskTable:  table,
			ColumnName: column,
		}
	}
	row.MaskType = mt
	row.Pattern = strings.TrimSpace(req.Pattern)
	if row.ID == 0 {
		err = s.repo.CreateColumnMaskRule(ctx, row)
	} else {
		err = s.repo.UpdateColumnMaskRule(ctx, row)
	}
	if err != nil {
		return nil, bizerrors.Pass(ctx, "dbmgmt.mask", "Upsert", err)
	}
	return row, nil
}

func (s *Service) DeleteColumnMaskRule(ctx context.Context, projectID, instanceID, id uint, actor *auth.CurrentUser) error {
	if _, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID); err != nil {
		return err
	}
	if err := s.requireInstanceManage(ctx, projectID, instanceID, actor); err != nil {
		return err
	}
	err := s.repo.DeleteColumnMaskRule(ctx, instanceID, id)
	if err == gorm.ErrRecordNotFound {
		return constants.ErrNotFound
	}
	return bizerrors.Pass(ctx, "dbmgmt.mask", "Delete", err)
}

func (s *Service) applyColumnMasks(ctx context.Context, instanceID uint, database, sqlText string, cols []string, rows [][]any) {
	if s == nil || s.repo == nil || instanceID == 0 || len(cols) == 0 {
		return
	}
	rules, err := s.repo.ListColumnMaskRules(ctx, instanceID)
	if err != nil || len(rules) == 0 {
		return
	}
	refs := extractQueryTableRefs(sqlText, database)
	tableSet := map[string]struct{}{}
	for _, r := range refs {
		t := strings.ToLower(strings.TrimSpace(r.Table))
		if t != "" {
			tableSet[t] = struct{}{}
		}
	}
	colIndex := map[string]int{}
	for i, c := range cols {
		name := strings.ToLower(strings.TrimSpace(c))
		colIndex[name] = i
		// 兼容 table.column / alias.column
		if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
			colIndex[name[idx+1:]] = i
		}
	}
	for _, rule := range rules {
		if db := strings.TrimSpace(database); db != "" && rule.SchemaName != "" && !strings.EqualFold(rule.SchemaName, db) {
			continue
		}
		maskTable := strings.ToLower(strings.TrimSpace(rule.MaskTable))
		if maskTable != "" && len(tableSet) > 0 {
			if _, ok := tableSet[maskTable]; !ok {
				continue
			}
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
