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
		Verdict:   "整体稳定",
		Assets: []AssetBucket{
			{Key: "基础设施层", Label: "基础设施层", Total: 10, Normal: 9, Abnormal: 1, Rate: 90},
			{Key: "k8s集群", Label: "k8s集群", Total: 2, Normal: 2, Abnormal: 0, Rate: 100},
		},
		CategorySections: []CategorySection{
			{Key: "基础设施层", Title: "基础设施层", Kind: "host", Total: 10, Normal: 9, Abnormal: 1, Rate: 90},
			{Key: "k8s集群", Title: "k8s集群", Kind: "k8s", Total: 2, Normal: 2, Abnormal: 0, Rate: 100,
				Metrics: []MetricSample{{Name: "Node Ready", Instance: "n1", Value: 1, Status: "normal"}},
			},
		},
		DetailPageStart: 4,
		StarRating:      "★★★★★",
		Conclusion:      "本次巡检共覆盖基础设施层 10 项、k8s集群 2 项。",
		Groups: []ReportGroup{{
			Type:    "基础设施层",
			Stats:   GroupStats{Total: 1, Normal: 1},
			Metrics: []MetricSample{{Name: "cpu", Instance: "a", Value: 1, Status: "normal"}},
		}},
	}
	b, err := renderHTMLWithTemplate("default", "", data)
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	for _, want := range []string{"demo", "巡检结论", "巡检统计", "风险问题汇总", "问题整改", "巡检总结", "k8s集群"} {
		if !strings.Contains(html, want) {
			t.Fatalf("default template missing %q", want)
		}
	}
	for _, code := range []string{"compact", "executive", "print"} {
		b, err := renderHTMLWithTemplate(code, "", data)
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if !strings.Contains(string(b), "demo") {
			t.Fatalf("%s missing project", code)
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
	// 服务端不再生成 PDF（仅浏览器 html2canvas/jsPDF）
	if got := renderBinaryPDF(data, nil); got != nil {
		t.Fatalf("server pdf should be disabled, got %d bytes", len(got))
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

func TestLocalReportStorePathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newLocalReportStore(dir)
	ctx := context.Background()
	if err := store.Put(ctx, "inspect/1/../../../etc/passwd", []byte("x"), "text/plain"); err == nil {
		t.Fatal("expected reject for traversal key on put")
	}
	if _, err := store.Get(ctx, "../secret"); err == nil {
		t.Fatal("expected reject for traversal key")
	}
}

func TestSafeLocalReportPath(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := safeLocalReportPath(root, "inspect/1/run-1.html"); err != nil {
		t.Fatal(err)
	}
	if _, err := safeLocalReportPath(root, "../evil"); err == nil {
		t.Fatal("expected error")
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
