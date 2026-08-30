package k8s

// K8sPodService 主文件：仅保留服务装配与 Pod 生命周期编排（List/Detail/Events/Delete/
// Restart/Create*/Update*）。其余职责已按域分文件：
//   - DTO：k8s_pod_types.go
//   - Exec/TTY：k8s_pod_exec.go
//   - 日志：k8s_pod_logs.go
//   - 容器内文件：k8s_pod_files.go
//   - 结构体构建与展示映射：k8s_pod_build.go
//   - 确定性诊断：k8s_pod_diagnose.go

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type K8sPodService struct {
	runtime     *K8sRuntimeService
	dyn         *DynamicResourceService
	nsDenyRepo  interfaces.K8sNamespaceDenyRepository
	nsAllowRepo interfaces.K8sNamespaceAllowRepository
}

// NewK8sPodService 创建相关逻辑。
func NewK8sPodService(
	runtime *K8sRuntimeService,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
) *K8sPodService {
	return &K8sPodService{
		runtime: runtime, dyn: NewDynamicResourceService(runtime),
		nsDenyRepo: nsDeny, nsAllowRepo: nsAllow,
	}
}

// List 查询列表相关的业务逻辑。
func (s *K8sPodService) List(ctx context.Context, query PodListQuery) ([]PodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(query.Namespace)
	clusterWide := ns == ""
	var pods []corev1.Pod
	listQuery := k.WithContext(ctx).Resource(&corev1.Pod{})
	if clusterWide {
		listQuery = listQuery.AllNamespace()
	} else {
		listQuery = listQuery.Namespace(ns)
	}
	if err := listQuery.List(&pods).Error; err != nil {
		return nil, err
	}

	var usageByKey map[string]podCPUMemUsage
	if clusterWide {
		usageByKey = listAllPodCPUMemUsage(ctx, s.dyn, k)
	} else {
		usageByKey = listPodCPUMemUsageByNamespace(ctx, s.dyn, k, ns)
	}
	nodeAlloc := map[string]nodeAllocResources{}
	{
		var nodes []corev1.Node
		if e := k.WithContext(ctx).Resource(&corev1.Node{}).List(&nodes).Error; e == nil {
			for _, n := range nodes {
				nodeAlloc[n.Name] = nodeAllocResources{
					CPU: n.Status.Allocatable[corev1.ResourceCPU],
					Mem: n.Status.Allocatable[corev1.ResourceMemory],
				}
			}
		}
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]PodItem, 0, len(pods))
	for _, p := range pods {
		if !s.namespaceAllowed(ctx, query.ClusterID, p.Namespace) {
			continue
		}
		usageKey := p.Name
		if clusterWide {
			usageKey = podMetricKey(p.Namespace, p.Name)
		}
		item := mapPodItem(p, usageByKey[usageKey], nodeAlloc[p.Spec.NodeName])
		if kw != "" {
			if !strings.Contains(strings.ToLower(item.Name), kw) && !strings.Contains(strings.ToLower(item.NodeName), kw) {
				continue
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *K8sPodService) namespaceAllowed(ctx context.Context, clusterID uint, namespace string) bool {
	ns := strings.TrimSpace(namespace)
	if ns == "" {
		return true
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return true
	}
	// super-admin 仍受 NS deny/allow 约束（与 K8sScopeAuthorize 一致）
	if s.nsDenyRepo == nil && s.nsAllowRepo == nil {
		return true
	}
	pack := k8sauth.PackFromCurrentUser(u)
	allowed, err := NamespaceAllowedByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, pack, clusterID, ns)
	return err == nil && allowed
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sPodService) Detail(ctx context.Context, query PodDetailQuery) (*PodDetail, error) {
	if !s.namespaceAllowed(ctx, query.ClusterID, query.Namespace) {
		return nil, constants.ErrForbiddenWithMsg("当前主体在此集群下禁止访问命名空间「" + strings.TrimSpace(query.Namespace) + "」")
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var pod corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(query.Namespace).Name(query.Name).Get(&pod).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmtc52b9130d74c)
	}
	containers := make([]PodContainerInfo, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		containers = append(containers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        k8sutil.ContainerState(c.State),
		})
	}
	initContainers := make([]PodContainerInfo, 0, len(pod.Status.InitContainerStatuses))
	for _, c := range pod.Status.InitContainerStatuses {
		initContainers = append(initContainers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        k8sutil.ContainerState(c.State),
		})
	}
	startTime := time.Time{}
	if pod.Status.StartTime != nil {
		startTime = pod.Status.StartTime.Time
	}
	return &PodDetail{
		Name:              pod.Name,
		Namespace:         pod.Namespace,
		UID:               string(pod.UID),
		Phase:             string(pod.Status.Phase),
		NodeName:          pod.Spec.NodeName,
		ServiceAccount:    pod.Spec.ServiceAccountName,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		QOSClass:          string(pod.Status.QOSClass),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		Containers:        containers,
		InitContainers:    initContainers,
		Conditions:        pod.Status.Conditions,
		Volumes:           pod.Spec.Volumes,
		Tolerations:       pod.Spec.Tolerations,
		NodeSelector:      pod.Spec.NodeSelector,
		PriorityClassName: pod.Spec.PriorityClassName,
		Affinity:          pod.Spec.Affinity,
		StartTime:         startTime,
		CreationTime:      pod.CreationTimestamp.Time,
	}, nil
}

// Events 执行对应的业务逻辑。
func (s *K8sPodService) Events(ctx context.Context, query PodEventQuery) ([]PodEventItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var list []corev1.Event
	if err := k.WithContext(ctx).
		Resource(&corev1.Event{}).
		Namespace(query.Namespace).
		WithFieldSelector(fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", query.Name)).
		List(&list).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.pod", "api", err, constants.ErrFmtbf8c73dd9a9e)
	}
	out := make([]PodEventItem, 0, len(list))
	for _, e := range list {
		out = append(out, PodEventItem{
			Type:           e.Type,
			Reason:         e.Reason,
			Message:        e.Message,
			Count:          e.Count,
			FirstTimestamp: e.FirstTimestamp.Time,
			LastTimestamp:  e.LastTimestamp.Time,
		})
	}
	return out, nil
}

// Delete 删除相关的业务逻辑。
func (s *K8sPodService) Delete(ctx context.Context, req PodDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	podGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
	if err := s.dyn.DeleteByGVK(ctx, k, podGVK, req.Namespace, req.Name, req.K8sDeleteOptions); err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}

// Restart 执行对应的业务逻辑。
func (s *K8sPodService) Restart(ctx context.Context, req PodRestartRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Delete().Error; err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}

// CreateByYAML 创建相关的业务逻辑。
func (s *K8sPodService) CreateByYAML(ctx context.Context, req PodCreateYAMLRequest) error {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := assertK8sWritable(ctx, cluster, "apply", ""); err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest, nil); err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}

// CreateSimple 创建相关的业务逻辑。
func (s *K8sPodService) CreateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k8sutil.ValidateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := k8sutil.ValidateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		// default container name equals pod name; validate as label too
		if err := k8sutil.ValidateRFC1123Label(req.Name); err != nil {
			return err
		}
	}
	pod := buildSimplePod(req)
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Create(pod).Error; err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}

// UpdateSimple 更新相关的业务逻辑。
func (s *K8sPodService) UpdateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k8sutil.ValidateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := k8sutil.ValidateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		if err := k8sutil.ValidateRFC1123Label(req.Name); err != nil {
			return err
		}
	}

	var existing corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Get(&existing).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsg86c3cb5f9474)
		}
		return k8sFail(ctx, "k8s.pod", "api", err)
	}

	desired := buildSimplePod(req)
	if msg := workloadManagedPodHint(ctx, k, &existing); msg != "" {
		return constants.ErrBadRequestWithMsg(msg)
	}
	if k8sutil.CanUpdateImageOnly(&existing, desired) {
		// Kubernetes 生态做法：仅更新镜像时，不删除重建 Pod
		copyPod := existing.DeepCopy()
		copyPod.Spec.Containers[0].Image = desired.Spec.Containers[0].Image
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Update(copyPod).Error; err != nil {
			return k8sFail(ctx, "k8s.pod", "api", err)
		}
		return nil
	}

	// 其他字段多为不可变：按同名删除后重建，并等待删除完成避免 Terminating 占用导致创建失败
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Delete().Error; err != nil && !apierrors.IsNotFound(err) {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		var current corev1.Pod
		err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Get(&current).Error
		if err != nil {
			if apierrors.IsNotFound(err) {
				break
			}
			return k8sFail(ctx, "k8s.pod", "api", err)
		}
		if time.Now().After(deadline) {
			return constants.ErrBadRequestWithMsg(constants.ErrMsgcc094caf1644)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Create(desired).Error; err != nil {
		return k8sFail(ctx, "k8s.pod", "api", err)
	}
	return nil
}
