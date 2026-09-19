package dictconfig

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

const (
	DefaultPodDebugImage     = "busybox:1.36"
	DictTypePodDebugImage    = "k8s_pod_debug_image"
)

// ResolvePodDebugImage 解析临时调试容器镜像：字典优先，空则回退默认 busybox。
func ResolvePodDebugImage(ctx context.Context, db *gorm.DB) string {
	if v, ok := FetchEnabledDictValueNonEmpty(ctx, db, DictTypePodDebugImage); ok {
		return strings.TrimSpace(v)
	}
	return DefaultPodDebugImage
}
