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
		regexp.MustCompile(`(?i)\bLOAD_FILE\s*\(`),
		regexp.MustCompile(`(?i)\bINTO\s+DUMPFILE\b`),
		regexp.MustCompile(`(?i)\bSET\s+GLOBAL\b`),
		regexp.MustCompile(`(?i)\bFLUSH\s+PRIVILEGES\b`),
		regexp.MustCompile(`(?i)\bLOAD\s+DATA\b`),
		regexp.MustCompile(`(?i)\bCOPY\s+.+\s+TO\s+PROGRAM\b`),
		regexp.MustCompile(`(?i)\bCOPY\s+.+\s+FROM\s+PROGRAM\b`),
		regexp.MustCompile(`(?i)\bCOPY\s+.+\s+TO\s+['"]`),
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
	reReadPrefix = regexp.MustCompile(`(?i)^\s*(SELECT|SHOW|DESCRIBE|DESC|EXPLAIN|WITH)\b`)
	// 文件导出 / 危险副作用：即使以 SELECT/WITH 开头也不得视为只读。
	reFileExport  = regexp.MustCompile(`(?i)\bINTO\s+(OUTFILE|DUMPFILE)\b`)
	reWriteInBody = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|REPLACE|CREATE|ALTER|DROP|TRUNCATE|RENAME|CALL|LOAD\s+DATA|LOCK\s+TABLES)\b`)
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

// stripSQLComments 将注释替换为空格，避免 INTO/**/OUTFILE 一类绕过。
// 字符串/反引号内内容原样保留；MySQL 可执行注释 /*!...*/ 展开为内部 SQL。
func stripSQLComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSingle, inDouble, inBacktick := false, false, false
	i := 0
	for i < len(s) {
		ch := s[i]
		writeEscaped := func(quote byte, flag *bool) bool {
			if !*flag {
				return false
			}
			b.WriteByte(ch)
			if quote != '`' && ch == '\\' && i+1 < len(s) {
				b.WriteByte(s[i+1])
				i += 2
				return true
			}
			if ch == quote {
				if i+1 < len(s) && s[i+1] == quote {
					b.WriteByte(s[i+1])
					i += 2
					return true
				}
				*flag = false
			}
			i++
			return true
		}
		if writeEscaped('\'', &inSingle) || writeEscaped('"', &inDouble) || writeEscaped('`', &inBacktick) {
			continue
		}
		if i+1 < len(s) && ch == '-' && s[i+1] == '-' {
			b.WriteByte(' ')
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if ch == '#' {
			b.WriteByte(' ')
			i++
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < len(s) && ch == '/' && s[i+1] == '*' {
			executable := i+2 < len(s) && s[i+2] == '!'
			i += 2
			if executable {
				i++
				for i < len(s) && s[i] >= '0' && s[i] <= '9' {
					i++
				}
			} else {
				b.WriteByte(' ')
			}
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				if executable {
					b.WriteByte(s[i])
				}
				i++
			}
			if i+1 < len(s) {
				i += 2
			} else {
				i = len(s)
			}
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inBacktick = true
		}
		b.WriteByte(ch)
		i++
	}
	return b.String()
}

func hasMultipleStatements(sqlText string) bool {
	return reMultiStmt.MatchString(strings.TrimSpace(sqlText))
}

// AssessSQL 评估单条或本地规则下的 SQL 风险（查询场景等）。
func AssessSQL(sqlText string, prodEnv bool) SQLAssessment {
	return assessSQLSingle(strings.TrimSpace(stripSQLComments(sqlText)), prodEnv)
}

// AssessSQLForWrite 写操作风险评估；启用 goInception 时允许多语句批量提交。
func AssessSQLForWrite(sqlText string, prodEnv, viaGoInception bool) SQLAssessment {
	text := strings.TrimSpace(stripSQLComments(sqlText))
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
	text = strings.TrimSpace(stripSQLComments(text))
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
	if reFileExport.MatchString(text) {
		return SQLAssessment{
			RiskLevel: model.DbRiskBlocked, Ops: ops, Blocked: true,
			Reason: "禁止 SELECT INTO OUTFILE/DUMPFILE",
		}
	}
	if reDDL.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops}
	}
	if reDML.MatchString(text) {
		return SQLAssessment{RiskLevel: model.DbRiskMedium, Ops: ops}
	}
	if reReadPrefix.MatchString(text) {
		// WITH/SELECT 前缀不能单独判定为只读：CTE 内可跟 DML/DDL。
		if reWriteInBody.MatchString(text) {
			return SQLAssessment{RiskLevel: model.DbRiskHigh, Ops: ops, Reason: "语句含写操作关键字"}
		}
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
	for i := range len(content) {
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

// reLimitAppendable 只有这些语句可以安全追加 LIMIT 子句；
// DESCRIBE/DESC/EXPLAIN 不支持 LIMIT，追加会造成语法错误，且其结果集天然有限。
var reLimitAppendable = regexp.MustCompile(`(?i)^\s*(SELECT|WITH|SHOW|TABLE)\b`)

func enforceLimit(sqlText string, maxRows int) string {
	text := strings.TrimSpace(sqlText)
	if maxRows <= 0 {
		maxRows = 1000
	}
	if rewritten, ok := enforceLimitAST(text, maxRows); ok {
		return rewritten
	}
	// AST 失败时回退到引号/注释感知扫描器。
	if idx := findTrailingLimit(text); idx >= 0 {
		return rewriteLimitClauseAt(text, idx, maxRows)
	}
	if reLimitAppendable.MatchString(text) {
		return fmt.Sprintf("%s\nLIMIT %d", strings.TrimRight(text, "; \t\r\n"), maxRows)
	}
	return text
}

func isSQLIdentByte(c byte) bool {
	switch {
	case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		return true
	case c == '_' || c == '$' || c >= 0x80:
		return true
	}
	return false
}

// findTrailingLimit 定位语句最外层（括号深度 0）最后一个真实 LIMIT 关键字的起始下标，
// 返回 -1 表示不存在可改写的 LIMIT 子句。
// 跳过字符串字面量、反引号标识符与注释；MySQL 可执行注释 /*!12345 ... */ 的内容
// 会被真实执行，故其内部同样参与判定（与 stripSQLComments 语义保持一致）。
func findTrailingLimit(s string) int {
	found, depth, i := -1, 0, 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == '\'' || ch == '"' || ch == '`':
			quote := ch
			i++
			for i < len(s) {
				if quote != '`' && s[i] == '\\' {
					i += 2
					continue
				}
				if s[i] == quote {
					if i+1 < len(s) && s[i+1] == quote { // '' 双写转义
						i += 2
						continue
					}
					i++
					break
				}
				i++
			}
			continue
		case ch == '-' && i+1 < len(s) && s[i+1] == '-', ch == '#':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		case ch == '/' && i+1 < len(s) && s[i+1] == '*':
			if i+2 < len(s) && s[i+2] == '!' { // 可执行注释：跳过 /*! 与版本号，内容继续扫描
				i += 3
				for i < len(s) && s[i] >= '0' && s[i] <= '9' {
					i++
				}
				continue
			}
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			} else {
				i = len(s)
			}
			continue
		case ch == '(':
			depth++
		case ch == ')':
			if depth > 0 {
				depth--
			}
		case (ch == 'L' || ch == 'l') && depth == 0 &&
			i+5 <= len(s) && strings.EqualFold(s[i:i+5], "LIMIT"):
			prevOK := i == 0 || !isSQLIdentByte(s[i-1])
			nextOK := i+5 >= len(s) || !isSQLIdentByte(s[i+5])
			if prevOK && nextOK {
				found = i
				i += 5
				continue
			}
		}
		i++
	}
	return found
}

var reLimitRest = regexp.MustCompile(`(?i)^(\d+)(?:\s*,\s*(\d+))?(?:\s+OFFSET\s+(\d+))?`)

func rewriteLimitClause(text string, maxRows int) string {
	idx := findTrailingLimit(text)
	if idx < 0 {
		return text
	}
	return rewriteLimitClauseAt(text, idx, maxRows)
}

// rewriteLimitClauseAt 在已定位的 LIMIT 关键字处收敛行数上限。
func rewriteLimitClauseAt(text string, idx, maxRows int) string {
	if maxRows <= 0 {
		maxRows = 1000
	}
	if idx < 0 || idx+5 > len(text) {
		return text
	}
	head := text[:idx]
	rest := strings.TrimSpace(text[idx+5:])
	m := reLimitRest.FindStringSubmatch(rest)
	if m == nil {
		// 无法安全解析 LIMIT 尾部：禁止把用户原文直接拼进子查询（注入面）。
		if wrapped, err := wrapAsLimitedSubquery(text, maxRows); err == nil {
			return wrapped
		}
		// 最后兜底：追加硬 LIMIT（可能产生语法错误，由上层执行失败拒绝）
		return fmt.Sprintf("%s\nLIMIT %d", strings.TrimRight(strings.TrimSpace(text), ";"), maxRows)
	}
	consumed := m[0]
	tail := rest[len(consumed):]
	capCount := func(raw string) int {
		var n int
		fmt.Sscanf(raw, "%d", &n)
		if n <= 0 || n > maxRows {
			return maxRows
		}
		return n
	}
	switch {
	case m[2] != "":
		return head + fmt.Sprintf("LIMIT %s, %d", m[1], capCount(m[2])) + tail
	case m[3] != "":
		return head + fmt.Sprintf("LIMIT %d OFFSET %s", capCount(m[1]), m[3]) + tail
	default:
		return head + fmt.Sprintf("LIMIT %d", capCount(m[1])) + tail
	}
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
