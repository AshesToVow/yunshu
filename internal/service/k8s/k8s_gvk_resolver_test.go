package k8s

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func apiRes(group, version, kind, name string, namespaced bool) *metav1.APIResource {
	return &metav1.APIResource{
		Group:      group,
		Version:    version,
		Kind:       kind,
		Name:       name,
		Namespaced: namespaced,
	}
}

var (
	hpaPreferred          = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}
	ingressPreferred      = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}
	ingressClassPreferred = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"}
)

// TestResolveFromAPIResourcesVersionOrder 集群同时提供多版本时应取候选表中最靠前（最新）的版本。
func TestResolveFromAPIResourcesVersionOrder(t *testing.T) {
	tests := []struct {
		name      string
		resources []*metav1.APIResource
		preferred schema.GroupVersionKind
		wantGVK   schema.GroupVersionKind
		wantGVR   schema.GroupVersionResource
		wantNSed  bool
	}{
		{
			name: "HPA 多版本共存取 v2",
			resources: []*metav1.APIResource{
				apiRes("autoscaling", "v1", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
				apiRes("autoscaling", "v2beta2", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
				apiRes("autoscaling", "v2", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
			},
			preferred: hpaPreferred,
			wantGVK:   schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"},
			wantGVR:   schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
			wantNSed:  true,
		},
		{
			name: "HPA 仅有 v2beta1 时降级",
			resources: []*metav1.APIResource{
				apiRes("autoscaling", "v1", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
				apiRes("autoscaling", "v2beta1", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
			},
			preferred: hpaPreferred,
			wantGVK:   schema.GroupVersionKind{Group: "autoscaling", Version: "v2beta1", Kind: "HorizontalPodAutoscaler"},
			wantGVR:   schema.GroupVersionResource{Group: "autoscaling", Version: "v2beta1", Resource: "horizontalpodautoscalers"},
			wantNSed:  true,
		},
		{
			name: "HPA 老集群仅有 v1",
			resources: []*metav1.APIResource{
				apiRes("autoscaling", "v1", "HorizontalPodAutoscaler", "horizontalpodautoscalers", true),
			},
			preferred: hpaPreferred,
			wantGVK:   schema.GroupVersionKind{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"},
			wantGVR:   schema.GroupVersionResource{Group: "autoscaling", Version: "v1", Resource: "horizontalpodautoscalers"},
			wantNSed:  true,
		},
		{
			name: "Ingress 跨 Group 兜底 extensions/v1beta1",
			resources: []*metav1.APIResource{
				apiRes("extensions", "v1beta1", "Ingress", "ingresses", true),
			},
			preferred: ingressPreferred,
			wantGVK:   schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "Ingress"},
			wantGVR:   schema.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "ingresses"},
			wantNSed:  true,
		},
		{
			name: "IngressClass 为集群级",
			resources: []*metav1.APIResource{
				apiRes("networking.k8s.io", "v1", "IngressClass", "ingressclasses", false),
			},
			preferred: ingressClassPreferred,
			wantGVK:   schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"},
			wantGVR:   schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingressclasses"},
			wantNSed:  false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveFromAPIResources(tc.resources, tc.preferred)
			if err != nil {
				t.Fatalf("resolveFromAPIResources() 意外报错: %v", err)
			}
			if got.GVK != tc.wantGVK {
				t.Fatalf("GVK = %v, want %v", got.GVK, tc.wantGVK)
			}
			if got.GVR != tc.wantGVR {
				t.Fatalf("GVR = %v, want %v", got.GVR, tc.wantGVR)
			}
			if got.Namespaced != tc.wantNSed {
				t.Fatalf("Namespaced = %v, want %v", got.Namespaced, tc.wantNSed)
			}
			if !got.Discovered {
				t.Fatal("Discovered 应为 true")
			}
		})
	}
}

// TestResolveFromAPIResourcesEmptyDiscovery discovery 未就绪时原样透传首选 GVK，不报错。
func TestResolveFromAPIResourcesEmptyDiscovery(t *testing.T) {
	got, err := resolveFromAPIResources(nil, hpaPreferred)
	if err != nil {
		t.Fatalf("空 discovery 不应报错: %v", err)
	}
	if got.GVK != hpaPreferred {
		t.Fatalf("GVK = %v, want %v", got.GVK, hpaPreferred)
	}
	if got.Discovered {
		t.Fatal("Discovered 应为 false")
	}
	if !got.Namespaced {
		t.Fatal("HPA 应为命名空间级")
	}
	if got.GVR != (schema.GroupVersionResource{}) {
		t.Fatalf("GVR 应为空，实际 %v", got.GVR)
	}

	clsCap, err := resolveFromAPIResources(nil, ingressClassPreferred)
	if err != nil {
		t.Fatalf("空 discovery 不应报错: %v", err)
	}
	if clsCap.Namespaced {
		t.Fatal("IngressClass 应按兜底表判定为集群级")
	}
}

// TestResolveFromAPIResourcesUnsupported discovery 已就绪但所有候选版本都不存在时给出明确错误。
func TestResolveFromAPIResourcesUnsupported(t *testing.T) {
	resources := []*metav1.APIResource{
		apiRes("", "v1", "Pod", "pods", true),
	}
	_, err := resolveFromAPIResources(resources, hpaPreferred)
	if err == nil {
		t.Fatal("集群不提供 HPA 时应报错")
	}
	msg := err.Error()
	if !strings.Contains(msg, "HorizontalPodAutoscaler") {
		t.Fatalf("错误信息应包含 Kind，实际: %s", msg)
	}
	for _, want := range []string{"autoscaling/v2", "autoscaling/v2beta2", "autoscaling/v2beta1", "autoscaling/v1"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("错误信息应列出已尝试版本 %s，实际: %s", want, msg)
		}
	}
}

// TestResolveFromAPIResourcesEmptyKind Kind 为空属参数错误。
func TestResolveFromAPIResourcesEmptyKind(t *testing.T) {
	if _, err := resolveFromAPIResources(nil, schema.GroupVersionKind{Group: "apps", Version: "v1"}); err == nil {
		t.Fatal("Kind 为空时应报错")
	}
}

// TestNamespacedFromAPIResources Group/Version 精确匹配优先，其次同 Kind 兜底，最后落到静态表。
func TestNamespacedFromAPIResources(t *testing.T) {
	resources := []*metav1.APIResource{
		apiRes("networking.k8s.io", "v1", "Ingress", "ingresses", true),
		apiRes("networking.k8s.io", "v1", "IngressClass", "ingressclasses", false),
		apiRes("example.com", "v1alpha1", "Widget", "widgets", true),
	}
	tests := []struct {
		name string
		gvk  schema.GroupVersionKind
		want bool
	}{
		{"Group+Version 精确匹配命名空间级", ingressPreferred, true},
		{"Group+Version 精确匹配集群级", ingressClassPreferred, false},
		{"Version 为空时只比 Group", schema.GroupVersionKind{Group: "networking.k8s.io", Kind: "IngressClass"}, false},
		{"Group 不匹配时按同 Kind 兜底", schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "Ingress"}, true},
		{"CRD 按 discovery 判定", schema.GroupVersionKind{Group: "example.com", Version: "v1alpha1", Kind: "Widget"}, true},
		{"未收录 Kind 落静态兜底表-集群级", schema.GroupVersionKind{Version: "v1", Kind: "Node"}, false},
		{"未收录 Kind 落静态兜底表-命名空间级", schema.GroupVersionKind{Version: "v1", Kind: "Pod"}, true},
		{"Kind 为空按命名空间级处理", schema.GroupVersionKind{Group: "apps", Version: "v1"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := namespacedFromAPIResources(resources, tc.gvk); got != tc.want {
				t.Fatalf("namespacedFromAPIResources(%v) = %v, want %v", tc.gvk, got, tc.want)
			}
		})
	}
	if namespacedFromAPIResources(nil, schema.GroupVersionKind{Version: "v1", Kind: "Node"}) {
		t.Fatal("空 discovery 下 Node 应判为集群级")
	}
}

// TestResourceExistsInAPIResources 用于区分「组件未安装」与「数据为空」。
func TestResourceExistsInAPIResources(t *testing.T) {
	resources := []*metav1.APIResource{
		apiRes("metrics.k8s.io", "v1beta1", "PodMetrics", "pods", true),
	}
	if !resourceExistsInAPIResources(resources, podMetricsGVK) {
		t.Fatal("已安装 metrics-server 时应判为存在")
	}
	if resourceExistsInAPIResources(resources, hpaPreferred) {
		t.Fatal("集群未提供 HPA 时应判为不存在")
	}
	if !resourceExistsInAPIResources(nil, podMetricsGVK) {
		t.Fatal("空 discovery 应返回 true 以避免误判组件缺失")
	}
}

// TestHasVersionCandidates 仅登记在降级表中的 Kind 才做严格校验，CRD 不受影响。
func TestHasVersionCandidates(t *testing.T) {
	tests := []struct {
		gvk  schema.GroupVersionKind
		want bool
	}{
		{hpaPreferred, true},
		{ingressPreferred, true},
		{ingressClassPreferred, true},
		{schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}, true},
		{schema.GroupVersionKind{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}, true},
		{schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"}, true},
		{schema.GroupVersionKind{Group: "example.com", Version: "v1alpha1", Kind: "Widget"}, false},
		{schema.GroupVersionKind{Version: "v1", Kind: "Pod"}, false},
		{schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, false},
		{podMetricsGVK, false},
	}
	for _, tc := range tests {
		if got := hasVersionCandidates(tc.gvk); got != tc.want {
			t.Fatalf("hasVersionCandidates(%v) = %v, want %v", tc.gvk, got, tc.want)
		}
	}
}

// TestCandidateGroupVersions 未登记的 Kind 只用自身 GroupVersion 作为唯一候选。
func TestCandidateGroupVersions(t *testing.T) {
	got := candidateGroupVersions(hpaPreferred)
	want := []schema.GroupVersion{
		{Group: "autoscaling", Version: "v2"},
		{Group: "autoscaling", Version: "v2beta2"},
		{Group: "autoscaling", Version: "v2beta1"},
		{Group: "autoscaling", Version: "v1"},
	}
	if len(got) != len(want) {
		t.Fatalf("候选数量 = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("候选[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	crd := schema.GroupVersionKind{Group: "example.com", Version: "v1alpha1", Kind: "Widget"}
	single := candidateGroupVersions(crd)
	if len(single) != 1 || single[0] != crd.GroupVersion() {
		t.Fatalf("未登记 Kind 候选 = %v, want [%v]", single, crd.GroupVersion())
	}
}

// TestResolveGVKForRequestPassthrough 未登记 Kind 直接透传，避免 CRD discovery 延迟导致误报。
func TestResolveGVKForRequestPassthrough(t *testing.T) {
	crd := schema.GroupVersionKind{Group: "example.com", Version: "v1alpha1", Kind: "Widget"}
	got, err := resolveGVKForRequest(nil, crd)
	if err != nil {
		t.Fatalf("未登记 Kind 不应报错: %v", err)
	}
	if got != crd {
		t.Fatalf("GVK = %v, want %v", got, crd)
	}
}

// TestResolveClusterResourceNilKubectl kom 实例为空时按空 discovery 处理，不得误报「集群不支持」。
func TestResolveClusterResourceNilKubectl(t *testing.T) {
	got, err := resolveClusterResource(nil, hpaPreferred)
	if err != nil {
		t.Fatalf("nil Kubectl 不应报错: %v", err)
	}
	if got.GVK != hpaPreferred || got.Discovered {
		t.Fatalf("capability = %+v, want 首选 GVK 且 Discovered=false", got)
	}
	if _, err := resolveGVKForCluster(nil, hpaPreferred); err != nil {
		t.Fatalf("resolveGVKForCluster(nil) 不应报错: %v", err)
	}
	if !clusterResourceExists(nil, hpaPreferred) {
		t.Fatal("nil Kubectl 时不得判为资源缺失")
	}
	if !namespacedForGVK(nil, hpaPreferred) {
		t.Fatal("nil Kubectl 时 HPA 应按兜底表判为命名空间级")
	}
}
