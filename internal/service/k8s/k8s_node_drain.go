package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sauth"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// NodeDrainRequest kubectl drain 语义：先 cordon，再逐个 Evict Pod。
type NodeDrainRequest struct {
	ClusterID          uint   `json:"cluster_id" binding:"required"`
	Name               string `json:"name" binding:"required"`
	GracePeriodSeconds *int64 `json:"grace_period_seconds"`
	// IgnoreDaemonSets 默认 true，跳过 DaemonSet 托管的 Pod。
	IgnoreDaemonSets *bool `json:"ignore_daemon_sets"`
	// DeleteEmptyDirData 默认 true；为 false 时跳过含 emptyDir 的 Pod。
	DeleteEmptyDirData *bool `json:"delete_emptydir_data"`
	// Force 对无控制器的裸 Pod 也执行 Evict。
	Force bool `json:"force"`
	// DryRun 仅列出将驱逐的 Pod，不 cordon / 不 Evict。
	DryRun bool `json:"dry_run"`
	// Confirm 高危确认（非 dry_run 时必填，与集群 RequireDestructiveConfirm 联动）。
	Confirm bool `json:"confirm"`
}

type NodeDrainPodItem struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Phase     string `json:"phase"`
	OwnerKind string `json:"owner_kind,omitempty"`
	Action    string `json:"action"` // evicted|skipped|failed|pending
	Reason    string `json:"reason,omitempty"`
}

type NodeDrainResult struct {
	NodeName    string             `json:"node_name"`
	Cordoned    bool               `json:"cordoned"`
	DryRun      bool               `json:"dry_run"`
	Evicted     int                `json:"evicted"`
	Skipped     int                `json:"skipped"`
	Failed      int                `json:"failed"`
	Pending     int                `json:"pending"`
	Pods        []NodeDrainPodItem `json:"pods"`
	Message     string             `json:"message"`
	CompletedAt string             `json:"completed_at"`
}

type NodeDrainStatusQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type NodeDrainStatus struct {
	NodeName      string             `json:"node_name"`
	Unschedulable bool               `json:"unschedulable"`
	Remaining     int                `json:"remaining"`
	DaemonSet     int                `json:"daemonset_pods"`
	Pods          []NodeDrainPodItem `json:"pods"`
	Drained       bool               `json:"drained"`
	Message       string             `json:"message"`
}

// DrainNode 驱逐节点上的工作负载（近似 kubectl drain）。
func (s *K8sNodeService) DrainNode(ctx context.Context, req NodeDrainRequest) (*NodeDrainResult, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg215d21a8863c)
	}
	if req.Confirm {
		ctx = k8sauth.WithDestructiveConfirm(ctx, true)
	}
	ignoreDS := true
	if req.IgnoreDaemonSets != nil {
		ignoreDS = *req.IgnoreDaemonSets
	}
	deleteEmptyDir := true
	if req.DeleteEmptyDirData != nil {
		deleteEmptyDir = *req.DeleteEmptyDirData
	}

	cluster, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	if err := assertK8sWritable(ctx, cluster, "drain", ""); err != nil {
		return nil, err
	}
	if !req.DryRun {
		if err := RequireDestructiveConfirm(ctx, cluster); err != nil {
			return nil, err
		}
	}
	var node corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Name(name).Get(&node).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg7b4519294b96)
		}
		return nil, k8sFail(ctx, "k8s.node", "api", err)
	}

	cs, err := s.nodeClientset(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	pods, err := listPodsOnNode(ctx, cs, name)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.node", "list_pods", err, "列出节点 Pod 失败")
	}

	out := &NodeDrainResult{
		NodeName: name,
		DryRun:   req.DryRun,
		Pods:     []NodeDrainPodItem{},
	}

	if !req.DryRun && !node.Spec.Unschedulable {
		updated := node.DeepCopy()
		updated.Spec.Unschedulable = true
		if err := k.WithContext(ctx).Resource(&corev1.Node{}).Update(updated).Error; err != nil {
			return nil, k8sFail(ctx, "k8s.node", "cordon", err)
		}
		out.Cordoned = true
	} else {
		out.Cordoned = node.Spec.Unschedulable
	}

	evictionGV := evictionPolicyGroupVersion(cs)
	for _, p := range pods {
		item, skip := classifyDrainPod(p, ignoreDS, deleteEmptyDir, req.Force)
		if skip {
			out.Skipped++
			out.Pods = append(out.Pods, item)
			continue
		}
		if req.DryRun {
			item.Action = "pending"
			item.Reason = "dry-run"
			out.Pending++
			out.Pods = append(out.Pods, item)
			continue
		}
		if err := evictPod(ctx, cs, p, req.GracePeriodSeconds, evictionGV); err != nil {
			item.Action = "failed"
			item.Reason = err.Error()
			out.Failed++
			out.Pods = append(out.Pods, item)
			continue
		}
		item.Action = "evicted"
		out.Evicted++
		out.Pods = append(out.Pods, item)
	}

	sort.Slice(out.Pods, func(i, j int) bool {
		if out.Pods[i].Namespace == out.Pods[j].Namespace {
			return out.Pods[i].Name < out.Pods[j].Name
		}
		return out.Pods[i].Namespace < out.Pods[j].Namespace
	})
	out.CompletedAt = time.Now().Format(time.RFC3339)
	if req.DryRun {
		out.Message = fmt.Sprintf("预览：将驱逐 %d，跳过 %d", out.Pending, out.Skipped)
	} else {
		out.Message = fmt.Sprintf("已 cordon；驱逐 %d，跳过 %d，失败 %d", out.Evicted, out.Skipped, out.Failed)
	}
	return out, nil
}

// DrainStatus 查询节点剩余可驱逐 Pod（用于进度浮窗）。
func (s *K8sNodeService) DrainStatus(ctx context.Context, q NodeDrainStatusQuery) (*NodeDrainStatus, error) {
	name := strings.TrimSpace(q.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg215d21a8863c)
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var node corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Name(name).Get(&node).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg7b4519294b96)
		}
		return nil, k8sFail(ctx, "k8s.node", "api", err)
	}
	cs, err := s.nodeClientset(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	pods, err := listPodsOnNode(ctx, cs, name)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.node", "list_pods", err, "列出节点 Pod 失败")
	}
	out := &NodeDrainStatus{
		NodeName:      name,
		Unschedulable: node.Spec.Unschedulable,
		Pods:          []NodeDrainPodItem{},
	}
	ignoreDS := true
	deleteEmptyDir := true
	for _, p := range pods {
		item, skip := classifyDrainPod(p, ignoreDS, deleteEmptyDir, false)
		if item.OwnerKind == "DaemonSet" {
			out.DaemonSet++
		}
		if skip {
			continue
		}
		item.Action = "pending"
		out.Remaining++
		out.Pods = append(out.Pods, item)
	}
	out.Drained = out.Remaining == 0
	if out.Drained {
		out.Message = "节点上无可驱逐工作负载"
	} else {
		out.Message = fmt.Sprintf("剩余 %d 个可驱逐 Pod（DaemonSet %d 个已忽略）", out.Remaining, out.DaemonSet)
	}
	return out, nil
}

func (s *K8sNodeService) nodeClientset(ctx context.Context, clusterID uint) (*kubernetes.Clientset, error) {
	_, cfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.node", "clientset", err, "创建集群客户端失败")
	}
	return cs, nil
}

func listPodsOnNode(ctx context.Context, cs *kubernetes.Clientset, nodeName string) ([]corev1.Pod, error) {
	list, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func classifyDrainPod(p corev1.Pod, ignoreDS, deleteEmptyDir, force bool) (NodeDrainPodItem, bool) {
	item := NodeDrainPodItem{
		Namespace: p.Namespace,
		Name:      p.Name,
		Phase:     string(p.Status.Phase),
		OwnerKind: primaryOwnerKind(p),
	}
	if p.Namespace == "kube-system" && isMirrorOrStatic(p) {
		item.Action = "skipped"
		item.Reason = "mirror/static pod"
		return item, true
	}
	if ignoreDS && item.OwnerKind == "DaemonSet" {
		item.Action = "skipped"
		item.Reason = "DaemonSet"
		return item, true
	}
	if !deleteEmptyDir && hasEmptyDir(p) {
		item.Action = "skipped"
		item.Reason = "emptyDir"
		return item, true
	}
	if item.OwnerKind == "" && !force {
		item.Action = "skipped"
		item.Reason = "裸 Pod，需 force=true"
		return item, true
	}
	if p.DeletionTimestamp != nil {
		item.Action = "skipped"
		item.Reason = "正在删除"
		return item, true
	}
	return item, false
}

func primaryOwnerKind(p corev1.Pod) string {
	for _, ref := range p.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			return ref.Kind
		}
	}
	if len(p.OwnerReferences) > 0 {
		return p.OwnerReferences[0].Kind
	}
	return ""
}

func isMirrorOrStatic(p corev1.Pod) bool {
	if _, ok := p.Annotations["kubernetes.io/config.mirror"]; ok {
		return true
	}
	if src, ok := p.Annotations["kubernetes.io/config.source"]; ok && src == "file" {
		return true
	}
	return false
}

func hasEmptyDir(p corev1.Pod) bool {
	for _, v := range p.Spec.Volumes {
		if v.EmptyDir != nil {
			return true
		}
	}
	return false
}

// evictionPolicyGroupVersion 参照 kubectl drain 的 CheckEvictionSupport：
// 从 core/v1 discovery 中读取 pods/eviction 子资源上声明的 Group/Version，
// 判定集群提供的是 policy/v1 还是 policy/v1beta1。
// 注意：不能用 kom 缓存的 APIResources —— kom initializeAPIResources 会把
// 每个 APIResource 的 Group/Version 覆写为所属 APIResourceList 的 GroupVersion（core/v1），
// 子资源上的 policy/v1 标注会因此丢失。
func evictionPolicyGroupVersion(cs *kubernetes.Clientset) schema.GroupVersion {
	fallback := schema.GroupVersion{Group: "policy", Version: "v1beta1"}
	if cs == nil {
		return policyv1GroupVersion
	}
	list, err := cs.Discovery().ServerResourcesForGroupVersion("v1")
	if err != nil || list == nil {
		// discovery 不可用时按新版本尝试，失败后由 evictPod 兜底降级。
		return policyv1GroupVersion
	}
	for i := range list.APIResources {
		r := list.APIResources[i]
		if r.Name != "pods/eviction" {
			continue
		}
		gv := schema.GroupVersion{Group: strings.TrimSpace(r.Group), Version: strings.TrimSpace(r.Version)}
		if gv.Version == "" {
			return policyv1GroupVersion
		}
		if gv.Group == "" {
			gv.Group = "policy"
		}
		return gv
	}
	return fallback
}

var policyv1GroupVersion = schema.GroupVersion{Group: "policy", Version: "v1"}

// evictPod 按集群支持的 Eviction 版本驱逐 Pod；policy/v1 不可用时降级 policy/v1beta1。
func evictPod(ctx context.Context, cs *kubernetes.Clientset, p corev1.Pod, grace *int64, evictionGV schema.GroupVersion) error {
	deleteOptions := &metav1.DeleteOptions{}
	if grace != nil {
		deleteOptions.GracePeriodSeconds = grace
	}
	meta := metav1.ObjectMeta{Name: p.Name, Namespace: p.Namespace}

	evictV1 := func() error {
		return cs.CoreV1().Pods(p.Namespace).EvictV1(ctx, &policyv1.Eviction{
			ObjectMeta:    meta,
			DeleteOptions: deleteOptions,
		})
	}
	evictV1beta1 := func() error {
		return cs.CoreV1().Pods(p.Namespace).EvictV1beta1(ctx, &policyv1beta1.Eviction{
			ObjectMeta:    meta,
			DeleteOptions: deleteOptions,
		})
	}

	var err error
	if evictionGV.Version == "v1beta1" {
		err = evictV1beta1()
	} else {
		err = evictV1()
		// 集群未提供 policy/v1 Eviction（404 / 405）时降级到 v1beta1；
		// Pod 本身不存在也会是 404，但降级请求同样会返回 404，最终按「已消失」处理。
		if err != nil && (apierrors.IsNotFound(err) || apierrors.IsMethodNotSupported(err)) {
			if beErr := evictV1beta1(); beErr == nil {
				return nil
			} else if !apierrors.IsNotFound(beErr) && !apierrors.IsMethodNotSupported(beErr) {
				err = beErr
			}
		}
	}
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		// Pod 已消失，视为驱逐成功
		return nil
	}
	return fmt.Errorf("驱逐 %s/%s: %w", p.Namespace, p.Name, err)
}
