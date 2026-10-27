package workflow

import (
	"context"
	"testing"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestForbidSelfApprove(t *testing.T) {
	t.Parallel()
	submitter := uint(7)
	cases := []struct {
		name    string
		actor   *auth.CurrentUser
		wantErr bool
	}{
		{"submitter blocked", &auth.CurrentUser{ID: 7, RoleCodes: []string{"dev"}}, true},
		{"other user allowed", &auth.CurrentUser{ID: 8, RoleCodes: []string{"dev"}}, false},
		{"super admin allowed", &auth.CurrentUser{ID: 7, RoleCodes: []string{"super-admin"}}, false},
		{"nil actor allowed", nil, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := forbidSelfApprove(tc.actor, submitter)
			if tc.wantErr && err == nil {
				t.Fatal("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("want nil, got %v", err)
			}
		})
	}
}

func TestWorkflowRefTypeUQ(t *testing.T) {
	t.Parallel()
	if p := workflowRefTypeUQ("", 1); p != nil {
		t.Fatalf("empty ref should be nil, got %v", *p)
	}
	if p := workflowRefTypeUQ("  ", 1); p != nil {
		t.Fatalf("blank ref should be nil, got %v", *p)
	}
	if p := workflowRefTypeUQ("alert_event", 0); p != nil {
		t.Fatalf("zero ref_id should be nil, got %v", *p)
	}
	p := workflowRefTypeUQ(" alert_event ", 5)
	if p == nil || *p != "alert_event" {
		t.Fatalf("valid ref should be trimmed pointer, got %v", p)
	}
}

func newServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkflowDefinition{}, &model.WorkflowStage{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpsertDefinitionRestoresSoftDeletedStage(t *testing.T) {
	t.Parallel()
	db := newServiceTestDB(t, "wf_upsert_restore")
	repo := repository.NewWorkflowRepository(db)
	svc := NewService(repo, nil, nil, nil, nil)
	ctx := context.Background()
	key := DefinitionKey{Domain: model.WorkflowDomainDbmgmt, ProjectID: 1}

	req := DefinitionUpsertRequest{Stages: []StageUpsertItem{{
		StageKey: "dba_lead", StageName: "DBA 负责人", SortOrder: 1,
		Enabled: true, AssigneeRuleType: model.WorkflowAssigneeUserGroup,
	}}}
	if _, err := svc.UpsertDefinition(ctx, key, req); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	// 软删该节点后再次 upsert：应复用旧行恢复而非插入（否则撞唯一键）
	var stage model.WorkflowStage
	if err := db.Where("stage_key = ?", "dba_lead").First(&stage).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&stage).Error; err != nil {
		t.Fatal(err)
	}
	req.Stages[0].StageName = "DBA 主管"
	if _, err := svc.UpsertDefinition(ctx, key, req); err != nil {
		t.Fatalf("second upsert should restore soft-deleted stage: %v", err)
	}
	var count int64
	if err := db.Model(&model.WorkflowStage{}).Where("stage_key = ?", "dba_lead").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("want single active stage, got %d", count)
	}
	var got model.WorkflowStage
	if err := db.Where("stage_key = ?", "dba_lead").First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.ID != stage.ID {
		t.Fatalf("restore should reuse row %d, got %d", stage.ID, got.ID)
	}
	if got.StageName != "DBA 主管" {
		t.Fatalf("restored stage name not updated: %+v", got)
	}
}
