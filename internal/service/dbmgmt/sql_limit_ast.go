package dbmgmt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xwb1989/sqlparser"
)

// enforceLimitAST 优先用 AST 改写 SELECT/UNION 的 LIMIT；失败时回退字符串扫描器。
// 禁止把用户原文直接拼进 SELECT * FROM (%s) 子查询（注入面）。
func enforceLimitAST(sqlText string, maxRows int) (string, bool) {
	if maxRows <= 0 {
		maxRows = 1000
	}
	text := strings.TrimRight(strings.TrimSpace(sqlText), "; \t\r\n")
	stmt, err := sqlparser.Parse(text)
	if err != nil {
		return "", false
	}
	switch s := stmt.(type) {
	case *sqlparser.Select:
		capSelectLimit(s, maxRows)
		return sqlparser.String(s), true
	case *sqlparser.Union:
		capUnionLimit(s, maxRows)
		return sqlparser.String(s), true
	case *sqlparser.Show:
		// SHOW 多数不支持 LIMIT；留给扫描器或原样返回
		return "", false
	default:
		return "", false
	}
}

func capSelectLimit(s *sqlparser.Select, maxRows int) {
	if s == nil {
		return
	}
	if s.Limit == nil {
		s.Limit = &sqlparser.Limit{Rowcount: sqlparser.NewIntVal([]byte(strconv.Itoa(maxRows)))}
		return
	}
	s.Limit.Rowcount = capSQLVal(s.Limit.Rowcount, maxRows)
}

func capUnionLimit(u *sqlparser.Union, maxRows int) {
	if u == nil {
		return
	}
	if u.Limit == nil {
		u.Limit = &sqlparser.Limit{Rowcount: sqlparser.NewIntVal([]byte(strconv.Itoa(maxRows)))}
		return
	}
	u.Limit.Rowcount = capSQLVal(u.Limit.Rowcount, maxRows)
}

func capSQLVal(v sqlparser.Expr, maxRows int) sqlparser.Expr {
	if v == nil {
		return sqlparser.NewIntVal([]byte(strconv.Itoa(maxRows)))
	}
	sv, ok := v.(*sqlparser.SQLVal)
	if !ok || sv.Type != sqlparser.IntVal {
		return sqlparser.NewIntVal([]byte(strconv.Itoa(maxRows)))
	}
	n, err := strconv.Atoi(string(sv.Val))
	if err != nil || n <= 0 || n > maxRows {
		return sqlparser.NewIntVal([]byte(strconv.Itoa(maxRows)))
	}
	return v
}

// wrapAsLimitedSubquery 仅在原文已通过只读校验且无法 AST/扫描改写时使用；
// 对内容做括号平衡与分号拒绝，降低拼接注入风险。
func wrapAsLimitedSubquery(text string, maxRows int) (string, error) {
	text = strings.TrimRight(strings.TrimSpace(text), "; \t\r\n")
	if text == "" || strings.Contains(text, ";") {
		return "", fmt.Errorf("refuse subquery wrap: empty or multi-statement")
	}
	depth := 0
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", fmt.Errorf("refuse subquery wrap: unbalanced parentheses")
			}
		}
	}
	if depth != 0 {
		return "", fmt.Errorf("refuse subquery wrap: unbalanced parentheses")
	}
	return fmt.Sprintf("SELECT * FROM (%s) AS _ys_lim LIMIT %d", text, maxRows), nil
}
