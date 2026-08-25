package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultK8sQPS   float32 = 50
	defaultK8sBurst         = 100
	// defaultKomRegisterTimeout 限制 kom 注册时对 APIServer 的单次 HTTP 超时，避免不可达集群拖死总览等批量路径。
	defaultKomRegisterTimeout = 8 * time.Second
	maxPodUploadBytes         = 32 << 20 // 32 MiB
	maxPodDownloadBytes       = 32 << 20 // 32 MiB
	maxPodLogTailLines        = int64(10000)
)

// resolveKubeconfigForIntent 按访问意图选择只读或可写凭证。
func (s *K8sRuntimeService) resolveKubeconfigForIntent(cluster *model.K8sCluster, intent k8sauth.AccessIntent) (string, string, error) {
	if cluster == nil {
		return "", "", fmt.Errorf("cluster is nil")
	}
	primary, err := s.resolveClusterKubeconfig(cluster)
	if err != nil {
		return "", "", err
	}
	regID := strconv.FormatUint(uint64(cluster.ID), 10)
	if intent == k8sauth.AccessIntentRead && strings.TrimSpace(cluster.KubeconfigReadonly) != "" {
		raw, oerr := s.OpenCredential(cluster.KubeconfigReadonly)
		if oerr != nil {
			return "", "", fmt.Errorf("解密只读 kubeconfig 失败: %w", oerr)
		}
		kc, nerr := normalizeKubeconfigForClientGo(strings.TrimSpace(raw))
		if nerr != nil {
			return "", "", nerr
		}
		if kc != "" {
			return kc, regID + ":ro", nil
		}
	}
	suffix := ""
	switch intent {
	case k8sauth.AccessIntentRead:
		suffix = ":r"
	case k8sauth.AccessIntentExec:
		suffix = ":x"
	default:
		suffix = ":w"
	}
	return primary, regID + suffix, nil
}

func applyRestConfigDefaults(cfg *rest.Config) {
	if cfg == nil {
		return
	}
	if cfg.QPS <= 0 {
		cfg.QPS = defaultK8sQPS
	}
	if cfg.Burst <= 0 {
		cfg.Burst = defaultK8sBurst
	}
}

func accessIntentFromContext(ctx context.Context) k8sauth.AccessIntent {
	if scope, ok := k8sauth.RequestScopeFromContext(ctx); ok {
		return k8sauth.AccessIntentFromScope(scope)
	}
	return k8sauth.AccessIntentWrite
}

func restConfigFromKubeconfig(kubeconfig string) (*rest.Config, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	applyRestConfigDefaults(cfg)
	return cfg, nil
}

// RequireDestructiveConfirm 高危操作确认：集群开启 RequireDestructiveConfirm 时须 confirm=true。
func RequireDestructiveConfirm(ctx context.Context, cluster *model.K8sCluster) error {
	if cluster == nil || !cluster.RequireDestructiveConfirm {
		return nil
	}
	if k8sauth.DestructiveConfirmFromContext(ctx) {
		return nil
	}
	return constants.ErrBadRequestWithMsg("高危操作须在请求中携带 confirm=true 确认")
}
