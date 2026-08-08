package cicd

import (
	"errors"
	"strings"
	"testing"
)

func TestMapMinioArtifactsError(t *testing.T) {
	t.Parallel()
	err := mapMinioArtifactsError(errors.New("The specified bucket does not exist"), "frontend-artifacts")
	if err == nil || !strings.Contains(err.Error(), "制品桶不存在") {
		t.Fatalf("bucket missing: %v", err)
	}
	err = mapMinioArtifactsError(errors.New("Access Denied."), "frontend-artifacts")
	if err == nil || !strings.Contains(err.Error(), "鉴权失败") {
		t.Fatalf("access denied: %v", err)
	}
	err = mapMinioArtifactsError(errors.New("dial tcp: connection refused"), "frontend-artifacts")
	if err == nil || !strings.Contains(err.Error(), "无法连接 MinIO") {
		t.Fatalf("connection: %v", err)
	}
	err = mapMinioArtifactsError(errors.New("something else weird"), "frontend-artifacts")
	if err == nil || !strings.Contains(err.Error(), "列出 MinIO 制品失败") {
		t.Fatalf("default: %v", err)
	}
}

func TestIsDeployArtifactName(t *testing.T) {
	t.Parallel()
	if !isDeployArtifactName("app.tar.gz") || !isDeployArtifactName("a.JAR") {
		t.Fatal("expected true")
	}
	if isDeployArtifactName("readme.md") {
		t.Fatal("expected false")
	}
}
