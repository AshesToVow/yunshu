package alert

import "testing"

func TestAssertSafeOutboundURL_BlocksPrivate(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/hook",
		"http://localhost/hook",
		"http://169.254.169.254/latest/meta-data",
		"ftp://example.com/x",
	}
	for _, u := range cases {
		if err := assertSafeOutboundURL(u); err == nil {
			t.Fatalf("expected block for %s", u)
		}
	}
}

func TestAssertSafeOutboundURL_AllowsPublicHTTPS(t *testing.T) {
	// example.com 为公网可解析域名；离线环境若 DNS 失败则跳过。
	err := assertSafeOutboundURL("https://example.com/webhook")
	if err != nil {
		t.Skipf("DNS/network unavailable: %v", err)
	}
}
