package alert

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParsePrometheusForDuration(t *testing.T) {
	t.Parallel()
	if parsePrometheusForDuration("5m") != 300 {
		t.Fatal("5m")
	}
	if parsePrometheusForDuration("30s") != 30 {
		t.Fatal("30s")
	}
	if parsePrometheusForDuration("") != 0 {
		t.Fatal("empty")
	}
}

func TestParsePromRuleFile(t *testing.T) {
	t.Parallel()
	raw := []byte(`
groups:
  - name: demo
    rules:
      - alert: HighCPU
        expr: up == 0
        for: 2m
        labels:
          severity: critical
      - record: job:up:sum
        expr: sum(up)
`)
	var file promRuleFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.Groups) != 1 || len(file.Groups[0].Rules) != 2 {
		t.Fatalf("%+v", file)
	}
	if file.Groups[0].Rules[0].Alert != "HighCPU" {
		t.Fatal(file.Groups[0].Rules[0].Alert)
	}
	if parsePrometheusForDuration(file.Groups[0].Rules[0].For) != 120 {
		t.Fatal("for")
	}
}
