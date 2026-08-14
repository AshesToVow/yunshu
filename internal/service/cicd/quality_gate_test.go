package cicd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestQualityGateBlockReason_Disabled(t *testing.T) {
	cfg := config.CicdConfig{Sonar: config.CicdSonarConfig{Enabled: false, GateBlock: true}}
	pass := false
	br := &model.CicdBuildRun{
		QualityGateStatus: model.CicdQualityGateError,
		SecurityScanPass:  &pass,
	}
	if err := qualityGateBlockReason(cfg, br); err != nil {
		t.Fatalf("expected nil when sonar disabled, got %v", err)
	}
}

func TestQualityGateBlockReason_ErrorBlocked(t *testing.T) {
	cfg := config.CicdConfig{Sonar: config.CicdSonarConfig{Enabled: true, GateBlock: true}}
	br := &model.CicdBuildRun{QualityGateStatus: model.CicdQualityGateError}
	if err := qualityGateBlockReason(cfg, br); err == nil {
		t.Fatal("expected error for ERROR gate")
	}
}

func TestQualityGateBlockReason_OKAllowed(t *testing.T) {
	cfg := config.CicdConfig{Sonar: config.CicdSonarConfig{Enabled: true, GateBlock: true}}
	br := &model.CicdBuildRun{QualityGateStatus: model.CicdQualityGateOK}
	if err := qualityGateBlockReason(cfg, br); err != nil {
		t.Fatalf("expected nil for OK, got %v", err)
	}
}

func TestVerifyJenkinsCallbackHMAC(t *testing.T) {
	body := []byte(`{"event":"stage"}`)
	secret := "test-secret"
	if VerifyJenkinsCallbackHMAC(secret, body, "sha256=deadbeef") {
		t.Fatal("expected false for bad signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !VerifyJenkinsCallbackHMAC(secret, body, sig) {
		t.Fatal("expected true for valid signature")
	}
}

func TestApplySonarParams_FromDict(t *testing.T) {
	cfg := config.DefaultCicdConfig()
	cfg.Sonar.Enabled = true
	cfg.Sonar.URL = "https://sonar.example.com"
	cfg.Callback.CallbackURL = "https://yunshu.example.com/api/v1/cicd/jenkins/callback"
	params := BuildJenkinsParams(BuildParamsInput{
		Service: &model.CicdService{
			ServiceType: model.CicdServiceTypeBackend,
			Identifier:  "demo",
		},
		CiConfig: &model.CicdCiConfig{
			GitURL:    "git@example.com:a/b.git",
			RefName:   "main",
			BuildType: "mvn",
		},
		Cfg:              cfg,
		PublishMode:      model.CicdPublishModeBuildOnly,
		YunshuBuildRunID: 42,
	})
	if params["enableSonar"] != "true" {
		t.Fatalf("enableSonar=%q", params["enableSonar"])
	}
	if params["SONAR_HOST_URL"] != "https://sonar.example.com" {
		t.Fatalf("SONAR_HOST_URL=%q", params["SONAR_HOST_URL"])
	}
	if params["YUNSHU_BUILD_RUN_ID"] != "42" {
		t.Fatalf("YUNSHU_BUILD_RUN_ID=%q", params["YUNSHU_BUILD_RUN_ID"])
	}
	paramsCD := BuildJenkinsParams(BuildParamsInput{
		Service:             &model.CicdService{ServiceType: model.CicdServiceTypeBackend, Identifier: "demo"},
		Cfg:                 cfg,
		PublishMode:         model.CicdPublishModeArtifactDeploy,
		YunshuReleaseRunID:  99,
	})
	if paramsCD["enableSonar"] != "false" {
		t.Fatalf("CD should disable sonar, got %q", paramsCD["enableSonar"])
	}
	if paramsCD["YUNSHU_RUN_KIND"] != model.CicdRunKindRelease {
		t.Fatalf("YUNSHU_RUN_KIND=%q", paramsCD["YUNSHU_RUN_KIND"])
	}
	if paramsCD["YUNSHU_BUILD_RUN_ID"] != "99" {
		t.Fatalf("YUNSHU_BUILD_RUN_ID=%q", paramsCD["YUNSHU_BUILD_RUN_ID"])
	}
}
