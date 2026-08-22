package inspect

import (
	"strings"
	"testing"
)

func TestGradeVerdict(t *testing.T) {
	t.Parallel()
	if got := gradeVerdict("A", 0, 0); got == "" {
		t.Fatal("expected non-empty verdict")
	}
	if got := gradeVerdict("D", 3, 2); got == "" {
		t.Fatal("expected non-empty verdict for D")
	}
}

func TestGradeLabelCN(t *testing.T) {
	t.Parallel()
	if gradeLabelCN("A") != "优秀" {
		t.Fatalf("got %q", gradeLabelCN("A"))
	}
	if gradeLabelCN("D") != "高风险" {
		t.Fatalf("got %q", gradeLabelCN("D"))
	}
}

func TestSeverityRank(t *testing.T) {
	t.Parallel()
	if severityRank("critical") >= severityRank("warning") {
		t.Fatal("critical should rank before warning")
	}
}

func TestCountCheckItems(t *testing.T) {
	t.Parallel()
	n := countCheckItems([]MetricSample{
		{Type: "host", Name: "cpu"},
		{Type: "host", Name: "cpu"},
		{Type: "host", Name: "mem"},
	})
	if n != 2 {
		t.Fatalf("got %d", n)
	}
}

func TestBuildDistBarSVG(t *testing.T) {
	t.Parallel()
	svg := buildDistBarSVG(1, 1, 8)
	if !strings.Contains(string(svg), "<svg") {
		t.Fatalf("got %q", svg)
	}
	if buildDistBarSVG(0, 0, 0) != "" {
		t.Fatal("expected empty for zero total")
	}
}
