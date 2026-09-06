package k8s

import (
	"errors"
	"strings"
	"testing"

	kom "github.com/weibaohui/kom/kom"
)

func TestMapPodFileErrorDistroless(t *testing.T) {
	err := mapPodFileError(errors.New(`error executing ListFiles: executable file not found in $PATH`), "列出目录")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "distroless") && !strings.Contains(msg, "kube-proxy") && !strings.Contains(msg, "ls/shell") {
		t.Fatalf("unexpected message: %s", msg)
	}
}

func TestMapPodFileErrorMultiContainer(t *testing.T) {
	err := mapPodFileError(errors.New("a container name must be specified for pod xxx"), "列出目录")
	if err == nil || !strings.Contains(err.Error(), "多个容器") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRejectOversizedPodFile(t *testing.T) {
	listFn := func(string) ([]*kom.FileInfo, error) {
		return []*kom.FileInfo{
			{Name: "big.log", Path: "/var/log/big.log", Size: 40 << 20},
			{Name: "ok.log", Path: "/var/log/ok.log", Size: 1024},
		}, nil
	}
	if err := rejectOversizedPodFile(listFn, "/var/log/big.log", 32<<20); err == nil {
		t.Fatal("expected size limit error")
	}
	if err := rejectOversizedPodFile(listFn, "/var/log/ok.log", 32<<20); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if err := rejectOversizedPodFile(func(string) ([]*kom.FileInfo, error) {
		return nil, errors.New("ls failed")
	}, "/var/log/big.log", 32<<20); err != nil {
		t.Fatalf("list failure should soft-pass: %v", err)
	}
}
