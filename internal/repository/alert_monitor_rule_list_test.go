package repository

import (
	"context"
	"testing"

	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAlertMonitorRuleListIncludesLogRulesByProject(t *testing.T) {
	t.Parallel()
	db, err := gorm.Open(sqlite.Open("file:alert_rule_list?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// gorm.io/driver/sqlite 依赖 CGO；Windows/CI 常以 CGO_ENABLED=0 编译。
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertDatasource{}, &model.AlertMonitorRule{}); err != nil {
		t.Fatal(err)
	}
	ds := model.AlertDatasource{ProjectID: 7, Name: "prom", Type: "prometheus", BaseURL: "http://x", Enabled: true}
	if err := db.Create(&ds).Error; err != nil {
		t.Fatal(err)
	}
	promRule := model.AlertMonitorRule{
		DatasourceID: ds.ID, Name: "cpu", RuleKind: model.AlertRuleKindPromQL,
		Expr: "1", Enabled: true, Severity: "warning",
	}
	logRule := model.AlertMonitorRule{
		DatasourceID: 0, ProjectID: 7, Name: "log-err", RuleKind: model.AlertRuleKindLog,
		Expr: `{"mode":"error_count","threshold":10}`, Enabled: true, Severity: "critical",
	}
	otherLog := model.AlertMonitorRule{
		DatasourceID: 0, ProjectID: 99, Name: "other", RuleKind: model.AlertRuleKindLog,
		Expr: `{"mode":"error_count","threshold":1}`, Enabled: true, Severity: "warning",
	}
	for _, r := range []*model.AlertMonitorRule{&promRule, &logRule, &otherLog} {
		if err := db.Create(r).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewAlertMonitorRuleRepository(db)
	pid := uint(7)
	list, total, err := repo.List(context.Background(), AlertMonitorRuleListFilter{ProjectID: &pid}, 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("want 2 rules for project 7, got total=%d list=%d", total, len(list))
	}
	var sawLog, sawProm bool
	for _, r := range list {
		if r.RuleKind == model.AlertRuleKindLog {
			sawLog = true
		}
		if r.RuleKind == model.AlertRuleKindPromQL {
			sawProm = true
		}
	}
	if !sawLog || !sawProm {
		t.Fatalf("expected both log and promql rules, got %+v", list)
	}
}
