package cicd

import "testing"

func TestExtractPackagePathFromConsole(t *testing.T) {
	log := `
[Pipeline] echo
上传 MinIO: frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz
[Pipeline] sh
+ mc cp dist.tar.gz myminio/frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz
`
	got := extractPackagePathFromConsole(log, "k8s-demo", "frontend-artifacts")
	want := "frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildArtifactPackagePath(t *testing.T) {
	got := buildArtifactPackagePath("k8s-demo", "frontend-artifacts", "k8s-demo-20260630_001257-ce7fb16.tar.gz")
	want := "frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
