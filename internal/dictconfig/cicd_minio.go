package dictconfig

import (
	"context"
	"strings"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// CicdMinioDictTypes Yunshu 直连 MinIO（列制品/下载）用的 AK/SK；Jenkins 侧仍用 cicd_minio_credential_id。
type CicdMinioDictTypes struct {
	AccessKey string
	SecretKey string
}

func DefaultCicdMinioDictTypes() CicdMinioDictTypes {
	return CicdMinioDictTypes{
		AccessKey: "cicd_minio_access_key",
		SecretKey: "cicd_minio_secret_key",
	}
}

// ResolveCicdMinioConfig 解析 Yunshu 访问 MinIO 的配置。
// Jenkins 凭据 minio-credentials 无法经 REST 读取，故 AK/SK 与 Jenkins 凭据内容保持一致，写入数据字典。
// 优先级：cicd_minio_* → minio_*（备份模块共用）→ cicd_minio_endpoint / minio_endpoint。
func ResolveCicdMinioConfig(ctx context.Context, db *gorm.DB) MinioConfig {
	base := ResolveMinioConfig(ctx, db, DefaultMinioDictTypes())
	types := DefaultCicdMinioDictTypes()
	if db != nil {
		if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.AccessKey); ok {
			base.AccessKey = v
		}
		if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.SecretKey); ok {
			base.SecretKey = strings.TrimSpace(v)
		}
	}
	cicdCfg := ResolveCicdConfig(ctx, db, config.DefaultCicdConfig(), DefaultCicdDictTypes())
	if ep := strings.TrimSpace(cicdCfg.MinIO.Endpoint); ep != "" {
		useSSL := base.UseSSL
		if v, ok := fetchEnabledDictValue(ctx, db, "minio_use_ssl"); ok {
			if b, ok2 := parseBoolLoose(v); ok2 {
				useSSL = b
			}
		}
		base.Endpoint = NormalizeMinioEndpoint(FormatMinioEndpointURL(ep, useSSL))
	}
	return base
}
