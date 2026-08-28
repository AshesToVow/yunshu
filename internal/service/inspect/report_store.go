package inspect

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"yunshu/internal/pkg/objectstore"

	"gorm.io/gorm"
)

const (
	StorageLocal = "local"
	StorageMinio = "minio"
)

// ReportStore 巡检报告读写抽象。
type ReportStore interface {
	Put(ctx context.Context, key string, body []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Backend() string
}

type localReportStore struct {
	root string
}

func newLocalReportStore(root string) *localReportStore {
	_ = os.MkdirAll(root, 0o755)
	return &localReportStore{root: root}
}

func (s *localReportStore) Backend() string { return StorageLocal }

func (s *localReportStore) abs(key string) (string, error) {
	return safeLocalReportPath(s.root, key)
}

// safeLocalReportPath 将对象 key 解析为 reportDir 下的绝对路径，拒绝穿越。
func safeLocalReportPath(root, key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", fmt.Errorf("empty key")
	}
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid key")
	}
	slashKey := strings.TrimPrefix(filepath.ToSlash(key), "/")
	if !strings.HasPrefix(slashKey, "inspect/") {
		return "", fmt.Errorf("invalid key prefix")
	}
	root = filepath.Clean(root)
	abs := filepath.Join(root, filepath.FromSlash(slashKey))
	abs = filepath.Clean(abs)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path outside root")
	}
	return abs, nil
}

// resolveLegacyReportPath 兼容历史 DB 中存储的本地路径，仅允许落在 reportDir 内。
func resolveLegacyReportPath(reportDir, key string) (string, bool) {
	key = strings.TrimSpace(key)
	if key == "" || strings.Contains(key, "..") {
		return "", false
	}
	reportDir = filepath.Clean(reportDir)
	cleanKey := filepath.Clean(key)
	if strings.HasPrefix(cleanKey, reportDir+string(os.PathSeparator)) {
		return cleanKey, true
	}
	if safe, err := safeLocalReportPath(reportDir, key); err == nil {
		return safe, true
	}
	// 旧格式 logs/inspect-reports/{pid}/run-{id}.html
	const legacyPrefix = "logs/inspect-reports/"
	slash := filepath.ToSlash(key)
	if after, ok := strings.CutPrefix(slash, legacyPrefix); ok {
		rel := after
		if safe, err := safeLocalReportPath(reportDir, "inspect/"+rel); err == nil {
			return safe, true
		}
	}
	return "", false
}

func (s *localReportStore) Put(_ context.Context, key string, body []byte, _ string) error {
	path, err := s.abs(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func (s *localReportStore) Get(_ context.Context, key string) ([]byte, error) {
	path, err := s.abs(key)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (s *localReportStore) Delete(_ context.Context, key string) error {
	path, err := s.abs(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

type minioReportStore struct {
	cli *objectstore.Client
}

func (s *minioReportStore) Backend() string { return StorageMinio }

func (s *minioReportStore) Put(ctx context.Context, key string, body []byte, contentType string) error {
	return s.cli.PutBytes(ctx, key, body, contentType)
}

func (s *minioReportStore) Get(ctx context.Context, key string) ([]byte, error) {
	return s.cli.GetBytes(ctx, key)
}

func (s *minioReportStore) Delete(ctx context.Context, key string) error {
	return s.cli.RemoveObject(ctx, key)
}

// resolveReportStore 优先 MinIO，配置不完整时降级本地。
func resolveReportStore(ctx context.Context, db *gorm.DB, localRoot string) ReportStore {
	local := newLocalReportStore(localRoot)
	if db == nil {
		return local
	}
	cli, err := objectstore.NewFromDB(ctx, db)
	if err != nil {
		slog.Default().With("component", "inspect.store").Info("inspect reports use local storage", "reason", err.Error())
		return local
	}
	return &minioReportStore{cli: cli}
}

func reportObjectKey(projectID, runID uint, ext string) string {
	ext = strings.TrimPrefix(ext, ".")
	return fmt.Sprintf("inspect/%d/run-%d.%s", projectID, runID, ext)
}
