package repository

import (
	"context"
	"testing"
	"time"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDbmgmtTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(
		&model.DbSqlTicket{},
		&model.DbSqlTicketStep{},
		&model.DbAccessRequest{},
		&model.DbAccessRequestStep{},
		&model.DbAccessGrant{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestClaimSqlTicketStep_LegacyApprovalOnlyOnce(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_claim_ticket_step")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()
	now := time.Now()

	ticket := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Status: model.DbTicketStatusPendingApproval}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	step := model.DbSqlTicketStep{TicketID: ticket.ID, StageKey: "dba_lead", StageName: "DBA", SortOrder: 1, Status: model.DbApprovalStepPending}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}

	n, err := repo.ClaimSqlTicketStep(ctx, step.ID, ticket.ID, map[string]any{
		"status": model.DbApprovalStepApproved, "reviewer_user_id": 10, "reviewed_at": now,
	})
	if err != nil || n != 1 {
		t.Fatalf("first claim: n=%d err=%v", n, err)
	}
	// 重复审批：第二次认领必须 0 行，且不得覆盖首个审批人。
	n, err = repo.ClaimSqlTicketStep(ctx, step.ID, ticket.ID, map[string]any{
		"status": model.DbApprovalStepRejected, "reviewer_user_id": 11, "reviewed_at": now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second claim must not succeed, n=%d", n)
	}
	var got model.DbSqlTicketStep
	if err := db.First(&got, step.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.DbApprovalStepApproved || got.ReviewerUserID == nil || *got.ReviewerUserID != 10 {
		t.Fatalf("step overwritten by stale approval: %+v", got)
	}

	// 错误的工单 ID 不命中。
	n, _ = repo.ClaimSqlTicketStep(ctx, step.ID, ticket.ID+99, map[string]any{"status": model.DbApprovalStepRejected})
	if n != 0 {
		t.Fatalf("claim with wrong ticket id must be 0, got %d", n)
	}
}

func TestUpdateAccessRequestFieldsIfStatus_OnlyFromPending(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_access_if_status")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	req := model.DbAccessRequest{ProjectID: 1, InstanceID: 1, RequesterUserID: 1, Status: model.DbAccessRequestStatusPending}
	if err := db.Create(&req).Error; err != nil {
		t.Fatal(err)
	}

	n, err := repo.UpdateAccessRequestFieldsIfStatus(ctx, req.ID, model.DbAccessRequestStatusPending, map[string]any{
		"status": model.DbAccessRequestStatusApproved,
	})
	if err != nil || n != 1 {
		t.Fatalf("first approve: n=%d err=%v", n, err)
	}
	// 重复终审/并发驳回不得覆盖已通过的终态。
	n, err = repo.UpdateAccessRequestFieldsIfStatus(ctx, req.ID, model.DbAccessRequestStatusPending, map[string]any{
		"status": model.DbAccessRequestStatusRejected,
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second transition must be rejected, n=%d", n)
	}
	var got model.DbAccessRequest
	if err := db.First(&got, req.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.DbAccessRequestStatusApproved {
		t.Fatalf("final status overwritten: %s", got.Status)
	}
}

func TestClaimAccessRequestStep_OnlyOnce(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_claim_access_step")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	req := model.DbAccessRequest{ProjectID: 1, InstanceID: 1, RequesterUserID: 1, Status: model.DbAccessRequestStatusPending}
	if err := db.Create(&req).Error; err != nil {
		t.Fatal(err)
	}
	step := model.DbAccessRequestStep{AccessRequestID: req.ID, StageKey: "dba_lead", StageName: "DBA", SortOrder: 1, Status: model.DbApprovalStepPending}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	n, err := repo.ClaimAccessRequestStep(ctx, step.ID, req.ID, map[string]any{"status": model.DbApprovalStepApproved, "reviewer_user_id": 10})
	if err != nil || n != 1 {
		t.Fatalf("first claim: n=%d err=%v", n, err)
	}
	n, err = repo.ClaimAccessRequestStep(ctx, step.ID, req.ID, map[string]any{"status": model.DbApprovalStepRejected, "reviewer_user_id": 11})
	if err != nil || n != 0 {
		t.Fatalf("second claim: n=%d err=%v", n, err)
	}
}

func TestFailStaleExecutingSqlTickets_OnlyStale(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_stale_executing")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	stale := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Status: model.DbTicketStatusExecuting}
	fresh := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Status: model.DbTicketStatusExecuting}
	done := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Status: model.DbTicketStatusSuccess}
	for _, tk := range []*model.DbSqlTicket{&stale, &fresh, &done} {
		if err := db.Create(tk).Error; err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := db.Model(&model.DbSqlTicket{}).Where("id = ?", stale.ID).UpdateColumn("updated_at", old).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.DbSqlTicket{}).Where("id = ?", done.ID).UpdateColumn("updated_at", old).Error; err != nil {
		t.Fatal(err)
	}

	n, err := repo.FailStaleExecutingSqlTickets(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 stale ticket marked, got %d", n)
	}
	var gotStale, gotFresh, gotDone model.DbSqlTicket
	_ = db.First(&gotStale, stale.ID).Error
	_ = db.First(&gotFresh, fresh.ID).Error
	_ = db.First(&gotDone, done.ID).Error
	if gotStale.Status != model.DbTicketStatusResultUnknown {
		t.Fatalf("stale ticket status=%s, want result_unknown", gotStale.Status)
	}
	if gotFresh.Status != model.DbTicketStatusExecuting {
		t.Fatalf("fresh ticket must stay executing, got %s", gotFresh.Status)
	}
	if gotDone.Status != model.DbTicketStatusSuccess {
		t.Fatalf("finished ticket must stay success, got %s", gotDone.Status)
	}
}

func TestCreateGrant_SourceRequestIDUnique(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_grant_source_unique")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()
	rid := uint(7)

	first := model.DbAccessGrant{ProjectID: 1, InstanceID: 1, PrincipalKind: "user", PrincipalRef: "1", SourceRequestID: &rid}
	if err := repo.CreateGrant(ctx, &first); err != nil {
		t.Fatal(err)
	}
	dup := model.DbAccessGrant{ProjectID: 1, InstanceID: 1, PrincipalKind: "user", PrincipalRef: "1", SourceRequestID: &rid}
	if err := repo.CreateGrant(ctx, &dup); err == nil {
		t.Fatal("duplicate source_request_id must be rejected by unique index")
	}
	// 手工授权（无来源申请）允许多条 NULL。
	for i := 0; i < 2; i++ {
		g := model.DbAccessGrant{ProjectID: 1, InstanceID: 1, PrincipalKind: "user", PrincipalRef: "2"}
		if err := repo.CreateGrant(ctx, &g); err != nil {
			t.Fatalf("manual grant %d must be allowed: %v", i, err)
		}
	}
	found, err := repo.GetGrantBySourceRequest(ctx, 1, rid)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != first.ID {
		t.Fatalf("GetGrantBySourceRequest id=%d, want %d", found.ID, first.ID)
	}
}

func TestListSqlTickets_KeywordSearchesReasonAndSqlText(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_ticket_keyword")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	tickets := []model.DbSqlTicket{
		{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Reason: "修复订单状态", SqlText: "UPDATE orders SET status=1", Status: "pending_approval"},
		{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Reason: "清理缓存", SqlText: "DELETE FROM cache WHERE id=1", Status: "pending_approval"},
		{ProjectID: 2, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 1, Reason: "修复订单状态", SqlText: "UPDATE orders SET status=1", Status: "pending_approval"},
	}
	for i := range tickets {
		if err := db.Create(&tickets[i]).Error; err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "match reason", keyword: "订单", want: 1},
		{name: "match sql text", keyword: "cache", want: 1},
		{name: "match both", keyword: "status", want: 1},
		{name: "no match", keyword: "not-exists-anywhere", want: 0},
		{name: "blank ignored", keyword: "  ", want: 2},
	}
	for _, c := range cases {
		list, total, err := repo.ListSqlTickets(ctx, DbSqlTicketListParams{ProjectID: 1, Keyword: c.keyword, Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if int(total) != c.want || len(list) != c.want {
			t.Fatalf("%s: total=%d len=%d want %d", c.name, total, len(list), c.want)
		}
	}
}

func TestDeleteInstance_CascadesDependentsAndKeepsAudit(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_delete_instance")
	ctx := context.Background()
	// 级联测试需要完整模型集，主测试库只迁移常用表，这里补充其余依赖表。
	if err := db.AutoMigrate(
		&model.DbInstance{},
		&model.DbSqlExecution{},
		&model.DbAppUserRequest{},
		&model.DbAppUserRequestStep{},
		&model.DbInstanceAccount{},
		&model.DbColumnMaskRule{},
		&model.DbAuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	repo := NewDbmgmtRepository(db)

	primary := model.DbInstance{ProjectID: 1, Name: "primary", Env: model.DbEnvDev, Driver: model.DbDriverMySQL, Role: model.DbInstanceRolePrimary}
	if err := db.Create(&primary).Error; err != nil {
		t.Fatal(err)
	}
	replica := model.DbInstance{ProjectID: 1, Name: "replica", Env: model.DbEnvDev, Driver: model.DbDriverMySQL, Role: model.DbInstanceRoleReplica, PrimaryInstanceID: &primary.ID}
	if err := db.Create(&replica).Error; err != nil {
		t.Fatal(err)
	}

	ticket := model.DbSqlTicket{ProjectID: 1, InstanceID: primary.ID, TicketType: "sql_execute", SubmitterUserID: 1, Status: "pending_approval"}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	ticketStep := model.DbSqlTicketStep{TicketID: ticket.ID, StageKey: "dba", StageName: "DBA", SortOrder: 1, Status: model.DbApprovalStepPending}
	access := model.DbAccessRequest{ProjectID: 1, InstanceID: primary.ID, RequesterUserID: 1, Status: "pending"}
	if err := db.Create(&access).Error; err != nil {
		t.Fatal(err)
	}
	accessStep := model.DbAccessRequestStep{AccessRequestID: access.ID, StageKey: "dba", StageName: "DBA", SortOrder: 1, Status: model.DbApprovalStepPending}
	appUser := model.DbAppUserRequest{ProjectID: 1, InstanceID: primary.ID, RequesterUserID: 1, ApplyType: model.DbAppUserApplyNewUser, MySQLUser: "u", MySQLHost: "%", Status: "pending"}
	if err := db.Create(&appUser).Error; err != nil {
		t.Fatal(err)
	}
	appUserStep := model.DbAppUserRequestStep{AppUserRequestID: appUser.ID, StageKey: "dba", StageName: "DBA", SortOrder: 1, Status: model.DbApprovalStepPending}
	grant := model.DbAccessGrant{ProjectID: 1, InstanceID: primary.ID, PrincipalKind: "user", PrincipalRef: "1"}
	account := model.DbInstanceAccount{ProjectID: 1, InstanceID: primary.ID, Username: "u", Host: "%"}
	mask := model.DbColumnMaskRule{InstanceID: primary.ID, MaskTable: "t", ColumnName: "c", MaskType: "redact"}
	exec := model.DbSqlExecution{ProjectID: 1, InstanceID: primary.ID, ExecutorUserID: 1, SqlExcerpt: "SELECT 1"}
	audit := model.DbAuditLog{ProjectID: 1, InstanceID: &primary.ID, ActorUserID: 1, Action: "instance_upsert"}
	for _, row := range []any{&ticketStep, &accessStep, &appUserStep, &grant, &account, &mask, &exec, &audit} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}

	if err := repo.DeleteInstance(ctx, primary.ID); err != nil {
		t.Fatal(err)
	}

	assertZero := func(name string, m any, where string, args ...any) {
		t.Helper()
		var n int64
		if err := db.Model(m).Where(where, args...).Count(&n).Error; err != nil {
			t.Fatalf("%s count: %v", name, err)
		}
		if n != 0 {
			t.Fatalf("%s left %d orphan rows", name, n)
		}
	}
	assertZero("tickets", &model.DbSqlTicket{}, "instance_id = ?", primary.ID)
	assertZero("ticket_steps", &model.DbSqlTicketStep{}, "ticket_id = ?", ticket.ID)
	assertZero("access_requests", &model.DbAccessRequest{}, "instance_id = ?", primary.ID)
	assertZero("access_steps", &model.DbAccessRequestStep{}, "access_request_id = ?", access.ID)
	assertZero("app_user_requests", &model.DbAppUserRequest{}, "instance_id = ?", primary.ID)
	assertZero("app_user_steps", &model.DbAppUserRequestStep{}, "app_user_request_id = ?", appUser.ID)
	assertZero("grants", &model.DbAccessGrant{}, "instance_id = ?", primary.ID)
	assertZero("accounts", &model.DbInstanceAccount{}, "instance_id = ?", primary.ID)
	assertZero("mask_rules", &model.DbColumnMaskRule{}, "instance_id = ?", primary.ID)
	assertZero("executions", &model.DbSqlExecution{}, "instance_id = ?", primary.ID)

	var gotReplica model.DbInstance
	if err := db.First(&gotReplica, replica.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotReplica.PrimaryInstanceID != nil {
		t.Fatalf("replica primary_instance_id must be cleared, got %v", *gotReplica.PrimaryInstanceID)
	}
	var auditCount int64
	if err := db.Model(&model.DbAuditLog{}).Where("instance_id = ?", primary.ID).Count(&auditCount).Error; err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("audit logs must be preserved, got %d", auditCount)
	}
}

func TestListSqlTicketsMine_DoneIncludesResultUnknown(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_mine_result_unknown")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	unknown := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 5, Reason: "超时", SqlText: "UPDATE t SET a=1", Status: model.DbTicketStatusResultUnknown}
	pendingExecution := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 5, Reason: "待执行", SqlText: "UPDATE t SET a=2", Status: model.DbTicketStatusPendingExecution}
	for _, tk := range []*model.DbSqlTicket{&unknown, &pendingExecution} {
		if err := db.Create(tk).Error; err != nil {
			t.Fatal(err)
		}
	}

	list, total, err := repo.ListSqlTicketsMine(ctx, DbTicketMineListParams{
		DbMineListParams: DbMineListParams{ProjectID: 1, UserID: 5, Scope: "done", Page: 1, PageSize: 20},
		MineTab:          "execution",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Status != model.DbTicketStatusResultUnknown {
		t.Fatalf("result_unknown must be listed as done: total=%d list=%v", total, list)
	}

	// 未指定 scope 时，待执行 + 结果未知都应返回。
	list, total, err = repo.ListSqlTicketsMine(ctx, DbTicketMineListParams{
		DbMineListParams: DbMineListParams{ProjectID: 1, UserID: 5, Page: 1, PageSize: 20},
		MineTab:          "execution",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Fatalf("execution all scope: total=%d len=%d want 2", total, len(list))
	}
}

func TestListSqlTicketsMine_KeywordFilter(t *testing.T) {
	t.Parallel()
	db := newDbmgmtTestDB(t, "dbmgmt_mine_keyword")
	repo := NewDbmgmtRepository(db)
	ctx := context.Background()

	rows := []model.DbSqlTicket{
		{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 5, Reason: "上线", SqlText: "ALTER TABLE users ADD COLUMN age INT", Status: model.DbTicketStatusSuccess},
		{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 5, Reason: "上线", SqlText: "ALTER TABLE orders ADD COLUMN amount INT", Status: model.DbTicketStatusSuccess},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	// 别人提交的同关键词工单不应出现在“我的”列表。
	other := model.DbSqlTicket{ProjectID: 1, InstanceID: 1, TicketType: "sql_execute", SubmitterUserID: 6, Reason: "上线", SqlText: "ALTER TABLE users ADD COLUMN age INT", Status: model.DbTicketStatusSuccess}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	list, total, err := repo.ListSqlTicketsMine(ctx, DbTicketMineListParams{
		DbMineListParams: DbMineListParams{ProjectID: 1, UserID: 5, Scope: "done", Page: 1, PageSize: 20},
		MineTab:          "execution",
		Keyword:          "users",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != rows[0].ID {
		t.Fatalf("keyword on mine list: total=%d list=%v", total, list)
	}
}
