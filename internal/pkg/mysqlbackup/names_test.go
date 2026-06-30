package mysqlbackup

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMinioObjectKey(t *testing.T) {
	t.Parallel()
	got := BuildMinioObjectKey("project_10.10.10.103_3306_20260630_123727", "sql.gz")
	want := "project_10.10.10.103_3306_20260630_123727.sql.gz"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildArtifactBasename(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 19, 4, 33, 12, 0, time.UTC) // CST 12:33:12
	base := BuildArtifactBasename("测试项目", "175.178.156.23", 3306, at)
	if !strings.Contains(base, "175.178.156.23_3306_20260519_123312") {
		t.Fatalf("unexpected basename: %q", base)
	}
	if strings.Contains(base, "yunshu_mysql") {
		t.Fatal("should not use legacy prefix")
	}
}

func TestBuildArtifactBasenameCST(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 6, 30, 14, 45, 0, 0, time.UTC)
	base := BuildArtifactBasename("proj", "10.10.10.103", 3306, at)
	want := "20260630_224500"
	if !strings.Contains(base, want) {
		t.Fatalf("expected CST timestamp %q in %q", want, base)
	}
}
