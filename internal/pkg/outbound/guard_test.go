package outbound

import "testing"

func TestValidatePublicHTTPURLBlocksUnsafeTargets(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/hook",
		"http://localhost/hook",
		"http://10.1.2.3/hook",
		"http://192.168.1.8/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://metadata.google.internal/nope",
		"ftp://example.com/x",
		"http://user:pass@example.com/x",
		"",
	}
	for _, u := range cases {
		if err := ValidatePublicHTTPURL(u); err == nil {
			t.Fatalf("expected block for %q", u)
		}
	}
}

func TestValidatePublicHTTPURLAllowsPublicIP(t *testing.T) {
	if err := ValidatePublicHTTPURL("https://93.184.216.34/hook"); err != nil {
		t.Fatalf("expected public IP allowed, got %v", err)
	}
}

func TestValidateClusterEndpointURLPolicy(t *testing.T) {
	for _, u := range []string{"https://127.0.0.1:6443", "https://169.254.169.254", "https://localhost:6443", "https://[::1]:6443"} {
		if err := ValidateClusterEndpointURL(u); err == nil {
			t.Fatalf("expected cluster endpoint block for %q", u)
		}
	}
	for _, u := range []string{"https://10.0.0.1:6443", "https://192.168.1.10:6443"} {
		if err := ValidateClusterEndpointURL(u); err != nil {
			t.Fatalf("expected private cluster endpoint allowed for %q, got %v", u, err)
		}
	}
}
