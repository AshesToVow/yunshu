package inspect

import "testing"

func TestGetStatusGreater(t *testing.T) {
	if getStatus(90, 80, "greater") != "critical" {
		t.Fatal("expected critical")
	}
	if getStatus(75, 80, "greater") != "warning" {
		t.Fatal("expected warning near threshold")
	}
	if getStatus(50, 80, "greater") != "normal" {
		t.Fatal("expected normal")
	}
}

func TestGetStatusEqual(t *testing.T) {
	if getStatus(1, 1, "equal") != "normal" {
		t.Fatal("equal match should be normal")
	}
	if getStatus(0, 1, "equal") != "critical" {
		t.Fatal("equal mismatch should be critical")
	}
}

func TestResolveInstanceTelegrafHost(t *testing.T) {
	got := resolveInstance(map[string]string{
		"__name__": "mem_used_percent",
		"host":     "10.10.10.5",
		"cpu":      "cpu-total",
	})
	if got != "10.10.10.5" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInstancePreferHostOverJob(t *testing.T) {
	got := resolveInstance(map[string]string{
		"job":  "telegraf",
		"host": "db-01",
	})
	if got != "db-01" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInstanceStripScrapePort(t *testing.T) {
	got := resolveInstance(map[string]string{"instance": "10.10.10.5:9273"})
	if got != "10.10.10.5" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveInstanceKeepBlackboxPort(t *testing.T) {
	got := resolveInstance(map[string]string{"instance": "10.10.10.5:3306"})
	if got != "10.10.10.5:3306" {
		t.Fatalf("got %q", got)
	}
}
