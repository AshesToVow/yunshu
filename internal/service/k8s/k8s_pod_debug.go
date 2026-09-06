package k8s

// Pod 临时调试容器（ephemeral container）：用于 distroless/scratch 等无法直接 Exec/文件管理的镜像。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/dictconfig"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodDebugRequest struct {
	ClusterID       uint   `json:"cluster_id" binding:"required"`
	Namespace       string `json:"namespace" binding:"required"`
	Name            string `json:"name" binding:"required"`
	TargetContainer string `json:"target_container"` // 可选：共享目标容器命名空间
	Image           string `json:"image"`            // 可选；空则用字典 k8s_pod_debug_image / 默认 busybox
	ContainerName   string `json:"container_name"`   // 可选：指定临时容器名
}

type PodDebugResult struct {
	EphemeralContainer string `json:"ephemeral_container"`
	Image              string `json:"image"`
	Ready              bool   `json:"ready"`
	Message            string `json:"message"`
}

// ResolveDebugImage 解析调试镜像：请求覆盖 > 数据字典 > 内置默认。
func (s *K8sPodService) ResolveDebugImage(ctx context.Context, override string) string {
	if v := strings.TrimSpace(override); v != "" {
		return v
	}
	return dictconfig.ResolvePodDebugImage(ctx, s.db)
}

// DebugEphemeral 向 Pod 注入临时调试容器（需集群支持 EphemeralContainers）。
func (s *K8sPodService) DebugEphemeral(ctx context.Context, req PodDebugRequest) (*PodDebugResult, error) {
	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	if err := assertK8sWritable(ctx, cluster, "exec", req.Namespace); err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	if ns == "" || name == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsge278df185255)
	}
	image := s.ResolveDebugImage(ctx, req.Image)
	ecName := strings.TrimSpace(req.ContainerName)
	if ecName == "" {
		ecName = fmt.Sprintf("yunshu-debug-%d", time.Now().Unix()%100000)
	}
	if len(ecName) > 63 {
		ecName = ecName[:63]
	}

	var pod corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).Name(name).Get(&pod).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg("Pod 不存在")
		}
		return nil, bizerrors.Internalf(ctx, "k8s.pod.debug", "get", err, "获取 Pod 失败")
	}

	for _, existing := range pod.Spec.EphemeralContainers {
		if existing.Name == ecName {
			ready := ephemeralContainerReady(&pod, ecName)
			return &PodDebugResult{
				EphemeralContainer: ecName,
				Image:              existing.Image,
				Ready:              ready,
				Message:            "临时调试容器已存在，可直接 Exec",
			}, nil
		}
	}

	target := strings.TrimSpace(req.TargetContainer)
	if target == "" && len(pod.Spec.Containers) > 0 {
		target = pod.Spec.Containers[0].Name
	}

	ec := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:                     ecName,
			Image:                    image,
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"sh", "-c", "sleep infinity"},
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}
	if target != "" {
		ec.TargetContainerName = target
	}
	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ec)

	if _, err := k.Client().CoreV1().Pods(ns).UpdateEphemeralContainers(ctx, name, &pod, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg("集群不支持 EphemeralContainers 或 Pod 不存在")
		}
		if apierrors.IsForbidden(err) {
			return nil, constants.ErrForbiddenWithMsg("无权限创建临时调试容器")
		}
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "ephemeralcontainers") {
			return nil, constants.ErrBadRequestWithMsg("集群未启用 EphemeralContainers 特性，无法注入调试容器")
		}
		return nil, bizerrors.Internalf(ctx, "k8s.pod.debug", "update", err, "注入临时调试容器失败")
	}

	ready := waitEphemeralContainerReady(ctx, k.Client().CoreV1().Pods(ns), name, ecName, 45*time.Second)
	msg := "已注入临时调试容器，可对该容器执行 Exec / 文件操作"
	if !ready {
		msg = "已提交临时调试容器，启动中；稍后在容器列表中选择并 Exec"
	}
	return &PodDebugResult{
		EphemeralContainer: ecName,
		Image:              image,
		Ready:              ready,
		Message:            msg,
	}, nil
}

func ephemeralContainerReady(pod *corev1.Pod, name string) bool {
	if pod == nil {
		return false
	}
	for _, st := range pod.Status.EphemeralContainerStatuses {
		if st.Name == name && st.State.Running != nil {
			return true
		}
	}
	return false
}

type ephemeralPodGetter interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error)
}

func waitEphemeralContainerReady(ctx context.Context, getter ephemeralPodGetter, podName, ecName string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		pod, err := getter.Get(ctx, podName, metav1.GetOptions{})
		if err == nil && ephemeralContainerReady(pod, ecName) {
			return true
		}
		time.Sleep(1500 * time.Millisecond)
	}
	return false
}
