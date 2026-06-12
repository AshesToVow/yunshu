package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TopologyQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Kind      string `form:"kind" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type TopologyNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	State      string `json:"state,omitempty"`
	StateLevel string `json:"state_level,omitempty"` // normal | progressing | abnormal
}

type TopologyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

type TopologyGraph struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

func (s *K8sWorkloadService) Topology(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	kind := strings.ToLower(strings.TrimSpace(q.Kind))
	switch kind {
	case "deployment", "deployments":
		return s.topologyDeployment(ctx, q)
	case "statefulset", "statefulsets":
		return s.topologyStatefulSet(ctx, q)
	case "daemonset", "daemonsets":
		return s.topologyDaemonSet(ctx, q)
	case "service", "services":
		return s.topologyService(ctx, q)
	case "ingress", "ingresses":
		return s.topologyIngress(ctx, q)
	default:
		return nil, constants.ErrBadRequestWithMsg("kind 支持 deployment/statefulset/daemonset/service/ingress")
	}
}

func (s *K8sWorkloadService) topologyDeployment(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var dep appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).Get(&dep).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmta3018a66177e)
	}
	b := newGraphBuilder()
	rootID := nodeID("deployment", q.Namespace, dep.Name)
	b.addNode(rootID, dep.Name, "Deployment", fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas))

	var rsList []appsv1.ReplicaSet
	if err := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(q.Namespace).List(&rsList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt38cc1640ac12)
	}
	pods, err := listCorePodsBySelector(ctx, k, q.Namespace, metav1.FormatLabelSelector(dep.Spec.Selector))
	if err != nil {
		return nil, err
	}
	matchedRS := false
	for _, rs := range rsList {
		if !isOwnedBy(&rs, "Deployment", dep.Name) {
			continue
		}
		matchedRS = true
		rsID := nodeID("replicaset", q.Namespace, rs.Name)
		b.addNode(rsID, rs.Name, "ReplicaSet", fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas))
		b.addEdge(rootID, rsID, "owns")
		appendPodsOwnedBy(b, q.Namespace, pods, "ReplicaSet", rs.Name, rsID)
	}
	if !matchedRS {
		appendPodsBySelector(b, rootID, q.Namespace, pods, "owns")
	}
	appendMatchingServices(b, ctx, k, q.Namespace, dep.Spec.Selector, rootID)
	appendIngressForServices(b, ctx, k, q.Namespace, b.nodes)
	return b.build(), nil
}

func (s *K8sWorkloadService) topologyStatefulSet(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var st appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(q.Namespace).Name(q.Name).Get(&st).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt70dba6fa52bd)
	}
	b := newGraphBuilder()
	rootID := nodeID("statefulset", q.Namespace, st.Name)
	b.addNode(rootID, st.Name, "StatefulSet", fmt.Sprintf("%d/%d", st.Status.ReadyReplicas, st.Status.Replicas))
	pods, err := listCorePodsBySelector(ctx, k, q.Namespace, metav1.FormatLabelSelector(st.Spec.Selector))
	if err != nil {
		return nil, err
	}
	appendPodsOwnedBy(b, q.Namespace, pods, "StatefulSet", st.Name, rootID)
	appendMatchingServices(b, ctx, k, q.Namespace, st.Spec.Selector, rootID)
	appendIngressForServices(b, ctx, k, q.Namespace, b.nodes)
	return b.build(), nil
}

func (s *K8sWorkloadService) topologyDaemonSet(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var ds appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(q.Namespace).Name(q.Name).Get(&ds).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt38cc1640ac12)
	}
	b := newGraphBuilder()
	rootID := nodeID("daemonset", q.Namespace, ds.Name)
	b.addNode(rootID, ds.Name, "DaemonSet", fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled))
	pods, err := listCorePodsBySelector(ctx, k, q.Namespace, metav1.FormatLabelSelector(ds.Spec.Selector))
	if err != nil {
		return nil, err
	}
	appendPodsOwnedBy(b, q.Namespace, pods, "DaemonSet", ds.Name, rootID)
	appendMatchingServices(b, ctx, k, q.Namespace, ds.Spec.Selector, rootID)
	appendIngressForServices(b, ctx, k, q.Namespace, b.nodes)
	return b.build(), nil
}

func (s *K8sWorkloadService) topologyService(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var svc corev1.Service
	if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(q.Namespace).Name(q.Name).Get(&svc).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt38cc1640ac12)
	}
	b := newGraphBuilder()
	sid := nodeID("service", q.Namespace, svc.Name)
	b.addNode(sid, svc.Name, "Service", string(svc.Spec.Type))
	appendIngressPointingToService(b, ctx, k, q.Namespace, svc.Name, sid)
	appendWorkloadsBehindService(ctx, k, b, q.Namespace, svc, sid)
	return b.build(), nil
}

func (s *K8sWorkloadService) topologyIngress(ctx context.Context, q TopologyQuery) (*TopologyGraph, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var ing networkingv1.Ingress
	if err := k.WithContext(ctx).Resource(&networkingv1.Ingress{}).Namespace(q.Namespace).Name(q.Name).Get(&ing).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt38cc1640ac12)
	}
	b := newGraphBuilder()
	iid := nodeID("ingress", q.Namespace, ing.Name)
	state := "rules:" + fmt.Sprintf("%d", len(ing.Spec.Rules))
	b.addNode(iid, ing.Name, "Ingress", state)
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.Backend.Service == nil {
				continue
			}
			sname := p.Backend.Service.Name
			if sname == "" {
				continue
			}
			sid := nodeID("service", q.Namespace, sname)
			b.addNode(sid, sname, "Service", "")
			b.addEdge(iid, sid, "routes")
			var svc corev1.Service
			if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(q.Namespace).Name(sname).Get(&svc).Error; err == nil {
				if svc.Spec.Type != "" {
					b.addNode(sid, svc.Name, "Service", string(svc.Spec.Type))
				}
				appendWorkloadsBehindService(ctx, k, b, q.Namespace, svc, sid)
			}
		}
	}
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
		sname := ing.Spec.DefaultBackend.Service.Name
		if sname != "" {
			sid := nodeID("service", q.Namespace, sname)
			b.addNode(sid, sname, "Service", "")
			b.addEdge(iid, sid, "routes")
			var svc corev1.Service
			if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(q.Namespace).Name(sname).Get(&svc).Error; err == nil {
				appendWorkloadsBehindService(ctx, k, b, q.Namespace, svc, sid)
			}
		}
	}
	return b.build(), nil
}

func appendPodsOwnedBy(b *graphBuilder, ns string, pods []corev1.Pod, ownerKind, ownerName, parentID string) {
	for _, p := range pods {
		if !isOwnedBy(&p, ownerKind, ownerName) {
			continue
		}
		pid := nodeID("pod", ns, p.Name)
		b.addNode(pid, p.Name, "Pod", string(p.Status.Phase))
		b.addEdge(parentID, pid, "owns")
	}
}

func appendPodsBySelector(b *graphBuilder, parentID, ns string, pods []corev1.Pod, edgeKind string) {
	for _, p := range pods {
		pid := nodeID("pod", ns, p.Name)
		b.addNode(pid, p.Name, "Pod", string(p.Status.Phase))
		b.addEdge(parentID, pid, edgeKind)
	}
}

func appendMatchingServices(b *graphBuilder, ctx context.Context, k *kom.Kubectl, ns string, sel *metav1.LabelSelector, rootID string) {
	var svcs []corev1.Service
	if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(ns).List(&svcs).Error; err != nil {
		return
	}
	for _, svc := range svcs {
		if !serviceMatchesWorkloadSelector(svc, sel) {
			continue
		}
		sid := nodeID("service", ns, svc.Name)
		b.addNode(sid, svc.Name, "Service", string(svc.Spec.Type))
		b.addEdge(sid, rootID, "backend")
	}
}

func appendWorkloadsBehindService(ctx context.Context, k *kom.Kubectl, b *graphBuilder, ns string, svc corev1.Service, sid string) {
	selector := labelsToSelector(svc.Spec.Selector)
	if selector == "" {
		return
	}
	pods, err := listCorePodsBySelector(ctx, k, ns, selector)
	if err != nil {
		return
	}

	var deps []appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(ns).List(&deps).Error; err == nil {
		for _, dep := range deps {
			if !serviceMatchesWorkloadSelector(svc, dep.Spec.Selector) {
				continue
			}
			did := nodeID("deployment", ns, dep.Name)
			b.addNode(did, dep.Name, "Deployment", fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas))
			b.addEdge(sid, did, "backend")
			appendDeploymentReplicaSetChain(b, ctx, k, ns, dep, pods, did)
		}
	}

	var sts []appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(ns).List(&sts).Error; err == nil {
		for _, st := range sts {
			if !serviceMatchesWorkloadSelector(svc, st.Spec.Selector) {
				continue
			}
			stid := nodeID("statefulset", ns, st.Name)
			b.addNode(stid, st.Name, "StatefulSet", fmt.Sprintf("%d/%d", st.Status.ReadyReplicas, st.Status.Replicas))
			b.addEdge(sid, stid, "backend")
			appendPodsOwnedBy(b, ns, pods, "StatefulSet", st.Name, stid)
		}
	}
}

func appendDeploymentReplicaSetChain(b *graphBuilder, ctx context.Context, k *kom.Kubectl, ns string, dep appsv1.Deployment, pods []corev1.Pod, did string) {
	var rsList []appsv1.ReplicaSet
	if err := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(ns).List(&rsList).Error; err != nil {
		return
	}
	matched := false
	for _, rs := range rsList {
		if !isOwnedBy(&rs, "Deployment", dep.Name) {
			continue
		}
		matched = true
		rsID := nodeID("replicaset", ns, rs.Name)
		b.addNode(rsID, rs.Name, "ReplicaSet", fmt.Sprintf("%d/%d", rs.Status.ReadyReplicas, rs.Status.Replicas))
		b.addEdge(did, rsID, "owns")
		appendPodsOwnedBy(b, ns, pods, "ReplicaSet", rs.Name, rsID)
	}
	if !matched {
		appendPodsBySelector(b, did, ns, pods, "owns")
	}
}

func appendIngressForServices(b *graphBuilder, ctx context.Context, k *kom.Kubectl, ns string, nodes []TopologyNode) {
	for _, n := range nodes {
		if n.Kind != "Service" {
			continue
		}
		parts := strings.SplitN(n.ID, "/", 3)
		if len(parts) < 3 {
			continue
		}
		appendIngressPointingToService(b, ctx, k, ns, parts[2], n.ID)
	}
}

func appendIngressPointingToService(b *graphBuilder, ctx context.Context, k *kom.Kubectl, ns, svcName, sid string) {
	var ings []networkingv1.Ingress
	if err := k.WithContext(ctx).Resource(&networkingv1.Ingress{}).Namespace(ns).List(&ings).Error; err != nil {
		return
	}
	for _, ing := range ings {
		if !ingressReferencesService(ing, svcName) {
			continue
		}
		iid := nodeID("ingress", ns, ing.Name)
		state := "rules:" + fmt.Sprintf("%d", len(ing.Spec.Rules))
		b.addNode(iid, ing.Name, "Ingress", state)
		b.addEdge(iid, sid, "routes")
	}
}

func ingressReferencesService(ing networkingv1.Ingress, svcName string) bool {
	if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil && ing.Spec.DefaultBackend.Service.Name == svcName {
		return true
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.Backend.Service != nil && p.Backend.Service.Name == svcName {
				return true
			}
		}
	}
	return false
}

func labelsToSelector(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}

func nodeID(kind, ns, name string) string {
	return strings.ToLower(kind) + "/" + ns + "/" + name
}

// serviceMatchesWorkloadSelector 判断 Service 是否路由到该 Workload 的 Pod（Service selector 为 Workload 标签子集）。
func serviceMatchesWorkloadSelector(svc corev1.Service, sel *metav1.LabelSelector) bool {
	if sel == nil || len(sel.MatchLabels) == 0 {
		return false
	}
	if len(svc.Spec.Selector) == 0 {
		return false
	}
	for k, v := range svc.Spec.Selector {
		if sel.MatchLabels[k] != v {
			return false
		}
	}
	return true
}
