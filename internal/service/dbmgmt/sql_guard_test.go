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
