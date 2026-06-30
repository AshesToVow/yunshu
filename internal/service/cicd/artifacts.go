package cicd

import (
	"context"
	"path"
	"sort"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

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
	prefix := strings.Trim(strings.TrimSpace(svc.JenkinsJob), "/") + "/"
	var items []ArtifactItem
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	for obj := range cli.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, bizerrors.Pass(ctx, "cicd", "ListArtifacts", obj.Err)
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
	if !minioCfg.Ready() {
		return nil, constants.ErrBadRequestWithMsg("MinIO 连接未配置完整：请在数据字典维护 cicd_minio_access_key/secret_key，或与 Jenkins minio-credentials 一致的 minio_access_key/secret_key")
	}
	endpoint := dictconfig.NormalizeMinioEndpoint(minioCfg.Endpoint)
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioCfg.AccessKey, minioCfg.SecretKey, ""),
		Secure: minioCfg.UseSSL,
		Region: minioCfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}
