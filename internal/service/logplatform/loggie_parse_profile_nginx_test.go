package logplatform

import (
	"regexp"
	"testing"
)

// 回归：log_format 在 combined 之后追加 "$http_x_forwarded_for" $request_time $upstream_response_time
// 时，整行仍须解析出 status/uri/request_time，否则 Nginx 访问统计恒为空。
func TestProfileNginxAccessTailFields(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(profileNginxAccess().regexPattern)
	cases := []struct {
		name        string
		line        string
		wantStatus  string
		wantURI     string
		wantReqTime string
	}{
		{
			name:        "xff+request_time+upstream",
			line:        `192.168.1.10 - - [28/Oct/2026:20:15:30 +0800] "GET /api/v1/users?page=1 HTTP/1.1" 200 1234 "https://example.com/" "Mozilla/5.0 (X11; Linux x86_64)" "10.0.0.1, 10.0.0.2" 0.123 0.120`,
			wantStatus:  "200",
			wantURI:     "/api/v1/users?page=1",
			wantReqTime: "0.123",
		},
		{
			name:        "combined_only",
			line:        `1.2.3.4 - - [28/Oct/2026:20:15:30 +0800] "POST /login HTTP/1.1" 302 0 "-" "curl/8.0"`,
			wantStatus:  "302",
			wantURI:     "/login",
			wantReqTime: "",
		},
		{
			name:        "combined_request_time",
			line:        `1.2.3.4 - - [28/Oct/2026:20:15:30 +0800] "GET /health HTTP/1.1" 200 2 "-" "kube-probe/1.27" 0.001`,
			wantStatus:  "200",
			wantURI:     "/health",
			wantReqTime: "0.001",
		},
		{
			name:        "upstream_absent_dash",
			line:        `10.1.1.1 - - [28/Oct/2026:20:15:30 +0800] "GET /a HTTP/2.0" 404 18 "-" "Mozilla/5.0" "-" 0.456 -`,
			wantStatus:  "404",
			wantURI:     "/a",
			wantReqTime: "0.456",
		},
		{
			name:        "upstream_multiple",
			line:        `10.1.1.1 - - [28/Oct/2026:20:15:30 +0800] "GET /b HTTP/1.1" 502 100 "-" "Mozilla/5.0" "-" 0.5 0.4, 0.3`,
			wantStatus:  "502",
			wantURI:     "/b",
			wantReqTime: "0.5",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := re.FindStringSubmatch(tc.line)
			if m == nil {
				t.Fatalf("nginx access line not matched: %s", tc.line)
			}
			got := map[string]string{}
			for i, name := range re.SubexpNames() {
				if name != "" && i < len(m) {
					got[name] = m[i]
				}
			}
			if got["status"] != tc.wantStatus {
				t.Fatalf("status got %q want %q", got["status"], tc.wantStatus)
			}
			if got["uri"] != tc.wantURI {
				t.Fatalf("uri got %q want %q", got["uri"], tc.wantURI)
			}
			if got["request_time"] != tc.wantReqTime {
				t.Fatalf("request_time got %q want %q", got["request_time"], tc.wantReqTime)
			}
		})
	}
}

func TestProfileNginxAccessMultilinePattern(t *testing.T) {
	t.Parallel()
	re := regexp.MustCompile(profileNginxAccess().multilinePattern)
	if !re.MatchString(`192.168.1.10 - - [28/Oct/2026:20:15:30 +0800] "GET / HTTP/1.1" 200 2 "-" "curl"`) {
		t.Fatal("multiline pattern should treat access log line as new event")
	}
}
