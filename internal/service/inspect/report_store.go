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

func (s *localReportStore) abs(key string) string {
	key = strings.TrimPrefix(filepath.ToSlash(key), "/")
	return filepath.Join(s.root, filepath.FromSlash(key))
}

func (s *localReportStore) Put(_ context.Context, key string, body []byte, _ string) error {
	path := s.abs(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func (s *localReportStore) Get(_ context.Context, key string) ([]byte, error) {
	return os.ReadFile(s.abs(key))
}

func (s *localReportStore) Delete(_ context.Context, key string) error {
	err := os.Remove(s.abs(key))
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
