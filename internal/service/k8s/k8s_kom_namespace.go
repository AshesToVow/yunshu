package k8s

import (
	"context"
	"strconv"

	"yunshu/internal/pkg/constants"

	"github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
)

// ListNamespacesViaKom 通过 kom SDK 列举命名空间；Unauthorized 时强制重注册 kom 后重试一次。
func (s *K8sRuntimeService) ListNamespacesViaKom(ctx context.Context, clusterID uint) ([]corev1.Namespace, error) {
	list, err := s.listNamespacesKomOnce(ctx, clusterID)
	if err == nil {
		return list, nil
	}
	if !isK8sUnauthorizedErr(err) {
		return nil, err
	}

	s.DeleteRegisterCache(clusterID)
	cluster, dbErr := s.repo.GetByID(ctx, clusterID)
	if dbErr != nil {
		return nil, err
	}
	kc, kcErr := s.resolveClusterKubeconfig(cluster)
	if kcErr != nil {
		return nil, err
	}
	cid := strconv.FormatUint(uint64(clusterID), 10)
	if regErr := s.registerClusterIfNeeded(cid, kc, true); regErr != nil {
		return nil, regErr
	}
	return s.listNamespacesKomOnce(ctx, clusterID)
}

func (s *K8sRuntimeService) listNamespacesKomOnce(ctx context.Context, clusterID uint) ([]corev1.Namespace, error) {
	_, k, err := s.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	var list []corev1.Namespace
	if listErr := k.WithContext(ctx).Resource(&corev1.Namespace{}).List(&list).Error; listErr != nil {
		return nil, k8sFail(ctx, "k8s.runtime", "ListNamespacesViaKom", listErr, "cluster_id", clusterID)
	}
	return list, nil
}

// probeClusterListNamespacesKom 心跳/「连接测试」：用网关凭证（裸 clusterID，无 Impersonation）列举命名空间。
// 不可走 ListNamespacesViaKom/GetClusterKubectl：开启 Impersonation 时会伪装为 yunshu:<用户>，
// 集群侧未授权会 403，前端误显示「当前账号无权执行该操作」（平台超管也会中招）。
func (s *K8sRuntimeService) probeClusterListNamespacesKom(ctx context.Context, clusterID uint) error {
	cid := strconv.FormatUint(uint64(clusterID), 10)
	k := kom.Cluster(cid)
	if k == nil {
		return constants.ErrInternalWithMsg(constants.ErrMsg5248c9e19a3f)
	}
	var list []corev1.Namespace
	if listErr := k.WithContext(ctx).Resource(&corev1.Namespace{}).List(&list).Error; listErr != nil {
		return k8sFail(ctx, "k8s.runtime", "probeClusterListNamespacesKom", listErr, "cluster_id", clusterID)
	}
	return nil
}
