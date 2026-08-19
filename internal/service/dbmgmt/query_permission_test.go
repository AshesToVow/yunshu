package dbmgmt

import (
	"testing"

	"yunshu/internal/model"
)

func TestGrantCoversQuery_UnknownTableRequiresDBLevel(t *testing.T) {
	t.Parallel()
	g := modelGrant("appdb", []string{"users"}, true)
	if ok, _ := grantCoversQuery(g, "appdb", ""); ok {
		t.Fatal("table-scoped grant must not cover unknown/empty table")
	}
	if ok, _ := grantCoversQuery(g, "appdb", "users"); !ok {
		t.Fatal("expected cover users")
	}
	if ok, _ := grantCoversQuery(g, "other", "users"); ok {
		t.Fatal("must not cover other database")
	}

	whole := modelGrant("appdb", nil, true)
	if ok, _ := grantCoversQuery(whole, "appdb", ""); !ok {
		t.Fatal("db-level grant should cover unknown table")
	}
	star := modelGrant("appdb", []string{"*"}, true)
	if ok, _ := grantCoversQuery(star, "appdb", "any"); !ok {
		t.Fatal("* should cover any table")
	}
}

func TestGrantMatchesDatabase(t *testing.T) {
	t.Parallel()
	instWide := modelGrant("", nil, true)
	if !grantMatchesDatabase(instWide, "x") || !grantMatchesDatabase(instWide, "") {
		t.Fatal("instance-wide should match any target")
	}
	named := modelGrant("appdb", nil, true)
	if grantMatchesDatabase(named, "") {
		t.Fatal("named grant must not match empty target")
	}
	if !grantMatchesDatabase(named, "appdb") || grantMatchesDatabase(named, "other") {
		t.Fatal("named grant matching failed")
	}
}

func TestNeedsApproval_RequireTicketForAllWrites(t *testing.T) {
	t.Parallel()
	s := &Service{}
	inst := &model.DbInstance{Env: model.DbEnvDev, RequireTicketForDML: true}
	a := SQLAssessment{RiskLevel: model.DbRiskLow}
	if !s.needsApproval(inst, a, configResolved{}) {
		t.Fatal("RequireTicketForDML should force approval even for low risk")
	}
	inst.RequireTicketForDML = false
	if s.needsApproval(inst, a, configResolved{}) {
		t.Fatal("low risk without flags should not require approval")
	}
}

func TestNeedsApproval_ProdForceAllWrites(t *testing.T) {
	t.Parallel()
	s := &Service{}
	inst := &model.DbInstance{Env: model.DbEnvProd, RequireTicketForDML: false}
	a := SQLAssessment{RiskLevel: model.DbRiskLow}
	if !s.needsApproval(inst, a, configResolved{ProdForceApproval: true}) {
		t.Fatal("prod + ProdForceApproval must require ticket even for low risk")
	}
	if s.needsApproval(inst, a, configResolved{ProdForceApproval: false}) {
		t.Fatal("prod without force and low risk should not require ticket")
	}
}

func modelGrant(db string, tables []string, canQuery bool) model.DbAccessGrant {
	g := model.DbAccessGrant{DatabaseName: db, CanQuery: canQuery}
	if tables == nil {
		return g
	}
	b := "["
	for i, t := range tables {
		if i > 0 {
			b += ","
		}
		b += `"` + t + `"`
	}
	b += "]"
	g.TableNamesJSON = b
	return g
}
