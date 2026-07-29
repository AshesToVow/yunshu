package helmscaffold_test

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"yunshu/internal/service/cicd/helmscaffold"
)

func TestSanitizeChartName(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"My_App":     "my-app",
		"foo.bar":    "foo-bar",
		"":           "app",
		"---":        "app",
		"springboot": "springboot",
	}
	for in, want := range cases {
		if got := helmscaffold.SanitizeChartName(in); got != want {
			t.Fatalf("SanitizeChartName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestBuildZipArchitecture(t *testing.T) {
	t.Parallel()
	name, data, err := helmscaffold.BuildZip(helmscaffold.Options{
		ChartName:       "demo-svc",
		ImageRepository: "harbor.local/registry/demo-svc",
		ReplicaCount:    2,
		ContainerPort:   9090,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, ".zip") {
		t.Fatalf("filename %q", name)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	need := map[string]bool{
		"helm/Chart.yaml":                              false,
		"helm/values.yaml":                             false,
		"helm/values-dev.yaml":                         false,
		"helm/values-prod.yaml":                        false,
		"helm/config-files/.gitkeep":                   false,
		"helm/charts/deployment-base/Chart.yaml":       false,
		"helm/charts/service-base/Chart.yaml":          false,
		"helm/charts/config-base/Chart.yaml":           false,
		"helm/charts/hpa-base/Chart.yaml":              false,
		"helm/charts/pvc-base/Chart.yaml":              false,
		"setup/Chart.yaml":                             false,
		"setup/values.yaml":                            false,
	}
	for _, f := range zr.File {
		if _, ok := need[f.Name]; ok {
			need[f.Name] = true
		}
		if strings.HasPrefix(f.Name, "helm/") || strings.HasPrefix(f.Name, "setup/") {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(rc)
			_ = rc.Close()
			s := string(body)
			if strings.Contains(s, "__CHART_NAME__") || strings.Contains(s, "__IMAGE_REPOSITORY__") {
				t.Fatalf("placeholder left in %s", f.Name)
			}
			if f.Name == "helm/Chart.yaml" {
				if !strings.Contains(s, "name: demo-svc") || !strings.Contains(s, "deployment-base") {
					t.Fatalf("Chart.yaml missing deps: %s", s)
				}
			}
			if f.Name == "helm/values.yaml" {
				if !strings.Contains(s, "replicaCount: 2") || !strings.Contains(s, "containerPort: 9090") {
					t.Fatalf("values.yaml: %s", s)
				}
				for _, key := range []string{"skywalking:", "strategy:", "lifecycle:", "dnsPolicy:", "dnsConfig:", "ports:"} {
					if !strings.Contains(s, key) {
						t.Fatalf("values.yaml missing %s", key)
					}
				}
			}
			if f.Name == "helm/charts/deployment-base/templates/deployment.yaml" {
				for _, key := range []string{"skywalking-agent", "lifecycle:", "dnsPolicy:", "strategy:", ".Values.ports"} {
					if !strings.Contains(s, key) {
						t.Fatalf("deployment.yaml missing %s", key)
					}
				}
			}
		}
	}
	for p, ok := range need {
		if !ok {
			t.Fatalf("missing path in zip: %s", p)
		}
	}
}
