package dbmgmt

import "testing"

func TestExtractTableRefsAST_SelectJoin(t *testing.T) {
	refs, ok := extractTableRefsAST("SELECT a.id, b.name FROM users a JOIN orders b ON a.id=b.uid", "db1")
	if !ok {
		t.Fatal("expected AST ok")
	}
	if len(refs) < 2 {
		t.Fatalf("want >=2 tables, got %#v", refs)
	}
	found := map[string]bool{}
	for _, r := range refs {
		found[r.Table] = true
	}
	if !found["users"] || !found["orders"] {
		t.Fatalf("missing tables: %#v", refs)
	}
}

func TestExtractQueryTableRefs_PrefersAST(t *testing.T) {
	refs := extractQueryTableRefs("SELECT * FROM `acct`.`user_profile` WHERE id=1", "other")
	if len(refs) != 1 {
		t.Fatalf("want 1 ref, got %#v", refs)
	}
	if refs[0].Schema != "acct" || refs[0].Table != "user_profile" {
		t.Fatalf("unexpected ref %#v", refs[0])
	}
}
