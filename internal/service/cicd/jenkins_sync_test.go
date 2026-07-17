package cicd

import (
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestExtractHelmChartRefFromConsole(t *testing.T) {
	harbor := config.HarborConfig{URL: "harbor.deploy.local", ProjectGroup: "registry"}
	log := `
[Pipeline] sh
+ helm push springbootdemo-34.tgz oci://harbor.deploy.local/registry/springbootdemo
Pushed: harbor.deploy.local/registry/springbootdemo:34
`
	got := extractHelmChartRefFromConsole(log, harbor, "springbootdemo")
	want := "oci://harbor.deploy.local/registry/springbootdemo"
	if got != want {
		t.Fatalf("helm push got %q want %q", got, want)
	}

	log2 := `
Successfully packaged chart and saved it to: /tmp/springbootdemo-34.tgz
`
	got2 := extractHelmChartRefFromConsole(log2, harbor, "springbootDemo")
	want2 := "oci://harbor.deploy.local/registry/springbootDemo:34"
	if got2 != want2 {
		t.Fatalf("packaged chart got %q want %q", got2, want2)
	}
}

func TestExtractPackagePathFromConsole(t *testing.T) {
	log := `
[Pipeline] echo
上传 MinIO: frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz
[Pipeline] sh
+ mc cp dist.tar.gz myminio/frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz
`
	got := extractPackagePathFromConsole(log, "frontend-artifacts", "k8s-demo")
	want := "frontend-artifacts/k8s-demo/k8s-demo-20260630_001257-ce7fb16.tar.gz"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractPackagePathFromConsoleBackendDeployInfo(t *testing.T) {
	log := `
[Pipeline] echo
DEPLOY_INFO 应用: springbootDemo | JAR: springbootDemo-20260701_153045-a1b2c3d.jar | Java: /export/server/jdk/bin/java
[Pipeline] sh
+ mc cp target/app.jar myminio/backend-artifacts/springbootDemo/springbootDemo-20260701_153045-a1b2c3d.jar
`
	got := extractPackagePathFromConsole(log, "backend-artifacts", "jenkins-other-name", "springbootDemo")
	want := "backend-artifacts/springbootDemo/springbootDemo-20260701_153045-a1b2c3d.jar"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractPackagePathFromConsoleUsesProjectNameHint(t *testing.T) {
	log := `
[Pipeline] echo
DEPLOY_INFO JAR: springboot-demo-20260701_120000-abc1234.jar
`
	got := extractPackagePathFromConsole(log, "backend-artifacts", "springboot-demo")
	want := "backend-artifacts/springboot-demo/springboot-demo-20260701_120000-abc1234.jar"
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

func TestResolveJenkinsJobName(t *testing.T) {
	if got := resolveJenkinsJobName(nil); got != "" {
		t.Fatalf("nil svc got %q", got)
	}
	svc := &model.CicdService{JenkinsJob: "custom-job", Identifier: "springbootDemo"}
	if got := resolveJenkinsJobName(svc); got != "custom-job" {
		t.Fatalf("got %q want custom-job", got)
	}
	svc.JenkinsJob = ""
	if got := resolveJenkinsJobName(svc); got != "springbootDemo" {
		t.Fatalf("got %q want springbootDemo", got)
	}
}

func TestMinioFolderHints(t *testing.T) {
	svc := &model.CicdService{JenkinsJob: "jenkins-job", Identifier: "springbootDemo"}
	ci := &model.CicdCiConfig{ProjectName: "springbootDemo"}
	hints := minioFolderHints(svc, ci)
	if len(hints) != 2 || hints[0] != "jenkins-job" || hints[1] != "springbootDemo" {
		t.Fatalf("unexpected hints: %#v", hints)
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
