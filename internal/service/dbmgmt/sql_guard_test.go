package dbmgmt

import (
	"strings"
	"testing"
)

func TestAssessSQL_WithCTEContainingDML(t *testing.T) {
	a := AssessSQL("WITH cte AS (SELECT 1) DELETE FROM users WHERE id = 1", false)
	if a.RiskLevel == "low" || a.Blocked {
		// blocked is ok for other reasons; low is the bug
	}
	if a.RiskLevel == "low" {
		t.Fatalf("WITH+DELETE must not be low risk, got %+v", a)
	}
}

func TestAssessSQL_SelectIntoOutfileBlocked(t *testing.T) {
	a := AssessSQL("SELECT * FROM t INTO OUTFILE '/tmp/x.csv'", false)
	if !a.Blocked {
		t.Fatalf("INTO OUTFILE must be blocked, got %+v", a)
	}
}

func TestAssessSQL_PlainSelectLow(t *testing.T) {
	a := AssessSQL("SELECT id, name FROM users WHERE id = 1", false)
	if a.Blocked || a.RiskLevel != "low" {
		t.Fatalf("plain SELECT should be low, got %+v", a)
	}
}

func TestIsReadOnlySQL(t *testing.T) {
	if isReadOnlySQL("WITH x AS (SELECT 1) UPDATE t SET a=1") {
		t.Fatal("WITH+UPDATE must not be read-only")
	}
	if !isReadOnlySQL("SHOW TABLES") {
		t.Fatal("SHOW TABLES should be read-only")
	}
}

func TestAssessSQL_CommentBypassIntoOutfile(t *testing.T) {
	t.Parallel()
	cases := []string{
		"SELECT * FROM t INTO/**/OUTFILE '/tmp/x.csv'",
		"SELECT * FROM t INTO --\n OUTFILE '/tmp/x.csv'",
		"SELECT /*!32302 1, */ * FROM t INTO OUTFILE '/tmp/x.csv'",
	}
	for _, sql := range cases {
		a := AssessSQL(sql, false)
		if !a.Blocked {
			t.Fatalf("%q must be blocked after comment strip, got %+v", sql, a)
		}
	}
}

func TestAssessSQL_LoadDataAndCopyBlocked(t *testing.T) {
	t.Parallel()
	cases := []string{
		"LOAD DATA INFILE '/tmp/x' INTO TABLE t",
		"COPY t TO PROGRAM 'cat /etc/passwd'",
		"COPY t TO '/tmp/out.csv'",
	}
	for _, sql := range cases {
		a := AssessSQL(sql, false)
		if !a.Blocked {
			t.Fatalf("%q must be blocked, got %+v", sql, a)
		}
	}
}

func TestRewriteLimitClause_OffsetCommaCount(t *testing.T) {
	t.Parallel()
	got := rewriteLimitClause("SELECT * FROM t LIMIT 1, 999999", 1000)
	if got != "SELECT * FROM t LIMIT 1, 1000" {
		t.Fatalf("got %q", got)
	}
	got = rewriteLimitClause("SELECT * FROM t LIMIT 5000 OFFSET 10", 1000)
	if got != "SELECT * FROM t LIMIT 1000 OFFSET 10" {
		t.Fatalf("got %q", got)
	}
	got = rewriteLimitClause("SELECT * FROM t LIMIT 50", 1000)
	if got != "SELECT * FROM t LIMIT 50" {
		t.Fatalf("small limit should keep, got %q", got)
	}
}

func TestStripSQLComments_PreservesStringLiteral(t *testing.T) {
	t.Parallel()
	got := stripSQLComments("SELECT 'a--b' FROM t /*c*/ WHERE x=1")
	if !strings.Contains(got, "'a--b'") {
		t.Fatalf("string literal must be preserved, got %q", got)
	}
	if strings.Contains(got, "/*c*/") {
		t.Fatalf("block comment must be stripped, got %q", got)
	}
}

func TestEnforceLimit_IgnoresFakeLimitKeyword(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, in, want string }{
		{
			name: "string literal",
			in:   "SELECT 'limit 9' FROM t",
			want: "SELECT 'limit 9' FROM t\nLIMIT 100",
		},
		{
			name: "line comment",
			in:   "SELECT a FROM t -- limit 9",
			want: "SELECT a FROM t -- limit 9\nLIMIT 100",
		},
		{
			name: "block comment",
			in:   "SELECT a FROM t /* LIMIT 9 */",
			want: "SELECT a FROM t /* LIMIT 9 */\nLIMIT 100",
		},
		{
			name: "identifier prefix",
			in:   "SELECT limit_col FROM t",
			want: "SELECT limit_col FROM t\nLIMIT 100",
		},
		{
			name: "backtick identifier",
			in:   "SELECT `limit 9` FROM t",
			want: "SELECT `limit 9` FROM t\nLIMIT 100",
		},
		{
			name: "subquery limit is not outermost",
			in:   "SELECT * FROM (SELECT a FROM t LIMIT 999999) x",
			want: "SELECT * FROM (SELECT a FROM t LIMIT 999999) x\nLIMIT 100",
		},
		{
			name: "trailing semicolon",
			in:   "SELECT a FROM t;",
			want: "SELECT a FROM t\nLIMIT 100",
		},
	}
	for _, c := range cases {
		if got := enforceLimit(c.in, 100); got != c.want {
			t.Fatalf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestEnforceLimit_CapsRealLimit(t *testing.T) {
	t.Parallel()
	cases := []struct{ in, want string }{
		{"SELECT a FROM t LIMIT 999999", "SELECT a FROM t LIMIT 100"},
		{"SELECT a FROM t limit 999999", "SELECT a FROM t LIMIT 100"},
		{"SELECT a FROM t LIMIT 10", "SELECT a FROM t LIMIT 10"},
		{"SELECT a FROM t LIMIT 5, 999999", "SELECT a FROM t LIMIT 5, 100"},
		{"SELECT a FROM t LIMIT 999999 OFFSET 20", "SELECT a FROM t LIMIT 100 OFFSET 20"},
		// 外层 LIMIT 存在时以外层为准，子查询内的巨大 LIMIT 不影响改写位置
		{"SELECT * FROM (SELECT a FROM t LIMIT 999999) x LIMIT 999999", "SELECT * FROM (SELECT a FROM t LIMIT 999999) x LIMIT 100"},
	}
	for _, c := range cases {
		if got := enforceLimit(c.in, 100); got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}

func TestEnforceLimit_NonAppendableKeptIntact(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"DESCRIBE users",
		"DESC users",
		"EXPLAIN SELECT a FROM t",
	} {
		if got := enforceLimit(in, 100); got != in {
			t.Fatalf("%q must stay intact, got %q", in, got)
		}
	}
}

func TestFindTrailingLimit(t *testing.T) {
	t.Parallel()
	if idx := findTrailingLimit("SELECT 'LIMIT 1' FROM t"); idx != -1 {
		t.Fatalf("literal LIMIT must be ignored, got %d", idx)
	}
	if idx := findTrailingLimit("SELECT limitless FROM t"); idx != -1 {
		t.Fatalf("identifier LIMIT must be ignored, got %d", idx)
	}
	// MySQL 可执行注释内的 LIMIT 会真实生效，必须被识别
	if idx := findTrailingLimit("SELECT a FROM t /*!40001 LIMIT 5 */"); idx < 0 {
		t.Fatalf("executable comment LIMIT must be detected")
	}
	got := findTrailingLimit("SELECT a FROM t LIMIT 5")
	if got != strings.LastIndex("SELECT a FROM t LIMIT 5", "LIMIT") {
		t.Fatalf("unexpected index %d", got)
	}
}
