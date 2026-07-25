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
		if !strings.Contains(string(b), "demo") {
			t.Fatalf("%s missing project name", code)
		}
	}
}

func TestRenderExcelAndPDF(t *testing.T) {
	t.Parallel()
	data := ReportData{
		Timestamp: time.Now(),
		Project:   "demo",
		Score:     80,
		Grade:     "B",
		Summary:   "summary",
		Findings:  []Finding{{Type: "t", Name: "n", Severity: "warning", Count: 1}},
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
