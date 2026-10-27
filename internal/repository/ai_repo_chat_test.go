package repository

import (
	"context"
	"sync"
	"testing"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newApprovalTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.AiToolApproval{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestClaimApprovalReview_OnlyFromPending(t *testing.T) {
	db := newApprovalTestDB(t, "ai_claim_review_seq")
	repo := NewAiRepository(db)
	ctx := context.Background()
	row := model.AiToolApproval{UserID: 1, ToolName: "restart_deployment", Status: "pending"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	n, err := repo.ClaimApprovalReview(ctx, row.ID, map[string]any{"status": "approved", "reviewer_id": 10})
	if err != nil || n != 1 {
		t.Fatalf("first claim: n=%d err=%v", n, err)
	}
	// 状态已非 pending，二次抢占必须失败且不得覆盖
	n, err = repo.ClaimApprovalReview(ctx, row.ID, map[string]any{"status": "rejected", "reviewer_id": 11})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("second claim should affect 0 rows, got %d", n)
	}
	var got model.AiToolApproval
	if err := db.First(&got, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "approved" {
		t.Fatalf("status overwritten to %q", got.Status)
	}
}

func TestClaimApprovalReview_ConcurrentSingleWinner(t *testing.T) {
	db := newApprovalTestDB(t, "ai_claim_review_conc")
	repo := NewAiRepository(db)
	ctx := context.Background()
	row := model.AiToolApproval{UserID: 1, ToolName: "scale_deployment", Status: "pending"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	const workers = 8
	var mu sync.Mutex
	var wins int64
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status := "approved"
			if i%2 == 1 {
				status = "rejected"
			}
			n, err := repo.ClaimApprovalReview(ctx, row.ID, map[string]any{"status": status, "reviewer_id": uint(i + 10)})
			if err != nil {
				t.Errorf("claim err: %v", err)
				return
			}
			mu.Lock()
			wins += n
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("exactly one concurrent claim should win, got %d", wins)
	}
}

func TestClaimApprovalExecution_NotFromExecuting(t *testing.T) {
	db := newApprovalTestDB(t, "ai_claim_exec")
	repo := NewAiRepository(db)
	ctx := context.Background()
	row := model.AiToolApproval{UserID: 1, ToolName: "delete_pod", Status: "approved"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	n, err := repo.ClaimApprovalExecution(ctx, row.ID, []string{"approved", "failed"})
	if err != nil || n != 1 {
		t.Fatalf("first claim: n=%d err=%v", n, err)
	}
	// executing 状态下不可再抢占（修复：from 状态含 executing 时高危操作会被执行两次）
	n, err = repo.ClaimApprovalExecution(ctx, row.ID, []string{"approved", "failed"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("executing row must not be claimable, affected %d", n)
	}
	// failed 允许重试
	if err := db.Model(&row).Updates(map[string]any{"status": "failed"}).Error; err != nil {
		t.Fatal(err)
	}
	n, err = repo.ClaimApprovalExecution(ctx, row.ID, []string{"approved", "failed"})
	if err != nil || n != 1 {
		t.Fatalf("retry from failed: n=%d err=%v", n, err)
	}
}
