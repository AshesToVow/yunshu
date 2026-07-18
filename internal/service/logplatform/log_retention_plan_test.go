package logplatform

import (
	"reflect"
	"testing"
	"time"

	"yunshu/internal/pkg/esclient"
)

func idx(name string, bytes int64) esclient.IndexInfo {
	return esclient.IndexInfo{Name: name, StoreBytes: bytes}
}

// 过期删除：早于 cutoff 的索引被删，之后的保留。
func TestPlanIndexDeletions_RetentionDays(t *testing.T) {
	cutoff := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	indices := []esclient.IndexInfo{
		idx("yunshu-agent-7-2026.07.10", 100),
		idx("yunshu-agent-7-2026.07.11", 100),
		idx("yunshu-agent-7-2026.07.13", 100),
	}
	got, dated := planIndexDeletions(indices, cutoff, 0, 0)
	if dated != 3 {
		t.Fatalf("dated=%d want 3", dated)
	}
	want := []string{"yunshu-agent-7-2026.07.10", "yunshu-agent-7-2026.07.11"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// 超过最大索引数：必须删除最旧的多余索引，而不是 _cat 返回顺序里的头几个。
// 这里刻意把最新的索引排在切片最前，验证不会误删新日志。
func TestPlanIndexDeletions_MaxIndexCountEvictsOldest(t *testing.T) {
	cutoff := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // 都不过期
	indices := []esclient.IndexInfo{
		idx("yunshu-agent-7-2026.07.15", 100), // 最新，排在最前
		idx("yunshu-agent-7-2026.07.10", 100), // 最旧
		idx("yunshu-agent-7-2026.07.12", 100),
	}
	got, _ := planIndexDeletions(indices, cutoff, 2, 0)
	want := []string{"yunshu-agent-7-2026.07.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v (应删最旧而非 _cat 首个)", got, want)
	}
}

// 超过最大存储：从最旧开始删除直到低于阈值。
func TestPlanIndexDeletions_MaxStoreBytesEvictsOldestFirst(t *testing.T) {
	cutoff := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	indices := []esclient.IndexInfo{
		idx("yunshu-agent-7-2026.07.14", 50),
		idx("yunshu-agent-7-2026.07.10", 50),
		idx("yunshu-agent-7-2026.07.12", 50),
	}
	// 总 150B，阈值 90B → 需删到 <=90，从最旧删两个（07.10, 07.12）留 07.14=50。
	got, _ := planIndexDeletions(indices, cutoff, 0, 90)
	want := []string{"yunshu-agent-7-2026.07.10", "yunshu-agent-7-2026.07.12"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// 各规则叠加不重复计数：天数删除后的索引不应在超量/超容阶段被重复删除或重复计入总量。
func TestPlanIndexDeletions_NoDoubleCount(t *testing.T) {
	cutoff := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	indices := []esclient.IndexInfo{
		idx("yunshu-agent-7-2026.07.09", 100), // 过期
		idx("yunshu-agent-7-2026.07.10", 100), // 过期
		idx("yunshu-agent-7-2026.07.13", 100),
		idx("yunshu-agent-7-2026.07.14", 100),
	}
	// 天数删 2 个后剩 2 个，maxIndexCount=2 不应再删；结果只含两个过期索引且不重复。
	got, _ := planIndexDeletions(indices, cutoff, 2, 0)
	want := []string{"yunshu-agent-7-2026.07.09", "yunshu-agent-7-2026.07.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// 无日期后缀的索引不参与超量/超容淘汰（无法判断新旧，避免误删）。
func TestPlanIndexDeletions_SkipsUndatedForEviction(t *testing.T) {
	cutoff := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	indices := []esclient.IndexInfo{
		idx("app-logs-static", 100),
		idx("yunshu-agent-7-2026.07.10", 100),
		idx("yunshu-agent-7-2026.07.12", 100),
	}
	got, dated := planIndexDeletions(indices, cutoff, 1, 0)
	if dated != 2 {
		t.Fatalf("dated=%d want 2", dated)
	}
	// 2 个带日期索引，max=1 → 删最旧 1 个；无日期索引不动。
	want := []string{"yunshu-agent-7-2026.07.10"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
