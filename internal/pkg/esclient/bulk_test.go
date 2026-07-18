package esclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return &Client{
		baseURL: srv.URL,
		http:    &http.Client{Timeout: 5 * time.Second},
	}, srv
}

// Bulk 部分失败：好文档已写入，坏文档被拒绝。必须返回 Failed>0 且不返回 error，
// 否则消费方会永久重试导致毒消息阻塞 + batch 无限增长。
func TestBulkPartialItemFailureIsNotError(t *testing.T) {
	body := `{"errors":true,"items":[
		{"index":{"status":201}},
		{"index":{"status":400,"error":{"type":"mapper_parsing_exception","reason":"bad field"}}}
	]}`
	cli, srv := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	})
	defer srv.Close()

	res, err := cli.Bulk(context.Background(), []byte("{}\n{}\n{}\n{}\n"))
	if err != nil {
		t.Fatalf("partial item failure must not return transport error, got %v", err)
	}
	if res == nil || res.Failed != 1 {
		t.Fatalf("expected Failed=1, got %+v", res)
	}
	if res.FirstError == "" {
		t.Fatal("expected FirstError sample to be populated")
	}
}

func TestBulkAllSucceed(t *testing.T) {
	cli, srv := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"errors":false,"items":[{"index":{"status":201}}]}`))
	})
	defer srv.Close()

	res, err := cli.Bulk(context.Background(), []byte("{}\n{}\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Failed != 0 {
		t.Fatalf("expected Failed=0, got %+v", res)
	}
}

// 传输级失败（ES 5xx / 不可达）必须返回 error，让消费方重试形成背压。
func TestBulkTransportFailureIsError(t *testing.T) {
	cli, srv := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error":"unavailable"}`))
	})
	defer srv.Close()

	res, err := cli.Bulk(context.Background(), []byte("{}\n{}\n"))
	if err == nil {
		t.Fatal("expected transport error for status 503")
	}
	if res != nil {
		t.Fatalf("expected nil result on transport error, got %+v", res)
	}
}
