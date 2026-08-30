package k8s

// 跨版本 GVK 解析：以集群 discovery（kom 已缓存的 Status().APIResources()）为准，
// 按「新版本优先」的候选顺序挑选集群真实提供的 Group/Version，
// 避免 kom Tools().GetGVRByKind 在同 Kind 多版本时返回任意一个。
//
// 不再叠加一层本地缓存：kom 在 ClusterInst.apiResources 上已做缓存，
// 并在 CRD 变更时刷新；再缓存只会引入「装了 metrics-server 仍报不支持」这类过期问题。

import (
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"

	kom "github.com/weibaohui/kom/kom"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// resourceCapability 集群针对某 Kind 实际可用的 GVK/GVR 与作用域。
type resourceCapability struct {
	GVK        schema.GroupVersionKind
	GVR        schema.GroupVersionResource
	Namespaced bool
	// Discovered 为 false 表示 discovery 缓存尚未就绪，字段取自调用方声明的首选 GVK。
	Discovered bool
}

// gvkVersionCandidates 各 Kind 的降级顺序（新 → 旧）。
// key 为调用方声明的首选 GroupKind；value 允许跨 Group（如 Ingress 的 extensions 兜底）。
var gvkVersionCandidates = map[schema.GroupKind][]schema.GroupVersion{
	{Group: "autoscaling", Kind: "HorizontalPodAutoscaler"}: {
		{Group: "autoscaling", Version: "v2"},
		{Group: "autoscaling", Version: "v2beta2"},
		{Group: "autoscaling", Version: "v2beta1"},
		{Group: "autoscaling", Version: "v1"},
	},
	{Group: "policy", Kind: "Eviction"}: {
		{Group: "policy", Version: "v1"},
		{Group: "policy", Version: "v1beta1"},
	},
	{Group: "networking.k8s.io", Kind: "Ingress"}: {
		{Group: "networking.k8s.io", Version: "v1"},
		{Group: "networking.k8s.io", Version: "v1beta1"},
		{Group: "extensions", Version: "v1beta1"},
	},
	{Group: "networking.k8s.io", Kind: "IngressClass"}: {
		{Group: "networking.k8s.io", Version: "v1"},
		{Group: "networking.k8s.io", Version: "v1beta1"},
	},
	{Group: "networking.k8s.io", Kind: "NetworkPolicy"}: {
		{Group: "networking.k8s.io", Version: "v1"},
		{Group: "extensions", Version: "v1beta1"},
	},
	{Group: "policy", Kind: "PodDisruptionBudget"}: {
		{Group: "policy", Version: "v1"},
		{Group: "policy", Version: "v1beta1"},
	},
	{Group: "batch", Kind: "CronJob"}: {
		{Group: "batch", Version: "v1"},
		{Group: "batch", Version: "v1beta1"},
	},
}

// candidateGroupVersions 返回 preferred 的候选 GroupVersion 列表；无预置表时仅用自身。
func candidateGroupVersions(preferred schema.GroupVersionKind) []schema.GroupVersion {
	key := schema.GroupKind{Group: strings.TrimSpace(preferred.Group), Kind: strings.TrimSpace(preferred.Kind)}
	if list, ok := gvkVersionCandidates[key]; ok && len(list) > 0 {
		return list
	}
	return []schema.GroupVersion{preferred.GroupVersion()}
}

func clusterAPIResources(k *kom.Kubectl) []*metav1.APIResource {
	if k == nil {
		return nil
	}
	return k.Status().APIResources()
}

// resolveClusterResource 在集群 discovery 中按候选顺序定位 Kind；
// discovery 未就绪时返回首选 GVK（Discovered=false），由 apiserver 自行报错；
// discovery 就绪但所有候选都不存在时返回明确的「集群不支持该资源」错误。
func resolveClusterResource(k *kom.Kubectl, preferred schema.GroupVersionKind) (resourceCapability, error) {
	return resolveFromAPIResources(clusterAPIResources(k), preferred)
}

// resolveFromAPIResources 解析核心（纯函数，便于单测）。
func resolveFromAPIResources(resources []*metav1.APIResource, preferred schema.GroupVersionKind) (resourceCapability, error) {
	kind := strings.TrimSpace(preferred.Kind)
	if kind == "" {
		return resourceCapability{}, constants.ErrBadRequestWithMsg("资源 Kind 不能为空")
	}
	if len(resources) == 0 {
		return resourceCapability{
			GVK:        preferred,
			Namespaced: !gvkClusterScoped(preferred),
		}, nil
	}
	candidates := candidateGroupVersions(preferred)
	for _, gv := range candidates {
		for _, r := range resources {
			if r == nil || r.Kind != kind {
				continue
			}
			if r.Group != gv.Group || r.Version != gv.Version {
				continue
			}
			return resourceCapability{
				GVK:        gv.WithKind(kind),
				GVR:        gv.WithResource(r.Name),
				Namespaced: r.Namespaced,
				Discovered: true,
			}, nil
		}
	}
	return resourceCapability{}, unsupportedResourceErr(kind, candidates)
}

func unsupportedResourceErr(kind string, candidates []schema.GroupVersion) error {
	tried := make([]string, 0, len(candidates))
	for _, gv := range candidates {
		tried = append(tried, gv.String())
	}
	return constants.ErrBadRequestWithMsg(fmt.Sprintf(
		"当前集群不提供资源 %s（已尝试 %s），请确认集群版本或相关组件是否已安装",
		kind, strings.Join(tried, ", ")))
}

// resolveGVKForCluster 仅需要 GVK 的调用方（List/Get 走 unstructured）使用。
func resolveGVKForCluster(k *kom.Kubectl, preferred schema.GroupVersionKind) (schema.GroupVersionKind, error) {
	capability, err := resolveClusterResource(k, preferred)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return capability.GVK, nil
}

// clusterResourceExists 判断集群是否提供该 Kind（用于区分「组件未安装」与「数据为空」）。
// discovery 未就绪时返回 true，避免把首次访问误判为组件缺失。
func clusterResourceExists(k *kom.Kubectl, gvk schema.GroupVersionKind) bool {
	return resourceExistsInAPIResources(clusterAPIResources(k), gvk)
}

// resourceExistsInAPIResources 存在性判定核心（纯函数，便于单测）。
func resourceExistsInAPIResources(resources []*metav1.APIResource, gvk schema.GroupVersionKind) bool {
	if len(resources) == 0 {
		return true
	}
	kind := strings.TrimSpace(gvk.Kind)
	group := strings.TrimSpace(gvk.Group)
	for _, r := range resources {
		if r == nil || r.Kind != kind {
			continue
		}
		if r.Group == group {
			return true
		}
	}
	return false
}

// hasVersionCandidates 判断 Kind 是否登记了跨版本降级表。
// 未登记的 Kind（含 CRD）不做严格校验：kom 的 APIResources 缓存对新建 CRD 有秒级延迟，
// 严格校验会把「刚创建的 CRD」误判为集群不支持。
func hasVersionCandidates(preferred schema.GroupVersionKind) bool {
	key := schema.GroupKind{Group: strings.TrimSpace(preferred.Group), Kind: strings.TrimSpace(preferred.Kind)}
	_, ok := gvkVersionCandidates[key]
	return ok
}

// resolveGVKForRequest 读写请求前的 GVK 归一：
// 仅对登记了降级表的 Kind 做严格解析，其余原样返回，保持既有行为。
func resolveGVKForRequest(k *kom.Kubectl, preferred schema.GroupVersionKind) (schema.GroupVersionKind, error) {
	if !hasVersionCandidates(preferred) {
		return preferred, nil
	}
	return resolveGVKForCluster(k, preferred)
}

// namespacedForGVK 以 discovery 的 Namespaced 字段判定作用域，取代硬编码 Kind 列表。
// discovery 未就绪或 Kind 未收录时退回 gvkClusterScoped 兜底表。
func namespacedForGVK(k *kom.Kubectl, gvk schema.GroupVersionKind) bool {
	return namespacedFromAPIResources(clusterAPIResources(k), gvk)
}

// namespacedFromAPIResources 作用域判定核心（纯函数，便于单测）。
func namespacedFromAPIResources(resources []*metav1.APIResource, gvk schema.GroupVersionKind) bool {
	kind := strings.TrimSpace(gvk.Kind)
	if kind == "" {
		return true
	}
	group := strings.TrimSpace(gvk.Group)
	version := strings.TrimSpace(gvk.Version)
	var kindOnly *metav1.APIResource
	for _, r := range resources {
		if r == nil || r.Kind != kind {
			continue
		}
		if r.Group == group && (version == "" || r.Version == version) {
			return r.Namespaced
		}
		if kindOnly == nil {
			kindOnly = r
		}
	}
	if kindOnly != nil {
		return kindOnly.Namespaced
	}
	return !gvkClusterScoped(gvk)
}
