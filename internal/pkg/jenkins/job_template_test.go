package jenkins

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestBuildPipelineJobConfigXML_Frontend(t *testing.T) {
	xml := BuildPipelineJobConfigXML(JobTemplateInput{
		Service: &model.CicdService{
			Name:        "demo-fe",
			Identifier:  "k8s-demo-fe",
			ServiceType: model.CicdServiceTypeFrontend,
		},
		CiConfig: &model.CicdCiConfig{
			GitURL:      "git@gitee.com:org/demo-fe.git",
			RefName:     "main",
			BuildType:   "npm",
			BuildShell:  "run build",
			BuildPath:   "dist",
			NodeVersion: "node18",
		},
		Cfg:        config.DefaultCicdConfig(),
		ScriptPath: "front.jenkinsfile",
	})
	for _, want := range []string{
		"front.jenkinsfile",
		"git@gitee.com:org/demo-fe.git",
		"<name>branchName</name>",
		"<name>nodeToolName</name>",
		"<defaultValue>node18</defaultValue>",
		"<name>SSH_KEY_CREDENTIAL_ID</name>",
		"SSH_KEY_CREDENTIAL_ID=target-server-credential",
		"MINIO_BUCKET=frontend-artifacts",
	} {
		if !strings.Contains(xml, want) {
			t.Fatalf("missing %q in xml:\n%s", want, xml)
		}
	}
}

func TestBuildPipelineJobConfigXML_Backend(t *testing.T) {
	xml := BuildPipelineJobConfigXML(JobTemplateInput{
		Service: &model.CicdService{
			Name:        "demo-be",
			Identifier:  "springboot-demo",
			ServiceType: model.CicdServiceTypeBackend,
		},
		CiConfig: &model.CicdCiConfig{
			GitURL:     "git@gitee.com:org/demo-be.git",
			RefName:    "develop",
			BuildType:  "mvn",
			BuildShell: "clean package -DskipTests",
		},
		Cfg:        config.DefaultCicdConfig(),
		ScriptPath: "backend.jenkinsfile",
	})
	if !strings.Contains(xml, "backend.jenkinsfile") {
		t.Fatalf("expected backend jenkinsfile path")
	}
	if !strings.Contains(xml, "projectName") {
		t.Fatalf("expected backend params")
	}
}
