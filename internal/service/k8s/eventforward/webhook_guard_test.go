package eventforward

import "testing"

func TestValidateWebhookTargetInternalAliases(t *testing.T) {
	for _, u := range []string{"", "internal", "AlertManager"} {
		if err := validateWebhookTarget(u); err != nil {
			t.Fatalf("expected %q treated as internal, got %v", u, err)
		}
	}
}

func TestValidateWebhookTargetBlocksSSRF(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:8080/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://10.0.0.8/hook",
		"file:///etc/passwd",
	} {
		if err := validateWebhookTarget(u); err == nil {
			t.Fatalf("expected block for %q", u)
		}
	}
}

func TestValidateWebhookTargetAllowsPublic(t *testing.T) {
	if err := validateWebhookTarget("https://93.184.216.34/hook"); err != nil {
		t.Fatalf("expected public URL allowed, got %v", err)
	}
}
