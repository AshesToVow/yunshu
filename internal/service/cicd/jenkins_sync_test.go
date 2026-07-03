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

func TestExtractImageAddressFromConsole(t *testing.T) {
	log := `
[Pipeline] echo
 [TAG] 🏷  镜像Tag: latest_prod_20260701_235120 🏷
[Pipeline] sh
+ docker build -t harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120 .
Step 1/5 : FROM harbor.deploy.local/registry/maven:3.8.6-jdk-8
Successfully tagged harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120
+ docker push harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120
The push refers to repository [harbor.deploy.local/registry/springbootdemo]
latest_prod_20260701_235120: digest: sha256:abc size: 2839
`
	got := extractImageAddressFromConsole(log, "springbootdemo")
	want := "harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	noisy := `
Successfully tagged harbor.deploy.local/registry/inbound-agent:latest-jdk21
Successfully tagged harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120
`
	got2 := extractImageAddressFromConsole(noisy, "springbootdemo")
	if got2 != want {
		t.Fatalf("noisy got %q want %q", got2, want)
	}
}

func TestIsIgnoredCIImage(t *testing.T) {
	if !isIgnoredCIImage("harbor.deploy.local/registry/inbound-agent:latest-jdk21") {
		t.Fatal("expected inbound-agent ignored")
	}
	if isIgnoredCIImage("harbor.deploy.local/registry/springbootdemo:latest_prod_20260701_235120") {
		t.Fatal("expected business image kept")
	}
}
