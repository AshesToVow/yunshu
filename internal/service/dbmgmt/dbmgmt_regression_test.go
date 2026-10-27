package dbmgmt

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"yunshu/internal/model"
)

func refKeys(refs []queryTableRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, strings.ToLower(r.Schema)+"."+strings.ToLower(r.Table))
	}
	return out
}

func hasRef(refs []queryTableRef, schema, table string) bool {
	for _, r := range refs {
		if strings.EqualFold(r.Schema, schema) && strings.EqualFold(r.Table, table) {
			return true
		}
	}
	return false
}

func TestExtractQueryTableRefs_SubqueryCrossDatabase(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		sql  string
		want [][2]string
	}{
		{
			name: "where in subquery",
			sql:  "SELECT * FROM users WHERE id IN (SELECT uid FROM other_db.orders)",
			want: [][2]string{{"app", "users"}, {"other_db", "orders"}},
		},
		{
			name: "delete with subquery",
			sql:  "DELETE FROM app.users WHERE id IN (SELECT id FROM secret.accounts)",
			want: [][2]string{{"app", "users"}, {"secret", "accounts"}},
		},
		{
			name: "from subquery",
			sql:  "SELECT * FROM (SELECT * FROM inner_db.events) t",
			want: [][2]string{{"inner_db", "events"}},
		},
		{
			name: "union",
			sql:  "SELECT id FROM a.t1 UNION SELECT id FROM b.t2",
			want: [][2]string{{"a", "t1"}, {"b", "t2"}},
		},
		{
			name: "insert select",
			sql:  "INSERT INTO arch.users SELECT * FROM prod.users",
			want: [][2]string{{"arch", "users"}, {"prod", "users"}},
		},
	}
	for _, c := range cases {
		refs := extractQueryTableRefs(c.sql, "app")
		if len(refs) != len(c.want) {
			t.Fatalf("%s: refs=%v want %v", c.name, refKeys(refs), c.want)
		}
		for _, w := range c.want {
			if !hasRef(refs, w[0], w[1]) {
				t.Fatalf("%s: missing %s.%s in %v", c.name, w[0], w[1], refKeys(refs))
			}
		}
	}
}

func TestExtractQueryTableRefs_ExplainUsesInnerStatement(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		sql  string
		want []queryTableRef
	}{
		{name: "explain select", sql: "EXPLAIN SELECT * FROM other.t", want: []queryTableRef{{Schema: "other", Table: "t"}}},
		{name: "explain format=json", sql: "EXPLAIN FORMAT=JSON SELECT * FROM other.t", want: []queryTableRef{{Schema: "other", Table: "t"}}},
		{name: "explain analyze", sql: "EXPLAIN ANALYZE SELECT * FROM app.t", want: []queryTableRef{{Schema: "app", Table: "t"}}},
		{name: "explain update", sql: "EXPLAIN UPDATE other.t SET a=1", want: []queryTableRef{{Schema: "other", Table: "t"}}},
		{name: "explain delete", sql: "EXPLAIN DELETE FROM other.t WHERE id=1", want: []queryTableRef{{Schema: "other", Table: "t"}}},
		{name: "explain shorthand qualified", sql: "EXPLAIN other.t", want: []queryTableRef{{Schema: "other", Table: "t"}}},
		{name: "explain shorthand local", sql: "EXPLAIN t", want: []queryTableRef{{Schema: "app", Table: "t"}}},
		{name: "explain only", sql: "EXPLAIN", want: nil},
	}
	for _, c := range cases {
		refs := extractQueryTableRefs(c.sql, "app")
		if len(refs) != len(c.want) {
			t.Fatalf("%s: refs=%v want %v", c.name, refKeys(refs), c.want)
		}
		for i := range c.want {
			if !strings.EqualFold(refs[i].Schema, c.want[i].Schema) || !strings.EqualFold(refs[i].Table, c.want[i].Table) {
				t.Fatalf("%s: refs[%d]=%+v want %+v", c.name, i, refs[i], c.want[i])
			}
		}
	}
}

func TestExtractQueryTableRefs_ShowCreateTargets(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		sql  string
		want queryTableRef
	}{
		{name: "table qualified", sql: "SHOW CREATE TABLE `db2`.`orders`", want: queryTableRef{Schema: "db2", Table: "orders"}},
		{name: "table local trailing semicolon", sql: "show create table orders;", want: queryTableRef{Schema: "app", Table: "orders"}},
		{name: "view qualified", sql: "SHOW CREATE VIEW db3.v1", want: queryTableRef{Schema: "db3", Table: "v1"}},
		{name: "temporary table", sql: "SHOW CREATE TEMPORARY TABLE t2", want: queryTableRef{Schema: "app", Table: "t2"}},
		{name: "database", sql: "SHOW CREATE DATABASE db4", want: queryTableRef{Schema: "db4"}},
		{name: "schema backticked", sql: "SHOW CREATE SCHEMA `db5`;", want: queryTableRef{Schema: "db5"}},
	}
	for _, c := range cases {
		refs := extractQueryTableRefs(c.sql, "app")
		if len(refs) != 1 {
			t.Fatalf("%s: refs=%v want single %+v", c.name, refKeys(refs), c.want)
		}
		if !strings.EqualFold(refs[0].Schema, c.want.Schema) || !strings.EqualFold(refs[0].Table, c.want.Table) {
			t.Fatalf("%s: got %+v want %+v", c.name, refs[0], c.want)
		}
	}
	// 带多余内容的 SHOW CREATE 不应被当作单对象放行。
	if refs := extractQueryTableRefs("SHOW CREATE TABLE db.t extra", "app"); len(refs) != 0 {
		t.Fatalf("malformed show create must not resolve refs, got %v", refKeys(refs))
	}
}

func TestSplitSQLStatements_CommentsAndQuotes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want int
	}{
		{name: "plain two", in: "SELECT 1; SELECT 2", want: 2},
		{name: "single quote semicolon", in: "SELECT ';' AS a; SELECT 2", want: 2},
		{name: "double quote semicolon", in: `SELECT "a;b" FROM t; SELECT 2`, want: 2},
		{name: "backtick semicolon", in: "SELECT `a;b` FROM t; SELECT 2", want: 2},
		{name: "doubled quote escape", in: "SELECT 'it''s; ok'; SELECT 2", want: 2},
		{name: "backslash escaped quote", in: `SELECT 'a\';b'; SELECT 2`, want: 2},
		{name: "line comment with semicolon", in: "-- c ; here\nSELECT 1; SELECT 2", want: 2},
		{name: "hash comment with semicolon", in: "SELECT 1; # hash ; comment\nSELECT 2", want: 2},
		{name: "block comment with semicolon", in: "SELECT 1 /* a;b */; SELECT 2", want: 2},
		{name: "comment only", in: "/* ; */ -- ;\n# ;", want: 0},
		{name: "empty statements", in: "SELECT 1;;;SELECT 2;", want: 2},
		{name: "unterminated quote", in: "SELECT 'unterminated; SELECT 2", want: 1},
	}
	for _, c := range cases {
		if got := len(splitSQLStatements(c.in)); got != c.want {
			t.Fatalf("%s: got %d statements %v, want %d", c.name, got, splitSQLStatements(c.in), c.want)
		}
	}
}

type timeoutTestError struct{}

func (timeoutTestError) Error() string   { return "i/o timeout" }
func (timeoutTestError) Timeout() bool   { return true }
func (timeoutTestError) Temporary() bool { return true }

type nonTimeoutTestError struct{}

func (nonTimeoutTestError) Error() string   { return "permission denied" }
func (nonTimeoutTestError) Timeout() bool   { return false }
func (nonTimeoutTestError) Temporary() bool { return false }

func TestIsExecutionResultUnknown(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "wrapped deadline", err: fmt.Errorf("exec: %w", context.DeadlineExceeded), want: true},
		{name: "canceled", err: context.Canceled, want: true},
		{name: "net timeout", err: timeoutTestError{}, want: true},
		{name: "net non-timeout", err: nonTimeoutTestError{}, want: false},
		{name: "invalid connection", err: errors.New("invalid connection"), want: true},
		{name: "unexpected eof", err: errors.New("unexpected EOF"), want: true},
		{name: "connection reset", err: errors.New("read tcp: connection reset by peer"), want: true},
		{name: "broken pipe", err: errors.New("write: broken pipe"), want: true},
		{name: "syntax error", err: errors.New("Error 1064: syntax error"), want: false},
		{name: "connection refused", err: errors.New("dial tcp: connection refused"), want: false},
	}
	for _, c := range cases {
		if got := isExecutionResultUnknown(c.err); got != c.want {
			t.Fatalf("%s: got %v want %v", c.name, got, c.want)
		}
	}
	// 确认 net.Error 分支确实可达。
	var netErr net.Error = timeoutTestError{}
	if !netErr.Timeout() {
		t.Fatal("test error must satisfy net.Error")
	}
}

func TestBuildAppUserSQL_NewUserCreatesHostsAndGrants(t *testing.T) {
	t.Parallel()
	req := &model.DbAppUserRequest{
		ApplyType:      model.DbAppUserApplyNewUser,
		MySQLUser:      "appuser",
		MySQLHost:      "%",
		DatabaseName:   "app",
		PrivLevel:      model.DbAppUserPrivDatabase,
		PrivilegesJSON: `["SELECT","INSERT"]`,
		GrantHosts:     "10.0.0.1|10.0.0.2",
	}
	stmts, password, err := buildAppUserSQL(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(password) != 16 {
		t.Fatalf("want 16-char password, got %q", password)
	}
	if len(stmts) != 4 {
		t.Fatalf("want 4 stmts (2 create + 2 grant), got %d: %+v", len(stmts), stmts)
	}
	for _, host := range []string{"10.0.0.1", "10.0.0.2"} {
		var create, grant bool
		for _, s := range stmts {
			if s.Host != host {
				continue
			}
			if strings.Contains(s.SQL, "CREATE USER IF NOT EXISTS 'appuser'@'"+host+"'") && s.CreatesUser {
				create = true
			}
			if strings.Contains(s.SQL, "GRANT SELECT, INSERT ON `app`.*") {
				grant = true
			}
		}
		if !create || !grant {
			t.Fatalf("host %s missing create/grant: %+v", host, stmts)
		}
	}
}

func TestBuildAppUserSQL_AddPrivOnlyCreatesNewHosts(t *testing.T) {
	t.Parallel()
	req := &model.DbAppUserRequest{
		ApplyType:      model.DbAppUserApplyAddPriv,
		MySQLUser:      "appuser",
		MySQLHost:      "10.0.0.1",
		DatabaseName:   "app",
		PrivLevel:      model.DbAppUserPrivDatabase,
		PrivilegesJSON: `["SELECT"]`,
		GrantHosts:     "10.0.0.1|10.0.0.2",
	}
	stmts, password, err := buildAppUserSQL(req)
	if err != nil {
		t.Fatal(err)
	}
	if password == "" {
		t.Fatal("new host requires generated password")
	}
	var creates, grants int
	for _, s := range stmts {
		if s.CreatesUser {
			creates++
			if s.Host != "10.0.0.2" {
				t.Fatalf("must not recreate existing host, got %+v", s)
			}
		} else {
			grants++
		}
	}
	if creates != 1 || grants != 2 {
		t.Fatalf("want 1 create + 2 grants, got %d/%d: %+v", creates, grants, stmts)
	}
}

func TestBuildAppUserSQL_AddIPCreatesUserForEachHost(t *testing.T) {
	t.Parallel()
	req := &model.DbAppUserRequest{
		ApplyType:      model.DbAppUserApplyAddIP,
		MySQLUser:      "appuser",
		MySQLHost:      "%",
		DatabaseName:   "app",
		PrivLevel:      model.DbAppUserPrivDatabase,
		PrivilegesJSON: `["SELECT"]`,
		GrantHosts:     "10.0.0.9",
	}
	stmts, password, err := buildAppUserSQL(req)
	if err != nil {
		t.Fatal(err)
	}
	if password == "" {
		t.Fatal("add_ip must generate password for the new host")
	}
	if len(stmts) != 2 || !stmts[0].CreatesUser || stmts[1].CreatesUser {
		t.Fatalf("want create+grant for new ip, got %+v", stmts)
	}
}

func TestBuildAppUserSQL_RevokeNeedsNoPassword(t *testing.T) {
	t.Parallel()
	req := &model.DbAppUserRequest{
		ApplyType:      model.DbAppUserApplyRevoke,
		MySQLUser:      "appuser",
		MySQLHost:      "10.0.0.1",
		DatabaseName:   "app",
		PrivLevel:      model.DbAppUserPrivDatabase,
		PrivilegesJSON: `["SELECT"]`,
		GrantHosts:     "10.0.0.1",
	}
	stmts, password, err := buildAppUserSQL(req)
	if err != nil {
		t.Fatal(err)
	}
	if password != "" {
		t.Fatalf("revoke must not generate password, got %q", password)
	}
	if len(stmts) != 1 || !strings.HasPrefix(stmts[0].SQL, "REVOKE SELECT ON `app`.*") {
		t.Fatalf("unexpected revoke stmts: %+v", stmts)
	}
}

func TestBuildAppUserSQL_EmptyPrivilegesRejected(t *testing.T) {
	t.Parallel()
	req := &model.DbAppUserRequest{
		ApplyType:      model.DbAppUserApplyNewUser,
		MySQLUser:      "appuser",
		MySQLHost:      "%",
		DatabaseName:   "app",
		PrivLevel:      model.DbAppUserPrivDatabase,
		PrivilegesJSON: `[]`,
		GrantHosts:     "10.0.0.1",
	}
	if _, _, err := buildAppUserSQL(req); err == nil {
		t.Fatal("empty privileges must be rejected")
	}
}

func TestEscapeMySQLString_BackslashBeforeQuote(t *testing.T) {
	t.Parallel()
	if got := escapeMySQLString(`a\`); got != `a\\` {
		t.Fatalf("trailing backslash: got %q", got)
	}
	if got := escapeMySQLString(`a'`); got != `a''` {
		t.Fatalf("single quote: got %q", got)
	}
	if got := escapeMySQLString(`a\'`); got != `a\\''` {
		t.Fatalf("backslash+quote: got %q", got)
	}
	if got := escapeMySQLString("a\x00b"); got != "ab" {
		t.Fatalf("NUL must be stripped, got %q", got)
	}
}

func TestParseOptionalExpiresAt(t *testing.T) {
	t.Parallel()
	if got, err := parseOptionalExpiresAt(nil); err != nil || got != nil {
		t.Fatalf("nil: got %v err %v", got, err)
	}
	blank := "  "
	if got, err := parseOptionalExpiresAt(&blank); err != nil || got != nil {
		t.Fatalf("blank: got %v err %v", got, err)
	}
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	if _, err := parseOptionalExpiresAt(&past); err == nil {
		t.Fatal("past expiry must be rejected")
	}
	bad := "2026-13-40"
	if _, err := parseOptionalExpiresAt(&bad); err == nil {
		t.Fatal("invalid format must be rejected")
	}
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	got, err := parseOptionalExpiresAt(&future)
	if err != nil || got == nil || !got.After(time.Now()) {
		t.Fatalf("future: got %v err %v", got, err)
	}
}

func TestGrantStillValid_ExpiryBoundary(t *testing.T) {
	t.Parallel()
	now := time.Now()
	if !grantStillValid(model.DbAccessGrant{}, now) {
		t.Fatal("nil expiry means permanent")
	}
	expired := now.Add(-time.Second)
	if grantStillValid(model.DbAccessGrant{ExpiresAt: &expired}, now) {
		t.Fatal("expired grant must be invalid")
	}
	valid := now.Add(time.Second)
	if !grantStillValid(model.DbAccessGrant{ExpiresAt: &valid}, now) {
		t.Fatal("future expiry must be valid")
	}
}

func TestIsFinalAccessApprovalStep(t *testing.T) {
	t.Parallel()
	steps := []model.DbAccessRequestStep{
		{ID: 1, SortOrder: 1, Status: model.DbApprovalStepPending},
		{ID: 2, SortOrder: 2, Status: model.DbApprovalStepPending},
	}
	if isFinalAccessApprovalStep(steps, nil) {
		t.Fatal("nil current step must not be final")
	}
	if isFinalAccessApprovalStep(steps, &steps[0]) {
		t.Fatal("first of two pending steps must not be final")
	}
	if !isFinalAccessApprovalStep(steps, &steps[1]) {
		t.Fatal("last step must be final")
	}
	single := []model.DbAccessRequestStep{{ID: 3, SortOrder: 1, Status: model.DbApprovalStepPending}}
	if !isFinalAccessApprovalStep(single, &single[0]) {
		t.Fatal("single step must be final")
	}
}
