package helmscaffold

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
)

//go:embed all:chart
var chartFS embed.FS

var chartNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// Options 生成应用 Chart 脚手架的参数（写入 values / Chart.yaml）。
type Options struct {
	ChartName       string
	ImageRepository string
	ReplicaCount    int
	ContainerPort   int
	ServicePort     int
}

// SanitizeChartName 转为合法 Helm chart 名（小写、数字、连字符）。
func SanitizeChartName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	s = strings.Trim(b.String(), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if len(s) > 63 {
		s = strings.Trim(s[:63], "-")
	}
	if s == "" {
		return "app"
	}
	if !chartNameRe.MatchString(s) {
		return "app"
	}
	return s
}

func (o Options) normalized() Options {
	out := o
	out.ChartName = SanitizeChartName(o.ChartName)
	if out.ReplicaCount <= 0 {
		out.ReplicaCount = 1
	}
	if out.ContainerPort <= 0 {
		out.ContainerPort = 8080
	}
	if out.ServicePort <= 0 {
		out.ServicePort = out.ContainerPort
	}
	repo := strings.TrimSpace(out.ImageRepository)
	if repo == "" {
		repo = "harbor.example/registry/" + out.ChartName
	}
	out.ImageRepository = repo
	return out
}

func (o Options) replace(content string) string {
	n := o.normalized()
	r := strings.NewReplacer(
		"__CHART_NAME__", n.ChartName,
		"__IMAGE_REPOSITORY__", n.ImageRepository,
		"__REPLICA_COUNT__", strconv.Itoa(n.ReplicaCount),
		"__CONTAINER_PORT__", strconv.Itoa(n.ContainerPort),
		"__SERVICE_PORT__", strconv.Itoa(n.ServicePort),
	)
	return r.Replace(content)
}

// BuildZip 生成可解压到业务仓库根目录的 zip：含 helm/（Application）与 setup/（全局固化）。
func BuildZip(opts Options) (filename string, data []byte, err error) {
	n := opts.normalized()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err = fs.WalkDir(chartFS, "chart", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		raw, readErr := chartFS.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		rel := strings.TrimPrefix(p, "chart/")
		rel = strings.TrimPrefix(rel, "chart")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		// chart/helm/... → helm/...；chart/setup/... → setup/...
		zipPath := path.Clean(rel)
		w, createErr := zw.Create(zipPath)
		if createErr != nil {
			return createErr
		}
		_, writeErr := w.Write([]byte(n.replace(string(raw))))
		return writeErr
	})
	if err != nil {
		_ = zw.Close()
		return "", nil, fmt.Errorf("walk scaffold chart: %w", err)
	}
	if err := zw.Close(); err != nil {
		return "", nil, err
	}
	return n.ChartName + "-helm-scaffold.zip", buf.Bytes(), nil
}
