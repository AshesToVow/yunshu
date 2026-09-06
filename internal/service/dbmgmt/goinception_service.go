package dbmgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/goinception"
)

func (s *Service) goInceptionAvailable(ctx context.Context, inst *model.DbInstance) bool {
	cfg := s.resolvedConfig(ctx)
	if !cfg.GoInceptionEnabled || strings.TrimSpace(cfg.GoInceptionHost) == "" {
		return false
	}
	if strings.ToLower(strings.TrimSpace(inst.Driver)) != model.DbDriverMySQL {
		return false
	}
	if strings.ToLower(strings.TrimSpace(inst.ConnectMode)) == model.DbConnectSSHTunnel {
		return false
	}
	return true
}

func (s *Service) goInceptionClient(ctx context.Context) *goinception.Client {
	cfg := s.resolvedConfig(ctx)
	return goinception.NewClient(cfg.GoInceptionHost, cfg.GoInceptionPort)
}

func (s *Service) instanceGoInceptionTarget(inst *model.DbInstance) (goinception.Target, error) {
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return goinception.Target{}, constants.ErrBadRequestWithMsg(constants.ErrMsgDbInstancePasswordDecryptFailed)
	}
	return goinception.Target{
		Host:     inst.Host,
		Port:     inst.Port,
		Username: inst.Username,
		Password: pw,
	}, nil
}

type SQLCheckRequest struct {
	Database  string `json:"database"`
	Sql       string `json:"sql" binding:"required"`
	// AuditMode: system=走 goInception；manual=仅平台本地规则，不连 goInception。
	AuditMode string `json:"audit_mode"`
}

type SQLCheckResponse struct {
	Checked      bool                  `json:"checked"`
	Goinception  bool                  `json:"goinception"`
	SyntaxType   int                   `json:"syntax_type"`
	ErrorCount   int                   `json:"error_count"`
	WarningCount int                   `json:"warning_count"`
	RiskLevel    string                `json:"risk_level"`
	Rows         []goinception.ReviewRow `json:"rows,omitempty"`
	Error        string                `json:"error,omitempty"`
}

func (s *Service) CheckSQL(ctx context.Context, projectID, instanceID uint, req SQLCheckRequest, actor *auth.CurrentUser) (*SQLCheckResponse, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	sqlText := strings.TrimSpace(req.Sql)
	if sqlText == "" {
		return nil, constants.ErrBadRequestWithMsg("SQL 不能为空")
	}
	instanceDDL := isInstanceLevelDDL(sqlText)
	if !instanceDDL {
		if _, err := resolveQueryDatabase(inst, req.Database); err != nil {
			return nil, err
		}
	}
	needDDL := reDDL.MatchString(strings.ToUpper(sqlText))
	if err := s.checkWritePermission(ctx, projectID, inst, req.Database, sqlText, needDDL, actor); err != nil {
		return nil, err
	}
	// 仅「系统审核」走 goInception；人工审核只做平台本地规则预检。
	useGoInception := normalizeAuditMode(req.AuditMode) == model.DbAuditModeSystem &&
		s.goInceptionAvailable(ctx, inst)
	assess := AssessSQLForWrite(sqlText, inst.Env == model.DbEnvProd, useGoInception)
	out := &SQLCheckResponse{
		RiskLevel:  assess.RiskLevel,
		SyntaxType: goinception.SyntaxDML,
	}
	if assess.Blocked {
		out.Checked = true
		out.Error = assess.Reason
		out.ErrorCount = 1
		return out, nil
	}
	if !useGoInception {
		out.Checked = true
		if assess.RiskLevel == model.DbRiskHigh || reDDL.MatchString(strings.ToUpper(sqlText)) {
			out.SyntaxType = goinception.SyntaxDDL
		}
		return out, nil
	}
	dbName := strings.TrimSpace(req.Database)
	if dbName == "" && !isInstanceLevelDDL(sqlText) {
		return nil, constants.ErrBadRequestWithMsg("请先选择数据库")
	}
	target, err := s.instanceGoInceptionTarget(inst)
	if err != nil {
		return nil, err
	}
	rs, err := s.goInceptionClient(ctx).Check(ctx, target, dbName, sqlText)
	out.Goinception = true
	out.Checked = true
	if rs != nil {
		out.Rows = rs.Rows
		out.ErrorCount = rs.ErrorCount
		out.WarningCount = rs.WarningCount
		out.SyntaxType = rs.SyntaxType
		if rs.Error != "" {
			out.Error = rs.Error
		} else if rs.ErrorCount > 0 {
			out.Error = firstReviewErrorMessage(rs, "")
		}
	}
	if err != nil {
		if out.Error == "" {
			out.Error = err.Error()
		}
		out.ErrorCount++
	}
	return out, nil
}

func (s *Service) runGoInceptionCheck(ctx context.Context, inst *model.DbInstance, dbName, sqlText string) (*goinception.ReviewSet, error) {
	target, err := s.instanceGoInceptionTarget(inst)
	if err != nil {
		return nil, err
	}
	return s.goInceptionClient(ctx).Check(ctx, target, dbName, sqlText)
}

func (s *Service) runGoInceptionExecute(ctx context.Context, inst *model.DbInstance, dbName, sqlText string, backup bool) (*goinception.ReviewSet, error) {
	target, err := s.instanceGoInceptionTarget(inst)
	if err != nil {
		return nil, err
	}
	return s.goInceptionClient(ctx).Execute(ctx, target, dbName, sqlText, backup)
}

func firstReviewErrorMessage(rs *goinception.ReviewSet, fallback string) string {
	if rs == nil {
		return fallback
	}
	if msg := strings.TrimSpace(rs.Error); msg != "" {
		return msg
	}
	for _, row := range rs.Rows {
		if row.ErrorLevel >= goinception.ErrLevelError && strings.TrimSpace(row.ErrorMessage) != "" {
			if row.OrderID > 0 {
				return fmt.Sprintf("第 %d 行: %s", row.OrderID, row.ErrorMessage)
			}
			return row.ErrorMessage
		}
	}
	if fallback != "" {
		return fallback
	}
	return "SQL 预检未通过"
}

func marshalReviewSet(rs *goinception.ReviewSet) string {
	if rs == nil {
		return ""
	}
	safe := sanitizeReviewSet(rs)
	b, _ := json.Marshal(safe)
	return string(b)
}

func sanitizeReviewSet(rs *goinception.ReviewSet) *goinception.ReviewSet {
	if rs == nil {
		return nil
	}
	cp := *rs
	cp.FullSQL = ""
	return &cp
}

func sanitizeReviewJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	rs := reviewSetFromJSON(raw)
	if rs == nil {
		return raw
	}
	b, _ := json.Marshal(sanitizeReviewSet(rs))
	return string(b)
}

func reviewSetFromJSON(raw string) *goinception.ReviewSet {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rs goinception.ReviewSet
	if err := json.Unmarshal([]byte(raw), &rs); err != nil {
		return nil
	}
	return &rs
}
