package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultK8sQPS   float32 = 50
	defaultK8sBurst         = 100
	maxPodUploadBytes       = 32 << 20 // 32 MiB
	maxPodLogTailLines      = int64(10000)
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

func applyImpersonation(cfg *rest.Config, cluster *model.K8sCluster, actor *auth.CurrentUser) {
	if cfg == nil || cluster == nil || !cluster.ImpersonateEnabled || actor == nil {
		return
	}
	prefix := strings.TrimSpace(cluster.ImpersonateUserPrefix)
	if prefix == "" {
		prefix = "yunshu:"
	}
	user := strings.TrimSpace(actor.Username)
	if user == "" {
		user = fmt.Sprintf("uid-%d", actor.ID)
	}
	groups := make([]string, 0, len(actor.RoleCodes)+1)
	groups = append(groups, prefix+"authenticated")
	for _, rc := range actor.RoleCodes {
		rc = strings.TrimSpace(rc)
		if rc == "" {
			continue
		}
		groups = append(groups, prefix+"role:"+rc)
	}
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: prefix + user,
		Groups:   groups,
	}
}

func accessIntentFromContext(ctx context.Context) k8sauth.AccessIntent {
	if scope, ok := k8sauth.RequestScopeFromContext(ctx); ok {
		return k8sauth.AccessIntentFromScope(scope)
	}
	return k8sauth.AccessIntentWrite
}

func actorFromContext(ctx context.Context) *auth.CurrentUser {
	if u, ok := auth.RequestUserFromContext(ctx); ok {
		return u
	}
	return nil
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
