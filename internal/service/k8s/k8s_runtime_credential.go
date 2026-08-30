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
// 注册 ID 只按「最终使用的凭证」分档：只读 kubeconfig -> :ro，主 kubeconfig -> :w。
// 早期按 intent 分出 :r/:w/:x 三档，但 read/exec/write 在没有只读凭证时用的是同一份主 kubeconfig，
// 会让同一集群在 kom 内注册三个完全等价的实例（各自一套 discovery / CRD watch / ristretto 缓存），
// 既浪费 APIServer 请求也浪费内存。
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
	// 走到这里说明使用主凭证（无只读 kubeconfig，或意图本身需要写/exec 权限）。
	return primary, regID + ":w", nil
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
