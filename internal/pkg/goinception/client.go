package goinception

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	SyntaxOther = 0
	SyntaxDDL   = 1
	SyntaxDML   = 2

	ErrLevelOK      = 0
	ErrLevelWarning = 1
	ErrLevelError   = 2
)

// Target 目标 MySQL 实例连接信息（goInception 代理执行时使用）。
type Target struct {
	Host     string
	Port     int
	Username string
	Password string
}

// ReviewRow goInception 审核/执行结果单行。
type ReviewRow struct {
	OrderID       int    `json:"order_id"`
	Stage         string `json:"stage"`
	ErrorLevel    int    `json:"error_level"`
	StageStatus   string `json:"stage_status"`
	ErrorMessage  string `json:"error_message"`
	SQL           string `json:"sql"`
	AffectedRows  int64  `json:"affected_rows"`
	Sequence      string `json:"sequence,omitempty"`
	BackupDBName  string `json:"backup_dbname,omitempty"`
	ExecuteTime   string `json:"execute_time,omitempty"`
	SQLSHA1       string `json:"sqlsha1,omitempty"`
	BackupTime    string `json:"backup_time,omitempty"`
}

// ReviewSet 审核或执行结果集。
type ReviewSet struct {
	Rows         []ReviewRow `json:"rows"`
	ErrorCount   int         `json:"error_count"`
	WarningCount int         `json:"warning_count"`
	SyntaxType   int         `json:"syntax_type"`
	Error        string      `json:"error,omitempty"`
	FullSQL      string      `json:"full_sql,omitempty"`
}

// Client goInception 审核/执行客户端。
type Client struct {
	Host    string
	Port    int
	Timeout time.Duration
}

func NewClient(host string, port int) *Client {
	if port <= 0 {
		port = 4000
	}
	return &Client{Host: strings.TrimSpace(host), Port: port, Timeout: 30 * time.Second}
}

func (c *Client) Check(ctx context.Context, target Target, dbName, sqlText string) (*ReviewSet, error) {
	header := fmt.Sprintf(`/*--user='%s';--password='%s';--host='%s';--port=%d;--check=1;*/`,
		escapeInceptionValue(target.Username),
		escapeInceptionValue(target.Password),
		escapeInceptionValue(target.Host),
		target.Port,
	)
	body := buildInceptionBody(dbName, sqlText)
	return c.run(ctx, header+body)
}

func (c *Client) Execute(ctx context.Context, target Target, dbName, sqlText string, backup bool) (*ReviewSet, error) {
	backupFlag := "--backup=0"
	if backup {
		backupFlag = "--backup=1"
	}
	header := fmt.Sprintf(`/*--user='%s';--password='%s';--host='%s';--port=%d;--execute=1;--ignore-warnings=1;%s;--sleep=200;--sleep_rows=100*/`,
		escapeInceptionValue(target.Username),
		escapeInceptionValue(target.Password),
		escapeInceptionValue(target.Host),
		target.Port,
		backupFlag,
	)
	body := buildInceptionBody(dbName, sqlText)
	rs, err := c.run(ctx, header+body)
	if err != nil {
		return rs, err
	}
	for _, row := range rs.Rows {
		if row.ErrorLevel >= ErrLevelWarning && !strings.Contains(row.StageStatus, "Execute Successfully") {
			if rs.Error == "" {
				rs.Error = fmt.Sprintf("第 %d 行: %s", row.OrderID, row.ErrorMessage)
			}
			if row.ErrorLevel >= ErrLevelError {
				rs.ErrorCount++
			}
		}
	}
	return rs, nil
}

func buildInceptionBody(dbName, sqlText string) string {
	dbName = strings.TrimSpace(dbName)
	sqlText = strings.TrimSpace(sqlText)
	sqlText = strings.TrimRight(sqlText, ";")
	var b strings.Builder
	b.WriteString("\ninception_magic_start;\n")
	if dbName != "" {
		b.WriteString("use `")
		b.WriteString(strings.ReplaceAll(dbName, "`", "``"))
		b.WriteString("`;\n")
	}
	b.WriteString(sqlText)
	b.WriteString(";\ninception_magic_commit;")
	return b.String()
}

func (c *Client) run(ctx context.Context, inceptionSQL string) (*ReviewSet, error) {
	if c.Host == "" {
		return nil, fmt.Errorf("goInception 未配置 host")
	}
	dsn := fmt.Sprintf("%s:%d@tcp(%s:%d)/?charset=utf8mb4&parseTime=true&timeout=10s&readTimeout=%ds&writeTimeout=%ds",
		"inception", 0, c.Host, c.Port, int(c.Timeout.Seconds()), int(c.Timeout.Seconds()))
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	qctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	rows, err := db.QueryContext(qctx, inceptionSQL)
	if err != nil {
		return &ReviewSet{Error: err.Error(), FullSQL: inceptionSQL}, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := &ReviewSet{FullSQL: inceptionSQL, SyntaxType: SyntaxDML}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := scanReviewRowByColumns(cols, vals)
		out.Rows = append(out.Rows, row)
		switch row.ErrorLevel {
		case ErrLevelWarning:
			out.WarningCount++
		case ErrLevelError:
			out.ErrorCount++
		}
		if out.SyntaxType == SyntaxDML && isDDLStatement(row.SQL) {
			out.SyntaxType = SyntaxDDL
		}
	}
	if err := rows.Err(); err != nil {
		out.Error = err.Error()
		return out, err
	}
	return out, nil
}

func scanReviewRowByColumns(cols []string, vals []any) ReviewRow {
	get := func(i int) string {
		if i >= len(vals) || vals[i] == nil {
			return ""
		}
		switch v := vals[i].(type) {
		case []byte:
			return string(v)
		case int64:
			return fmt.Sprintf("%d", v)
		case int:
			return fmt.Sprintf("%d", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	toInt := func(s string) int {
		var n int
		fmt.Sscanf(s, "%d", &n)
		return n
	}
	toInt64 := func(s string) int64 {
		var n int64
		fmt.Sscanf(s, "%d", &n)
		return n
	}
	field := map[string]string{}
	for i, col := range cols {
		field[strings.ToLower(strings.TrimSpace(col))] = get(i)
	}
	pick := func(names ...string) string {
		for _, n := range names {
			if v := strings.TrimSpace(field[strings.ToLower(n)]); v != "" {
				return v
			}
		}
		return ""
	}
	row := ReviewRow{
		OrderID:      toInt(pick("order_id")),
		Stage:        pick("stage"),
		ErrorLevel:   toInt(pick("error_level")),
		StageStatus:  pick("stage_status"),
		ErrorMessage: pick("error_message"),
		SQL:          pick("sql"),
		AffectedRows: toInt64(pick("affected_rows")),
		Sequence:     pick("sequence", "opid_time"),
		BackupDBName: pick("backup_dbname"),
		ExecuteTime:  pick("execute_time"),
		SQLSHA1:      pick("sqlsha1"),
		BackupTime:   pick("backup_time"),
	}
	if row.OrderID == 0 && len(vals) > 0 {
		return scanReviewRow(vals)
	}
	return row
}

func scanReviewRow(vals []any) ReviewRow {
	get := func(i int) string {
		if i >= len(vals) || vals[i] == nil {
			return ""
		}
		switch v := vals[i].(type) {
		case []byte:
			return string(v)
		case int64:
			return fmt.Sprintf("%d", v)
		case int:
			return fmt.Sprintf("%d", v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	toInt := func(s string) int {
		var n int
		fmt.Sscanf(s, "%d", &n)
		return n
	}
	toInt64 := func(s string) int64 {
		var n int64
		fmt.Sscanf(s, "%d", &n)
		return n
	}
	row := ReviewRow{
		OrderID:      toInt(get(0)),
		Stage:        get(1),
		ErrorLevel:   toInt(get(2)),
		StageStatus:  get(3),
		ErrorMessage: get(4),
		SQL:          get(5),
		AffectedRows: toInt64(get(6)),
	}
	if len(vals) > 7 {
		row.Sequence = get(7)
	}
	if len(vals) > 8 {
		row.BackupDBName = get(8)
	}
	if len(vals) > 9 {
		row.ExecuteTime = get(9)
	}
	if len(vals) > 10 {
		row.SQLSHA1 = get(10)
	}
	if len(vals) > 11 {
		row.BackupTime = get(11)
	}
	return row
}

var reDDL = regexp.MustCompile(`(?i)^\s*(CREATE|ALTER|DROP|RENAME|TRUNCATE)\b`)

func isDDLStatement(sql string) bool {
	return reDDL.MatchString(strings.TrimSpace(sql))
}

func escapeInceptionValue(s string) string {
	return strings.ReplaceAll(s, `'`, `\'`)
}
