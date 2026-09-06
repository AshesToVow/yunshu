package k8s

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	bizerrors "yunshu/internal/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentRevisionItem struct {
	Revision  int64      `json:"revision"`
	Replicas  int32      `json:"replicas"`
	Ready     int32      `json:"ready"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	Current   bool       `json:"current"`
}

type DeploymentRevisionQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

// ListDeploymentRevisions 列出 Deployment 历史 ReplicaSet revision，供回滚助手选择。
func (s *K8sWorkloadService) ListDeploymentRevisions(ctx context.Context, q DeploymentRevisionQuery) ([]DeploymentRevisionItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	name := strings.TrimSpace(q.Name)
	var dep appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(ns).Name(name).Get(&dep).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "get", err, "获取 Deployment 失败")
	}
	curRev := strings.TrimSpace(dep.Annotations["deployment.kubernetes.io/revision"])
	selector := metav1.FormatLabelSelector(dep.Spec.Selector)
	var rsList []appsv1.ReplicaSet
	qry := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(ns)
	if selector != "" {
		qry = qry.WithLabelSelector(selector)
	}
	if err := qry.List(&rsList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "list_rs", err, "列出 ReplicaSet 失败")
	}
	out := make([]DeploymentRevisionItem, 0)
	for i := range rsList {
		rs := rsList[i]
		if !metav1.IsControlledBy(&rs, &dep) {
			continue
		}
		revStr := strings.TrimSpace(rs.Annotations["deployment.kubernetes.io/revision"])
		rev, _ := strconv.ParseInt(revStr, 10, 64)
		if rev <= 0 {
			continue
		}
		var created *time.Time
		if t := rs.CreationTimestamp; !t.IsZero() {
			tt := t.Time
			created = &tt
		}
		out = append(out, DeploymentRevisionItem{
			Revision:  rev,
			Replicas:  rs.Status.Replicas,
			Ready:     rs.Status.ReadyReplicas,
			CreatedAt: created,
			Current:   revStr == curRev,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision > out[j].Revision })
	return out, nil
}

// ListStatefulSetRevisions 列出 StatefulSet ControllerRevision 历史。
func (s *K8sWorkloadService) ListStatefulSetRevisions(ctx context.Context, q DeploymentRevisionQuery) ([]DeploymentRevisionItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	name := strings.TrimSpace(q.Name)
	var sts appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(ns).Name(name).Get(&sts).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "get", err, "获取 StatefulSet 失败")
	}
	var crList []appsv1.ControllerRevision
	if err := k.WithContext(ctx).Resource(&appsv1.ControllerRevision{}).Namespace(ns).List(&crList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "list_cr", err, "列出 ControllerRevision 失败")
	}
	owned := filterOwnedControllerRevisions(crList, &sts)
	curName := strings.TrimSpace(sts.Status.CurrentRevision)
	var maxRev int64
	for i := range owned {
		if owned[i].Revision > maxRev {
			maxRev = owned[i].Revision
		}
	}
	out := make([]DeploymentRevisionItem, 0, len(owned))
	for i := range owned {
		cr := owned[i]
		var created *time.Time
		if t := cr.CreationTimestamp; !t.IsZero() {
			tt := t.Time
			created = &tt
		}
		current := false
		if curName != "" {
			current = cr.Name == curName
		} else {
			current = cr.Revision == maxRev
		}
		out = append(out, DeploymentRevisionItem{
			Revision:  cr.Revision,
			CreatedAt: created,
			Current:   current,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision > out[j].Revision })
	return out, nil
}

// ListDaemonSetRevisions 列出 DaemonSet ControllerRevision 历史。
func (s *K8sWorkloadService) ListDaemonSetRevisions(ctx context.Context, q DeploymentRevisionQuery) ([]DeploymentRevisionItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(q.Namespace)
	name := strings.TrimSpace(q.Name)
	var ds appsv1.DaemonSet
	if err := k.WithContext(ctx).Resource(&appsv1.DaemonSet{}).Namespace(ns).Name(name).Get(&ds).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "get", err, "获取 DaemonSet 失败")
	}
	var crList []appsv1.ControllerRevision
	if err := k.WithContext(ctx).Resource(&appsv1.ControllerRevision{}).Namespace(ns).List(&crList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.revisions", "list_cr", err, "列出 ControllerRevision 失败")
	}
	owned := filterOwnedDaemonSetControllerRevisions(crList, &ds)
	curRev := ds.Status.CurrentNumberScheduled // placeholder; mark highest as current by name match
	out := make([]DeploymentRevisionItem, 0, len(owned))
	_ = curRev
	var maxRev int64
	for i := range owned {
		if owned[i].Revision > maxRev {
			maxRev = owned[i].Revision
		}
	}
	for i := range owned {
		cr := owned[i]
		var created *time.Time
		if t := cr.CreationTimestamp; !t.IsZero() {
			tt := t.Time
			created = &tt
		}
		out = append(out, DeploymentRevisionItem{
			Revision:  cr.Revision,
			CreatedAt: created,
			Current:   cr.Revision == maxRev,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Revision > out[j].Revision })
	return out, nil
}

func filterOwnedDaemonSetControllerRevisions(items []appsv1.ControllerRevision, ds *appsv1.DaemonSet) []appsv1.ControllerRevision {
	out := make([]appsv1.ControllerRevision, 0)
	for i := range items {
		if metav1.IsControlledBy(&items[i], ds) {
			out = append(out, items[i])
		}
	}
	return out
}
