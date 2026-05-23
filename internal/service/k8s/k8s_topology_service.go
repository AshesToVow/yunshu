package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TopologyQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Kind      string `form:"kind" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type TopologyNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
	State string `json:"state,omitempty"`
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
	default:
		return nil, constants.ErrBadRequestWithMsg("kind 仅支持 deployment")
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
	rootID := nodeID("deployment", q.Namespace, dep.Name)
	nodes := []TopologyNode{{ID: rootID, Label: dep.Name, Kind: "Deployment", State: fmt.Sprintf("%d/%d", dep.Status.ReadyReplicas, dep.Status.Replicas)}}
	edges := []TopologyEdge{}

	selector := metav1.FormatLabelSelector(dep.Spec.Selector)
	pods, err := listPodsBySelector(ctx, k, q.Namespace, selector)
	if err != nil {
		return nil, err
	}
	for _, p := range pods {
		pid := nodeID("pod", q.Namespace, p.Name)
		nodes = append(nodes, TopologyNode{ID: pid, Label: p.Name, Kind: "Pod", State: p.Phase})
		edges = append(edges, TopologyEdge{From: rootID, To: pid, Kind: "owns"})
	}

	var svcs []corev1.Service
	if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(q.Namespace).List(&svcs).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.topology", "api", err, constants.ErrFmt38cc1640ac12)
	}
	for _, svc := range svcs {
		if !serviceSelectsPods(svc, dep.Spec.Selector) {
			continue
		}
		sid := nodeID("service", q.Namespace, svc.Name)
		nodes = append(nodes, TopologyNode{ID: sid, Label: svc.Name, Kind: "Service", State: string(svc.Spec.Type)})
		edges = append(edges, TopologyEdge{From: rootID, To: sid, Kind: "selects"})
		for _, p := range pods {
			edges = append(edges, TopologyEdge{From: sid, To: nodeID("pod", q.Namespace, p.Name), Kind: "routes"})
		}
	}
	return &TopologyGraph{Nodes: nodes, Edges: edges}, nil
}

func nodeID(kind, ns, name string) string {
	return strings.ToLower(kind) + "/" + ns + "/" + name
}

func serviceSelectsPods(svc corev1.Service, sel *metav1.LabelSelector) bool {
	if sel == nil || len(sel.MatchLabels) == 0 {
		return false
	}
	if len(svc.Spec.Selector) == 0 {
		return false
	}
	for k, v := range sel.MatchLabels {
		if svc.Spec.Selector[k] != v {
			return false
		}
	}
	return true
}
