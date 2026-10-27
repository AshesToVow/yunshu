package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newWorkflowTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkflowTicket{}, &model.WorkflowTicketStep{}); err != nil {
		t.Fatal(err)
	}
	return db
}

// newWorkflowPendingTestDB ListPending 的 SQL 引用多张关联表（EXISTS 子查询），
// 这些表在 sqlite 里只需存在即可（子查询仅判存在性），故建最小结构。
func newWorkflowPendingTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db := newWorkflowTestDB(t, name)
	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS user_group_users (user_group_id INTEGER, user_id INTEGER)`,
		`CREATE TABLE IF NOT EXISTS cicd_release_runs (id INTEGER PRIMARY KEY, status TEXT, deleted_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS db_sql_tickets (id INTEGER PRIMARY KEY, status TEXT)`,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func newWorkflowTicketWithStep(t *testing.T, db *gorm.DB, ticket model.WorkflowTicket, step model.WorkflowTicketStep) (uint, uint) {
	t.Helper()
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	step.TicketID = ticket.ID
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	return ticket.ID, step.ID
}

func TestWorkflowClaimStepReview_OnlyFromPending(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_claim_seq")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	now := time.Now()
	ticketID, stepID := newWorkflowTicketWithStep(t, db,
		model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 1, Title: "t", Status: "pending", SubmitterUserID: 1},
		model.WorkflowTicketStep{StageKey: "rd_lead", StageName: "研发负责人", SortOrder: 1, Status: "pending", ActivatedAt: &now})

	n, err := repo.ClaimStepReview(ctx, stepID, ticketID, map[string]any{"status": "approved", "reviewer_user_id": 10, "reviewed_at": now})
	if err != nil || n != 1 {
		t.Fatalf("first claim: n=%d err=%v", n, err)
	}
	// 状态已非 pending，二次抢占必须失败且不得覆盖
	n, err = repo.ClaimStepReview(ctx, stepID, ticketID, map[string]any{"status": "rejected", "reviewer_user_id": 11, "reviewed_at": now})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second claim should affect 0 rows, got %d", n)
	}
	var got model.WorkflowTicketStep
	if err := db.First(&got, stepID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "approved" || got.ReviewerUserID == nil || *got.ReviewerUserID != 10 {
		t.Fatalf("step overwritten: status=%s reviewer=%v", got.Status, got.ReviewerUserID)
	}
}

func TestWorkflowClaimStepReview_ConcurrentSingleWinner(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_claim_conc")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	now := time.Now()
	ticketID, stepID := newWorkflowTicketWithStep(t, db,
		model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 1, Title: "t", Status: "pending", SubmitterUserID: 1},
		model.WorkflowTicketStep{StageKey: "rd_lead", StageName: "研发负责人", SortOrder: 1, Status: "pending", ActivatedAt: &now})

	const workers = 8
	var mu sync.Mutex
	winners := 0
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			n, err := repo.ClaimStepReview(ctx, stepID, ticketID, map[string]any{
				"status": "approved", "reviewer_user_id": id, "reviewed_at": time.Now(),
			})
			if err != nil {
				t.Errorf("claim err: %v", err)
				return
			}
			if n == 1 {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		}(uint(100 + i))
	}
	wg.Wait()
	if winners != 1 {
		t.Fatalf("want exactly 1 winner, got %d", winners)
	}
	var got model.WorkflowTicketStep
	if err := db.First(&got, stepID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "approved" || got.ReviewerUserID == nil {
		t.Fatalf("unexpected final step state: %+v", got)
	}
}

func TestWorkflowUpdateTicketFieldsIfStatus_Conditional(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_ticket_ifstatus")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	ticket := model.WorkflowTicket{Domain: "dbmgmt", TicketType: "sql_ticket", ProjectID: 1, Title: "t", Status: "pending", SubmitterUserID: 1}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	n, err := repo.UpdateTicketFieldsIfStatus(ctx, ticket.ID, model.WorkflowTicketStatusPending,
		map[string]any{"status": model.WorkflowTicketStatusApproved})
	if err != nil || n != 1 {
		t.Fatalf("first update: n=%d err=%v", n, err)
	}
	// 前置状态已不满足，第二次更新应影响 0 行
	n, err = repo.UpdateTicketFieldsIfStatus(ctx, ticket.ID, model.WorkflowTicketStatusPending,
		map[string]any{"status": model.WorkflowTicketStatusRejected})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second update should affect 0 rows, got %d", n)
	}
	var got model.WorkflowTicket
	if err := db.First(&got, ticket.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != model.WorkflowTicketStatusApproved {
		t.Fatalf("status overwritten to %q", got.Status)
	}
}

func TestWorkflowActivateStepIfTicketPending_ClosedTicket(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_activate_closed")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	now := time.Now()
	ticket := model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 1, Title: "t", Status: "closed", SubmitterUserID: 1}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	step := model.WorkflowTicketStep{TicketID: ticket.ID, StageKey: "ops", StageName: "运维", SortOrder: 2, Status: "pending"}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}
	n, err := repo.ActivateStepIfTicketPending(ctx, step.ID, ticket.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("closed ticket should not activate step, got %d", n)
	}
}

func TestWorkflowSkipPendingSteps(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_skip_steps")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	now := time.Now()
	ticket := model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 1, Title: "t", Status: "closed", SubmitterUserID: 1}
	if err := db.Create(&ticket).Error; err != nil {
		t.Fatal(err)
	}
	for i, st := range []string{"pending", "approved", "pending"} {
		step := model.WorkflowTicketStep{TicketID: ticket.ID, StageKey: fmt.Sprintf("s%d", i), StageName: st, SortOrder: i + 1, Status: st}
		if i == 0 {
			step.ActivatedAt = &now
		}
		if err := db.Create(&step).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.SkipPendingSteps(ctx, ticket.ID, now); err != nil {
		t.Fatal(err)
	}
	var steps []model.WorkflowTicketStep
	if err := db.Where("ticket_id = ?", ticket.ID).Order("sort_order ASC").Find(&steps).Error; err != nil {
		t.Fatal(err)
	}
	if steps[0].Status != "skipped" || steps[0].ReviewedAt == nil {
		t.Fatalf("pending step not skipped: %+v", steps[0])
	}
	if steps[1].Status != "approved" {
		t.Fatalf("approved step modified: %+v", steps[1])
	}
	if steps[2].Status != "skipped" {
		t.Fatalf("second pending step not skipped: %+v", steps[2])
	}
}

func TestWorkflowListPending_UserIDZeroReturnsEmpty(t *testing.T) {
	t.Parallel()
	repo := NewWorkflowRepository(nil)
	rows, total, err := repo.ListPending(context.Background(), WorkflowPendingFilter{
		UserID: 0, IsSuper: false, MineScope: "all",
	})
	if err != nil {
		t.Fatalf("want no error for zero user, got %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("want empty result, got total=%d rows=%d", total, len(rows))
	}
}

func TestWorkflowListPending_DutyVisibility(t *testing.T) {
	t.Parallel()
	db := newWorkflowPendingTestDB(t, "wf_pending_duty")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	now := time.Now()
	ticketID, stepID := newWorkflowTicketWithStep(t, db,
		model.WorkflowTicket{Domain: "incident", TicketType: "incident", ProjectID: 1, Title: "t", Status: "pending", SubmitterUserID: 1},
		model.WorkflowTicketStep{StageKey: "oncall_duty", StageName: "当班处理人", SortOrder: 1, Status: "pending", AssigneeRuleType: model.WorkflowAssigneeDuty, ActivatedAt: &now})

	// 平台审批人可看到未解析值班人的 duty 节点
	rows, total, err := repo.ListPending(ctx, WorkflowPendingFilter{
		UserID: 99, IsSuper: false, CanPlatformReview: true, MineScope: "pending",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(rows) != 1 || rows[0].StepID != stepID {
		t.Fatalf("duty-unresolved step should be visible to platform reviewer: total=%d rows=%+v", total, rows)
	}
	// 无平台审批权限时不可见
	rows, total, err = repo.ListPending(ctx, WorkflowPendingFilter{
		UserID: 99, IsSuper: false, CanPlatformReview: false, MineScope: "pending",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("duty-unresolved step should be hidden without platform review: total=%d", total)
	}
	// 值班人已解析为某用户时，仅该用户可见
	if err := db.Model(&model.WorkflowTicketStep{}).Where("id = ?", stepID).
		Update("assignee_user_id", 77).Error; err != nil {
		t.Fatal(err)
	}
	rows, total, err = repo.ListPending(ctx, WorkflowPendingFilter{
		UserID: 77, IsSuper: false, CanPlatformReview: false, MineScope: "pending",
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(rows) != 1 || rows[0].StepID != stepID {
		t.Fatalf("resolved duty step should be visible to assignee: total=%d", total)
	}
	_ = ticketID
}

func TestWorkflowListTickets_Visibility(t *testing.T) {
	t.Parallel()
	db := newWorkflowTestDB(t, "wf_list_visibility")
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	mine := model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 1, Title: "mine", Status: "pending", SubmitterUserID: 1}
	other := model.WorkflowTicket{Domain: "cicd", TicketType: "release", ProjectID: 2, Title: "other", Status: "pending", SubmitterUserID: 2}
	if err := db.Create(&mine).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	// user 1 是 other 工单某步骤的指派人
	step := model.WorkflowTicketStep{TicketID: other.ID, StageKey: "rd_lead", StageName: "研发负责人", SortOrder: 1, Status: "pending", AssigneeUserID: uintPtr(1)}
	if err := db.Create(&step).Error; err != nil {
		t.Fatal(err)
	}

	uid := uint(1)
	rows, total, err := repo.ListTickets(ctx, WorkflowTicketListParams{VisibleUserID: &uid, Offset: 0, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("submitter+participant should see 2 tickets, got %d", total)
	}

	// 无关用户：仅项目成员可见
	uid3 := uint(3)
	rows, total, err = repo.ListTickets(ctx, WorkflowTicketListParams{VisibleUserID: &uid3, Offset: 0, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(rows) != 0 {
		t.Fatalf("outsider should see nothing, got total=%d", total)
	}
	rows, total, err = repo.ListTickets(ctx, WorkflowTicketListParams{
		VisibleUserID: &uid3, MemberProjectIDs: []uint{2}, Offset: 0, Limit: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != other.ID {
		t.Fatalf("project member should see only project ticket, got %+v", rows)
	}
}

func TestWorkflowStageSoftDeleteRestore(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:wf_stage_restore?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkflowDefinition{}, &model.WorkflowStage{}); err != nil {
		t.Fatal(err)
	}
	repo := NewWorkflowRepository(db)
	ctx := context.Background()
	def := model.WorkflowDefinition{Domain: "dbmgmt", ProjectID: 1, TicketType: "default", Name: "flow", Enabled: true}
	if err := db.Create(&def).Error; err != nil {
		t.Fatal(err)
	}
	stage := model.WorkflowStage{DefinitionID: def.ID, StageKey: "dba_lead", StageName: "DBA", SortOrder: 1, Enabled: true, AssigneeRuleType: "user_group"}
	if err := db.Create(&stage).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&stage).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetStageByKey(ctx, def.ID, "dba_lead"); err == nil {
		t.Fatal("soft-deleted stage should not be visible to GetStageByKey")
	}
	soft, err := repo.GetStageByKeyUnscoped(ctx, def.ID, "dba_lead")
	if err != nil || soft == nil || !soft.DeletedAt.Valid {
		t.Fatalf("GetStageByKeyUnscoped should find soft-deleted stage: %v %+v", err, soft)
	}
	if err := repo.RestoreStage(ctx, soft.ID, map[string]any{
		"stage_name": "DBA 负责人", "sort_order": 2, "enabled": true,
		"assignee_rule_type": "user_group", "deleted_at": nil,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetStageByKey(ctx, def.ID, "dba_lead")
	if err != nil || got == nil {
		t.Fatalf("stage should be restored: %v", err)
	}
	if got.StageName != "DBA 负责人" || got.SortOrder != 2 {
		t.Fatalf("restored fields mismatch: %+v", got)
	}
}
