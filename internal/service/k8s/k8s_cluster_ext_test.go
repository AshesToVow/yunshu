package k8s

import (
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestNormalizeKubeconfigForClientGo_InsecureWithCA(t *testing.T) {
	raw := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCg==
    insecure-skip-tls-verify: true
    server: https://10.10.10.103:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
users:
- name: kubernetes-admin
  user:
    token: test-token
`
	out, err := normalizeKubeconfigForClientGo(raw)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if strings.Contains(out, "certificate-authority-data:") {
		t.Fatalf("expected CA data removed when insecure, got: %s", out)
	}
	if _, err := clientcmd.RESTConfigFromKubeConfig([]byte(out)); err != nil {
		t.Fatalf("RESTConfigFromKubeConfig failed after normalize: %v", err)
	}
}

func TestNormalizeKubeconfigForClientGo_SecureWithCA(t *testing.T) {
	raw := `apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCg==
    server: https://10.10.10.103:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
users:
- name: kubernetes-admin
  user:
    token: test-token
`
	out, err := normalizeKubeconfigForClientGo(raw)
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if strings.Contains(out, "insecure-skip-tls-verify:") {
		t.Fatalf("unexpected insecure flag in secure kubeconfig")
	}
	if !strings.Contains(out, "certificate-authority-data:") {
		t.Fatalf("expected CA data preserved for secure kubeconfig")
	}
}

func TestBuildKubeconfigFromDirectConfig_RequiresCAOrInsecure(t *testing.T) {
	t.Parallel()
	_, err := buildKubeconfigFromDirectConfig(&DirectConfig{
		Server: "https://10.0.0.1:6443",
		Token:  "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.e30.x",
	})
	if err == nil || !strings.Contains(err.Error(), "未配置 CA") {
		t.Fatalf("expected CA required error, got %v", err)
	}
	out, err := buildKubeconfigFromDirectConfig(&DirectConfig{
		Server:                "https://10.0.0.1:6443",
		Token:                 "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.e30.x",
		InsecureSkipTLSVerify: true,
	})
	if err != nil {
		t.Fatalf("insecure should allow empty CA: %v", err)
	}
	if !strings.Contains(out, "insecure-skip-tls-verify: true") {
		t.Fatalf("expected insecure flag, got %s", out)
	}
}

func TestClassifyClusterConnectError_TLS(t *testing.T) {
	t.Parallel()
	msg := classifyClusterConnectError(errString("tls: failed to verify certificate: x509: certificate signed by unknown authority"))
	if !strings.Contains(msg, "TLS") {
		t.Fatalf("expected TLS hint, got %q", msg)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
