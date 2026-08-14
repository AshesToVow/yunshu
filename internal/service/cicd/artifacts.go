package cicd

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/constants"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ArtifactItem MinIO 中可部署的制品（与 deploy.listMinioArtifacts 过滤规则对齐）。
type ArtifactItem struct {
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	LastModified string `json:"last_modified,omitempty"`
	ObjectKey    string `json:"object_key"`
}

func (s *Service) ListArtifacts(ctx context.Context, projectID, serviceID uint) ([]ArtifactItem, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	cfg := s.resolvedConfig(ctx)
	bucket := dictconfig.MinIOBucketForService(cfg, svc.ServiceType)
	if bucket == "" {
		return nil, constants.ErrBadRequestWithMsg("MinIO 制品桶未配置，请在数据字典设置 cicd_minio_bucket_frontend/backend")
	}
	cli, err := s.newCicdMinioClient(ctx)
	if err != nil {
		return nil, err
	}
	folder := strings.Trim(strings.TrimSpace(svc.JenkinsJob), "/")
	if folder == "" {
		folder = strings.Trim(strings.TrimSpace(svc.Identifier), "/")
	}
	if folder == "" {
		return nil, constants.ErrBadRequestWithMsg("无法确定制品目录：请先配置应用唯一标识或 Jenkins Job 名称")
	}
	prefix := folder + "/"
	var items []ArtifactItem
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	for obj := range cli.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, mapMinioArtifactsError(obj.Err, bucket)
		}
		name := path.Base(obj.Key)
		if name == "" || name == "/" {
			continue
		}
		if !isDeployArtifactName(name) {
			continue
		}
		items = append(items, ArtifactItem{
			Name:         name,
			Size:         obj.Size,
			LastModified: obj.LastModified.Format("2006-01-02 15:04:05"),
			ObjectKey:    obj.Key,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name > items[j].Name
	})
	return items, nil
}

func isDeployArtifactName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".jar") ||
		strings.HasSuffix(lower, ".bin")
}

func (s *Service) newCicdMinioClient(ctx context.Context) (*minio.Client, error) {
	minioCfg := dictconfig.ResolveCicdMinioConfig(ctx, s.db)
	// 列制品只需要 endpoint + AK/SK；桶名由 cicd_minio_bucket_* 单独决定，不要求备份用的 minio_bucket。
	if strings.TrimSpace(minioCfg.Endpoint) == "" ||
		strings.TrimSpace(minioCfg.AccessKey) == "" ||
		strings.TrimSpace(minioCfg.SecretKey) == "" {
		return nil, constants.ErrBadRequestWithMsg("MinIO 连接未配置完整：请在数据字典维护 cicd_minio_endpoint 与 cicd_minio_access_key/secret_key（须与 Jenkins minio-credentials 一致；也可回退 minio_*）")
	}
	endpoint := dictconfig.NormalizeMinioEndpoint(minioCfg.Endpoint)
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.UseSSL,
		Region: minioCfg.Region,
	})
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("MinIO 客户端初始化失败，请检查 cicd_minio_endpoint：" + truncateErr(err.Error(), 120))
	}
	return cli, nil
}

func mapMinioArtifactsError(err error, bucket string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "nosuchbucket"),
		strings.Contains(lower, "the specified bucket does not exist"):
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(
			"MinIO 制品桶不存在：%s。请先在 MinIO 创建该桶，或检查数据字典 cicd_minio_bucket_frontend/backend",
			bucket,
		))
	case strings.Contains(lower, "access denied"),
		strings.Contains(lower, "invalidaccesskeyid"),
		strings.Contains(lower, "signaturedoesnotmatch"),
		strings.Contains(lower, "invalid access key"):
		return constants.ErrBadRequestWithMsg("MinIO 鉴权失败：请核对 cicd_minio_access_key/secret_key（须与 Jenkins 凭据 minio-credentials 一致）")
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "timeout"),
		strings.Contains(lower, "no such host"),
		strings.Contains(lower, "network is unreachable"),
		strings.Contains(lower, "connection reset"):
		return constants.ErrBadRequestWithMsg("无法连接 MinIO：请检查 cicd_minio_endpoint（S3 API，一般为 :9000，不是控制台 :9001）及网络连通性")
	default:
		// 业务可读错误，避免 Pass 包装成英文 operation failed
		return constants.ErrBadRequestWithMsg("列出 MinIO 制品失败：" + truncateErr(msg, 160))
	}
}

func truncateErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
