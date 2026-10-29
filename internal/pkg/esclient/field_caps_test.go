package esclient

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestFieldCapsReturnsAllTypesPerField(t *testing.T) {
	t.Parallel()
	var gotPath, gotQuery string
	cli, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("fields")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"indices":["a","b"],"fields":{
			"remote":{"text":{"type":"text"},"keyword":{"type":"keyword","ignore_above":256}},
			"remote.keyword":{"keyword":{"type":"keyword","ignore_above":256}},
			"status":{"text":{"type":"text"}},
			"fields.request_time":{"double":{"type":"double"}}
		}}`))
	})
	defer srv.Close()

	caps, err := cli.FieldCaps(context.Background(), "yunshu-agent-*",
		[]string{"remote", "remote.keyword", "status", "fields.request_time", "remote", " "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/yunshu-agent-*/_field_caps" {
		t.Fatalf("path=%q", gotPath)
	}
	// 去重、去空，顺序保持候选顺序
	if gotQuery != "remote,remote.keyword,status,fields.request_time" {
		t.Fatalf("fields query=%q", gotQuery)
	}
	if len(caps["remote"]) != 2 {
		t.Fatalf("remote should expose both text and keyword: %#v", caps["remote"])
	}
	if len(caps["status"]) != 1 || caps["status"][0] != "text" {
		t.Fatalf("status=%#v", caps["status"])
	}
	if len(caps["fields.request_time"]) != 1 || caps["fields.request_time"][0] != "double" {
		t.Fatalf("fields.request_time=%#v", caps["fields.request_time"])
	}
}

func TestFieldCapsErrorOnBadStatus(t *testing.T) {
	t.Parallel()
	cli, srv := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"type":"security_exception"}}`))
	})
	defer srv.Close()

	if _, err := cli.FieldCaps(context.Background(), "yunshu-agent-*", []string{"status"}); err == nil {
		t.Fatal("expected error on 403")
	} else if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error should carry status: %v", err)
	}
}
