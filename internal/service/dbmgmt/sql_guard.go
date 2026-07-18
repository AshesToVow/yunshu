package dbmgmt

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// AssessSQL 评估 SQL 风险等级。
type SQLAssessment struct {
	RiskLevel string   `json:"risk_level"`
	Ops       []string `json:"ops"`
	Blocked   bool     `json:"blocked"`
	Reason    string   `json:"reason,omitempty"`
}

var (
	reMultiStmt = regexp.MustCompile(`;\s*\S`)
	reBlocked   = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bDROP\s+DATABASE\b`),
		regexp.MustCompile(`(?i)\bGRANT\b`),
		regexp.MustCompile(`(?i)\bREVOKE\b`),
		regexp.MustCompile(`(?i)\\!`),
		regexp.MustCompile(`(?i)\bSOURCE\b`),
	}
	reHigh = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bDROP\s+TABLE\b`),
		regexp.MustCompile(`(?i)\bALTER\s+TABLE\b`),
		regexp.MustCompile(`(?i)\bCREATE\s+INDEX\b`),
		regexp.MustCompile(`(?i)\bDROP\s+INDEX\b`),
		regexp.MustCompile(`(?i)\bTRUNCATE\b`),
	}
	reDML = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE)\b`)
	reDDL = regexp.MustCompile(`(?i)\b(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b`)
	// 无 WHERE 的 DELETE/UPDATE 会影响全表，风险高于带条件的定向写。
	reDeleteStmt = regexp.MustCompile(`(?i)^\s*DELETE\s+FROM\b`)
	reUpdateStmt = regexp.MustCompile(`(?i)^\s*UPDATE\b`)
	reHasWhere   = regexp.MustCompile(`(?i)\bWHERE\b`)
	reRead  = regexp.MustCompile(`(?i)^\s*(SELECT|SHOW|DESCRIBE|DESC|EXPLAIN|WITH)\b`)
	// goInception 备份库命名：host(点改下划线)_port_原库名，如 10_10_10_103_3306_test
	reGoInceptionBackupDB = regexp.MustCompile(`^\d{1,3}(?:_\d{1,3}){3}_\d+_.+`)
)

// shouldHideDatabase 过滤系统库与 goInception 自动创建的备份库，避免在控制台树中展示。
func shouldHideDatabase(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "information_schema", "performance_schema", "sys", "mysql":
		return true
	}
	n := strings.TrimSpace(name)
	if strings.HasPrefix(n, "_$inception") || strings.HasPrefix(n, "_$Inception") {
		return true
	}
	return reGoInceptionBackupDB.MatchString(n)
}

func hasMultipleStatements(sqlText string) bool {
	return reMultiStmt.MatchString(strings.TrimSpace(sqlText))
}

// AssessSQL 评估单条或本地规则下的 SQL 风险（查询场景等）。
func AssessSQL(sqlText string, prodEnv bool) SQLAssessment {
	return assessSQLSingle(strings.TrimSpace(sqlText), prodEnv)
}

// AssessSQLForWrite 写操作风险评估；启用 goInception 时允许多语句批量提交。
func AssessSQLForWrite(sqlText string, prodEnv, viaGoInception bool) SQLAssessment {
	text := strings.TrimSpace(sqlText)
	if text == "" {
		return SQLAssessment{RiskLevel: model.DbRiskLow, Ops: []string{"empty"}}
	}
	if hasMultipleStatements(text) {
		if !viaGoInception {
			return SQLAssessment{
				RiskLevel: model.DbRiskBlocked, Ops: []string{"multi_statement"}, Blocked: true,
				Reason: "禁止多语句执行（未启用 goInception，请逐条提交或使用「SQL 文件」导入）",
			}
		}
		return assessMultiSQLBatch(text, prodEnv)
	}
	return assessSQLSingle(text, prodEnv)
}

func assessMultiSQLBatch(text string, prodEnv bool) SQLAssessment {
	stmts := splitSQLStatements(text)
	if len(stmts) == 0 {
		return SQLAssessment{RiskLevel: model.DbRiskLow, Ops: []string{"empty"}}
	}
	merged := SQLAssessment{RiskLevel: model.DbRiskLow, Ops: []string{"multi_statement"}}
	for _, st := range stmts {
		a := assessSQLSingle(strings.TrimSpace(st), prodEnv)
		if a.Blocked {
			return a
		}
		merged = mergeSQLAssessment(merged, a)
	}
	return merged
}

func mergeSQLAssessment(a, b SQLAssessment) SQLAssessment {
	out := a
	if sqlRiskRank(b.RiskLevel) > sqlRiskRank(a.RiskLevel) {
		out.RiskLevel = b.RiskLevel
	}
	out.Ops = append(out.Ops, b.Ops...)
	if out.Reason == "" {
		out.Reason = b.Reason
	}
	return out
}

func sqlRiskRank(level string) int {
	switch level {
	case model.DbRiskBlocked:
		return 5
	case model.DbRiskHigh:
		return 4
	case model.DbRiskMedium:
		return 3
	case model.DbRiskLow:
		return 1
	default:
		return 2
	}
}

func assessSQLSingle(text string, prodEnv bool) SQLAssessment {
	if text == "" {
		return SQLAssessment{RiskLevel: model.DbRiskLow, Ops: []string{"empty"}}
	}
	if hasMultipleStatements(text) {
		return SQLAssessment{RiskLevel: model.DbRiskBlocked, Ops: []string{"multi_statement"}, Blocked: true, Reason: "禁止多语句执行"}
	}
	for _, re := range reBlocked {
		if re.MatchString(text) {
			return SQLAssessment{RiskLevel: model.DbRiskBlocked, Ops: []string{"blocked"}, Blocked: true, Reason: "命中阻断规则"}
		}
	}
	if prodEnv && regexp.MustCompile(`(?i)\bTRUNCATE\b`).MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskBlocked, Ops: []string{"truncate"}, Blocked: true, Reason: "生产环境禁止 TRUNCATE"}
	}
	ops := detectOps(text)
	for _, re := range reHigh {
		if re.MatchString(text) {
			return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops}
		}
	}
	// 无 WHERE 的 DELETE/UPDATE 影响全表，判为高危。
	if (reDeleteStmt.MatchString(text) || reUpdateStmt.MatchString(text)) && !reHasWhere.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops, Reason: "缺少 WHERE 条件，将影响全表"}
	}
	if reDDL.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops}
	}
	if reDML.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskMedium, Ops: ops}
	}
	if reRead.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskLow, Ops: ops}
	}
	return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops, Reason: "无法解析，保守视为高危"}
}

func detectOps(text string) []string {
	keys := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE", "SHOW", "EXPLAIN"}
	var ops []string
	upper := strings.ToUpper(text)
	for _, k := range keys {
		if strings.Contains(upper, k) {
			ops = append(ops, strings.ToLower(k))
		}
	}
	if len(ops) == 0 {
		ops = []string{"unknown"}
	}
	return ops
}

func isReadOnlySQL(sqlText string) bool {
	a := AssessSQL(sqlText, false)
	return a.RiskLevel == model.DbRiskLow && !a.Blocked
}

// normalizeQuerySQL 去掉 USE 语句（目标库已由请求参数指定），仅保留单条只读查询。
func normalizeQuerySQL(sqlText string) (string, error) {
	stmts := splitSQLStatements(strings.TrimSpace(sqlText))
	var queries []string
	useRe := regexp.MustCompile(`(?i)^USE\s+`)
	for _, s := range stmts {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if useRe.MatchString(s) {
			continue
		}
		queries = append(queries, s)
	}
	if len(queries) == 0 {
		return "", constants.ErrBadRequestWithMsg("请输入查询语句")
	}
	if len(queries) > 1 {
		return "", constants.ErrBadRequestWithMsg("查询仅允许单条 SELECT/SHOW/DESCRIBE/EXPLAIN 语句")
	}
	return queries[0], nil
}

func splitSQLStatements(content string) []string {
	var stmts []string
	var buf strings.Builder
	inSingle := false
	inDouble := false
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
		}
		if ch == ';' && !inSingle && !inDouble {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			buf.Reset()
			continue
		}
		buf.WriteByte(ch)
	}
	if s := strings.TrimSpace(buf.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}

func enforceLimit(sqlText string, maxRows int) string {
	text := strings.TrimSpace(sqlText)
	if maxRows <= 0 {
		maxRows = 1000
	}
	upper := strings.ToUpper(text)
	if idx := strings.LastIndex(upper, "LIMIT "); idx >= 0 {
		// 已有 LIMIT：若数值超过上限则追加子查询包装（简单场景直接截断重写）
		return rewriteLimitClause(text, maxRows)
	}
	if reRead.MatchString(text) {
		return fmt.Sprintf("%s LIMIT %d", text, maxRows)
	}
	return text
}

func rewriteLimitClause(text string, maxRows int) string {
	upper := strings.ToUpper(text)
	idx := strings.LastIndex(upper, "LIMIT ")
	if idx < 0 {
		return text
	}
	rest := strings.TrimSpace(text[idx+6:])
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return text
	}
	var n int
	if _, err := fmt.Sscanf(parts[0], "%d", &n); err == nil && n > maxRows {
		return text[:idx] + fmt.Sprintf("LIMIT %d", maxRows)
	}
	return text
}

func scanRows(rows *sql.Rows, maxRows int) ([]string, [][]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	if maxRows <= 0 {
		maxRows = 1000
	}
	var data [][]any
	for rows.Next() {
		if len(data) >= maxRows {
			break
		}
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		for i, v := range vals {
			vals[i] = normalizeCellValue(v)
		}
		data = append(data, vals)
	}
  return cols, data, rows.Err()
}

func nonNilRows(data [][]any) [][]any {
	if data == nil {
		return [][]any{}
	}
	return data
}

func nonNilCols(cols []string) []string {
	if cols == nil {
		return []string{}
	}
	return cols
}

func isInstanceLevelDDL(sqlText string) bool {
	return regexp.MustCompile(`(?i)^\s*CREATE\s+DATABASE\b`).MatchString(strings.TrimSpace(sqlText)) ||
		regexp.MustCompile(`(?i)^\s*DROP\s+DATABASE\b`).MatchString(strings.TrimSpace(sqlText))
}

func normalizeCellValue(v any) any {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339)
	case fmt.Stringer:
		return x.String()
	default:
		return v
	}
}
