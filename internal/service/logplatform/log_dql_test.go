package logplatform

import "testing"

func TestApplySimplifiedDQL(t *testing.T) {
	q := LogSearchQuery{Keyword: `level:ERROR AND service:payment host:node-1 connection refused`}
	ApplySimplifiedDQL(&q)
	if q.Level != "ERROR" {
		t.Fatalf("level=%q", q.Level)
	}
	if q.ServiceName != "payment" {
		t.Fatalf("service=%q", q.ServiceName)
	}
	if q.Host != "node-1" {
		t.Fatalf("host=%q", q.Host)
	}
	if q.Keyword != "connection refused" {
		t.Fatalf("keyword residual=%q", q.Keyword)
	}
}

func TestApplySimplifiedDQLQuoted(t *testing.T) {
	q := LogSearchQuery{Keyword: `service:"pay svc" level:WARN`}
	ApplySimplifiedDQL(&q)
	if q.ServiceName != "pay svc" {
		t.Fatalf("service=%q", q.ServiceName)
	}
	if q.Level != "WARN" {
		t.Fatalf("level=%q", q.Level)
	}
}

func TestApplySimplifiedDQLFormPrecedence(t *testing.T) {
	q := LogSearchQuery{Level: "INFO", Keyword: "level:ERROR"}
	ApplySimplifiedDQL(&q)
	if q.Level != "INFO" {
		t.Fatalf("form level should win, got %q", q.Level)
	}
}

func TestApplySimplifiedDQLUnknownField(t *testing.T) {
	q := LogSearchQuery{Keyword: "route:/api/v1 foo"}
	ApplySimplifiedDQL(&q)
	if q.ExtraField != "route" || q.ExtraValue != "/api/v1" {
		t.Fatalf("extra=%s:%s", q.ExtraField, q.ExtraValue)
	}
	if q.Keyword != "foo" {
		t.Fatalf("keyword=%q", q.Keyword)
	}
}
