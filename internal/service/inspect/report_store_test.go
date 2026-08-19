package inspect

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRenderBuiltinTemplates(t *testing.T) {
	t.Parallel()
	data := ReportData{
		Timestamp: time.Now(),
		Project:   "demo",
		Score:     90,
		Grade:     "A",
		Summary:   "ok",
		Groups: []ReportGroup{{
			Type:    "host",
			Stats:   GroupStats{Total: 1, Normal: 1},
			Metrics: []MetricSample{{Name: "cpu", Instance: "a", Value: 1, Status: "normal"}},
		}},
	}
	for _, code := range []string{"default", "compact", "executive"} {
		b, err := renderHTMLWithTemplate(code, "", data)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		html := string(b)
		if !strings.Contains(html, "demo") {
			t.Fatalf("%s missing project name", code)
		}
		if !strings.Contains(html, "grade-a") {
			t.Fatalf("%s missing grade class", code)
		}
	}
}

func TestRenderExcelAndPDF(t *testing.T) {
	t.Parallel()
	data := ReportData{
		Timestamp:      time.Now(),
		Project:        "测试功能项目",
		Datasource:     "测试环境",
		InspectionUser: "System Admin",
		Score:          14,
		Grade:          "D",
		Summary:        "共 39 条样本：严重 7、警告 10、正常 22。健康分 14（D）。",
		Total:          39,
		Critical:       7,
		Warning:        10,
		Normal:         22,
		Findings: []Finding{{
			Type: "基础设施层", Name: "TCP 探测", Severity: "critical", Count: 3,
			Hint: "建议立即排查并处理，必要时扩容或修复故障。",
		}},
		Groups: []ReportGroup{{
			Type:  "k8s集群",
			Stats: GroupStats{Total: 1, Warning: 1},
			Metrics: []MetricSample{{
				Name: "coredns", Instance: "无数据", Value: 0, Threshold: 1, Status: "warning",
				Error: "Prometheus 无返回样本（检查指标名/job 是否与 Telegraf、Blackbox 一致）",
			}},
		}},
	}
	xlsx, err := renderExcel(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(xlsx) < 100 {
		t.Fatalf("excel too small: %d", len(xlsx))
	}
	pdf := renderBinaryPDF(data)
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		n := 8
		if len(pdf) < n {
			n = len(pdf)
		}
		t.Fatalf("not pdf: %q", string(pdf[:n]))
	}
	body := string(pdf)
	if strings.Contains(body, "Helvetica") {
		t.Fatal("pdf still uses Helvetica (Chinese will break)")
	}
	if !strings.Contains(body, "STSong-Light") || !strings.Contains(body, "UniGB-UCS2-H") {
		t.Fatal("pdf missing CJK font resources")
	}
	// 中文「巡」U+5DE1 → UTF-16BE hex 5DE1，不得被替换成 ASCII '?'
	if !strings.Contains(body, "5DE1") {
		t.Fatal("pdf content missing Chinese UTF-16 hex for 巡")
	}
	if strings.Count(body, "(?)") > 0 {
		t.Fatal("pdf still contains ASCII ? placeholders for CJK")
	}
}

func TestSampleNoteText(t *testing.T) {
	t.Parallel()
	got := sampleNoteText("Prometheus 无返回样本（检查指标名/job 是否与 Telegraf、Blackbox 一致）")
	if !strings.Contains(got, "采集无数据") {
		t.Fatalf("got %q", got)
	}
}

func TestReportObjectKey(t *testing.T) {
	t.Parallel()
	if got := reportObjectKey(3, 9, "html"); got != "inspect/3/run-9.html" {
		t.Fatalf("got %s", got)
	}
}

func TestLocalReportStoreRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newLocalReportStore(dir)
	ctx := context.Background()
	key := "inspect/1/run-1.html"
	if err := store.Put(ctx, key, []byte("<html>ok</html>"), "text/html"); err != nil {
		t.Fatal(err)
	}
	b, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "<html>ok</html>" {
		t.Fatalf("got %s", b)
	}
}
