package inspect

import (
	"context"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/objectstore"

	"gorm.io/gorm"
)

// ReportStorageInfo 巡检报告存储状态（供管理台展示与 MinIO 未配置提示）。
type ReportStorageInfo struct {
	Backend     string `json:"backend"`                // minio | local
	MinioReady  bool   `json:"minio_ready"`            // 数据字典 MinIO 是否可用
	LocalRoot   string `json:"local_root,omitempty"`   // 本地存储根目录
	MinioReason string `json:"minio_reason,omitempty"` // MinIO 不可用时原因
}

// resolveReportStorageInfo 判定当前巡检报告会写入 MinIO 还是本地。
func resolveReportStorageInfo(ctx context.Context, db *gorm.DB, localRoot string) ReportStorageInfo {
	info := ReportStorageInfo{
		Backend:   StorageLocal,
		LocalRoot: localRoot,
	}
	if db == nil {
		info.MinioReason = "数据库不可用"
		return info
	}
	cfg := dictconfig.ResolveMinioConfig(ctx, db, dictconfig.DefaultMinioDictTypes())
	if !cfg.Ready() {
		info.MinioReason = "数据字典 MinIO 配置未启用或不完整（minio_endpoint / minio_access_key / minio_secret_key / minio_bucket）"
		return info
	}
	if _, err := objectstore.NewFromDB(ctx, db); err != nil {
		info.MinioReason = err.Error()
		return info
	}
	info.Backend = StorageMinio
	info.MinioReady = true
	return info
}
