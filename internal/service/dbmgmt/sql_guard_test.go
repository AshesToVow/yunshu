package dbmgmt

import "testing"

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
