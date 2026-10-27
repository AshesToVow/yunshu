package dictconfig

import (
	"context"
	"fmt"
	"testing"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openDictFetchTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:dict_fetch_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	if err := db.AutoMigrate(&model.DictEntry{}); err != nil {
		t.Skipf("sqlite migrate failed: %v", err)
	}
	return db
}

func seedDict(t *testing.T, db *gorm.DB, dictType, value string, sort int, status int) *model.DictEntry {
	t.Helper()
	row := &model.DictEntry{DictType: dictType, Label: dictType, Value: value, Sort: sort, Status: status}
	if err := db.Create(row).Error; err != nil {
		t.Fatalf("seed %s: %v", dictType, err)
	}
	return row
}

func TestNormalizeDictTypes(t *testing.T) {
	got := normalizeDictTypes([]string{" a ", "b", "a", "", "  "})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("应去空白去重并保序，实际 %v", got)
	}
}

func TestPickFirstValuePerType(t *testing.T) {
	rows := []model.DictEntry{
		{DictType: "batch_sort", Value: "first"},
		{DictType: "batch_sort", Value: "second"},
		{DictType: "BATCH_CASE", Value: " upper "},
		{DictType: "batch_empty", Value: "  "},
		{DictType: "batch_extra", Value: "not-requested"},
	}
	got := pickFirstValuePerType(
		[]string{"batch_sort", "batch_case", "batch_empty", "batch_missing"},
		rows,
	)
	if v, ok := got["batch_sort"]; !ok || v != "first" {
		t.Fatalf("每类型应取首条，实际 %q ok=%v", v, ok)
	}
	if v, ok := got["batch_case"]; !ok || v != "upper" {
		t.Fatalf("大小写差异应按请求键名回填并去空白，实际 %q ok=%v", v, ok)
	}
	if v, ok := got["batch_empty"]; !ok || v != "" {
		t.Fatalf("显式空值应保留，实际 %q ok=%v", v, ok)
	}
	if _, ok := got["batch_missing"]; ok {
		t.Fatalf("未配置类型不应出现: %v", got)
	}
	if _, ok := got["batch_extra"]; ok {
		t.Fatalf("未请求类型不应出现: %v", got)
	}
	if len(got) != 3 {
		t.Fatalf("应命中 3 个类型，实际 %d: %v", len(got), got)
	}
}

func TestFetchEnabledDictValuesRules(t *testing.T) {
	ctx := context.Background()

	if got := FetchEnabledDictValues(ctx, nil, []string{"a"}); len(got) != 0 {
		t.Fatalf("nil db 应返回空结果，实际 %v", got)
	}
	if got := FetchEnabledDictValues(ctx, nil, nil); got == nil || len(got) != 0 {
		t.Fatalf("nil 类型列表应返回空 map，实际 %v", got)
	}

	db := openDictFetchTestDB(t)

	// sort 小者优先；同 sort 取 id 大者。
	seedDict(t, db, "batch_sort", "later", 2, 1)
	seedDict(t, db, "batch_sort", "first", 1, 1)
	seedDict(t, db, "batch_tie", "old", 1, 1)
	seedDict(t, db, "batch_tie", "new", 1, 1)
	// 停用项不生效。
	seedDict(t, db, "batch_disabled", "off", 1, 0)
	// 显式空值应保留为「已配置但为空」。
	seedDict(t, db, "batch_empty", "  ", 1, 1)
	// 软删除项不生效。
	deleted := seedDict(t, db, "batch_deleted", "gone", 1, 1)
	if err := db.Delete(deleted).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	got := FetchEnabledDictValues(ctx, db, []string{
		"batch_sort", "batch_tie", "batch_disabled", "batch_empty", "batch_deleted", "batch_missing", "batch_sort", " ",
	})
	if v, ok := got["batch_sort"]; !ok || v != "first" {
		t.Fatalf("batch_sort 应取 sort 最小项 first，实际 %q ok=%v", v, ok)
	}
	if v, ok := got["batch_tie"]; !ok || v != "new" {
		t.Fatalf("batch_tie 同 sort 应取 id 最大项 new，实际 %q ok=%v", v, ok)
	}
	if _, ok := got["batch_disabled"]; ok {
		t.Fatalf("停用项不应出现在结果中: %v", got)
	}
	if v, ok := got["batch_empty"]; !ok || v != "" {
		t.Fatalf("显式空值应保留，实际 %q ok=%v", v, ok)
	}
	if _, ok := got["batch_deleted"]; ok {
		t.Fatalf("软删除项不应出现在结果中: %v", got)
	}
	if _, ok := got["batch_missing"]; ok {
		t.Fatalf("未配置项不应出现: %v", got)
	}
	if _, ok := got[""]; ok {
		t.Fatalf("空 dict_type 不应出现: %v", got)
	}
	if len(got) != 3 {
		t.Fatalf("应命中 3 个类型，实际 %d: %v", len(got), got)
	}

	snap := fetchDictSnapshot(ctx, db, "batch_empty", "batch_missing")
	if _, ok := snap.getNonEmpty("batch_empty"); ok {
		t.Fatal("getNonEmpty 不应把显式空值视为命中")
	}
	if _, ok := snap.get("batch_empty"); !ok {
		t.Fatal("get 应把显式空值视为命中")
	}
	if _, ok := snap.getNonEmpty("batch_missing"); ok {
		t.Fatal("getNonEmpty 不应命中缺失类型")
	}
}

// countDictQueries 统计 gorm 查询次数，用于校验 resolver 已收敛为单次批量查询。
func countDictQueries(t *testing.T, db *gorm.DB) *int {
	t.Helper()
	count := 0
	name := fmt.Sprintf("test:count_dict_queries_%d", time.Now().UnixNano())
	if err := db.Callback().Query().After("gorm:query").Register(name, func(*gorm.DB) { count++ }); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	return &count
}

func TestResolversUseSingleBatchQuery(t *testing.T) {
	ctx := context.Background()
	db := openDictFetchTestDB(t)

	cases := []struct {
		name    string
		resolve func()
	}{
		{"MinIO", func() { ResolveMinioConfig(ctx, db, DefaultMinioDictTypes()) }},
		{"Elasticsearch", func() {
			ResolveElasticsearchConfig(ctx, db, config.ElasticsearchConfig{})
		}},
		{"Kafka", func() { ResolveKafkaConfig(ctx, db, config.KafkaConfig{}) }},
	}
	for _, tc := range cases {
		count := countDictQueries(t, db)
		*count = 0
		tc.resolve()
		if *count != 1 {
			t.Fatalf("%s resolver 应只查 1 次字典，实际 %d", tc.name, *count)
		}
	}
}
