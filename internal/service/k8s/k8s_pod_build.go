package k8s

// Pod 结构体构建与展示映射：请求 DTO → corev1.Pod、corev1.Pod → PodItem，
// 以及「Pod 由工作负载管理」的编辑提示。

import (
	"context"
	"fmt"
	"time"

	"yunshu/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func mapPodItem(p corev1.Pod, usage podCPUMemUsage, alloc nodeAllocResources) PodItem {
	var restart int32
	images := make([]string, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		images = append(images, c.Image)
	}
	readyCnt := 0
	for _, st := range p.Status.ContainerStatuses {
		restart += st.RestartCount
		if st.Ready {
			readyCnt++
		}
	}
	ready := len(p.Status.ContainerStatuses) > 0 && readyCnt == len(p.Status.ContainerStatuses)
	startTime := time.Time{}
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Time
	}
	reqCPU, reqMem, limCPU, limMem := podResourceTotals(p)
	item := PodItem{
		Name:            p.Name,
		Namespace:       p.Namespace,
		Phase:           string(p.Status.Phase),
		NodeName:        p.Spec.NodeName,
		Ready:           ready,
		PodIP:           p.Status.PodIP,
		HostIP:          p.Status.HostIP,
		QOSClass:        string(p.Status.QOSClass),
		RestartCount:    restart,
		Images:          images,
		StartTime:       startTime,
		HostNetwork:     p.Spec.HostNetwork,
		ContainersText:  podContainersImageText(p),
		ResourceText:    podAggregatedResourceText(reqCPU, reqMem, limCPU, limMem),
		LabelCount:      len(p.Labels),
		AnnotationCount: len(p.Annotations),
	}
	if !usage.CPU.IsZero() {
		item.CPUUsage = formatQuantityCPUReadable(usage.CPU)
	} else {
		item.CPUUsage = "-"
	}
	if !usage.Mem.IsZero() {
		item.MemUsage = formatQuantityMemReadable(usage.Mem)
	} else {
		item.MemUsage = "-"
	}
	item.CPUPctRequest = quantityPercent(usage.CPU, reqCPU)
	item.CPUPctLimit = quantityPercent(usage.CPU, limCPU)
	item.CPUPctNodeAlloc = quantityPercent(usage.CPU, alloc.CPU)
	item.MemPctRequest = quantityPercent(usage.Mem, reqMem)
	item.MemPctLimit = quantityPercent(usage.Mem, limMem)
	item.MemPctNodeAlloc = quantityPercent(usage.Mem, alloc.Mem)
	return item
}

func workloadManagedPodHint(ctx context.Context, k *kom.Kubectl, pod *corev1.Pod) string {
	if k == nil || pod == nil {
		return ""
	}
	owner := k8sutil.ControllerOwner(pod.OwnerReferences)
	if owner == nil {
		return ""
	}
	switch owner.Kind {
	case "StatefulSet":
		return "该 Pod 由 StatefulSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 StatefulSet 中修改镜像或配置后滚动更新。"
	case "ReplicaSet":
		// 绝大多数情况 ReplicaSet 来自 Deployment，尽力探测 Deployment 名称
		var rs appsv1.ReplicaSet
		err := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(pod.Namespace).Name(owner.Name).Get(&rs).Error
		if err == nil {
			if depName := k8sutil.RSOwnerDeploymentName(&rs); depName != "" {
				return fmt.Sprintf("该 Pod 由 Deployment(%s) 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 Deployment 中修改镜像或配置后滚动更新。", depName)
			}
		}
		return "该 Pod 由 ReplicaSet（通常来自 Deployment）管理，直接编辑 Pod 可能会被控制器回滚/重建；请到对应工作负载中修改镜像或配置后滚动更新。"
	case "DaemonSet":
		return "该 Pod 由 DaemonSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 DaemonSet 中修改镜像或配置后滚动更新。"
	}
	return ""
}

func buildSimplePod(req PodCreateSimpleRequest) *corev1.Pod {
	tolerations := make([]k8sutil.SimpleTolerationInput, 0, len(req.Tolerations))
	for _, t := range req.Tolerations {
		tolerations = append(tolerations, k8sutil.SimpleTolerationInput{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return k8sutil.BuildSimplePod(k8sutil.SimplePodBuildInput{
		Name:              req.Name,
		Namespace:         req.Namespace,
		Image:             req.Image,
		Command:           req.Command,
		ContainerName:     req.ContainerName,
		ImagePullPolicy:   req.ImagePullPolicy,
		RestartPolicy:     req.RestartPolicy,
		Port:              req.Port,
		Env:               req.Env,
		Labels:            req.Labels,
		RequestsCPU:       req.RequestsCPU,
		RequestsMemory:    req.RequestsMemory,
		LimitsCPU:         req.LimitsCPU,
		LimitsMemory:      req.LimitsMemory,
		Tolerations:       tolerations,
		NodeSelector:      req.NodeSelector,
		PriorityClassName: req.PriorityClassName,
		Affinity:          req.Affinity,
	})
}
