package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RolloutUndoRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Revision  int64  `json:"revision"` // 0 = 上一版
}

type RolloutUndoResult struct {
	Kind         string `json:"kind"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	FromRevision string `json:"from_revision,omitempty"`
	ToRevision   string `json:"to_revision,omitempty"`
	Message      string `json:"message"`
}

// DeploymentRolloutUndo 回滚 Deployment 到指定或上一 ReplicaSet revision。
func (s *K8sWorkloadService) DeploymentRolloutUndo(ctx context.Context, req RolloutUndoRequest) (*RolloutUndoResult, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	var dep appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(ns).Name(name).Get(&dep).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "get", err, "获取 Deployment 失败")
	}
	fromRev := dep.Annotations["deployment.kubernetes.io/revision"]
	selector := metav1.FormatLabelSelector(dep.Spec.Selector)
	var rsList []appsv1.ReplicaSet
	q := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(ns)
	if selector != "" {
		q = q.WithLabelSelector(selector)
	}
	if err := q.List(&rsList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "list_rs", err, "列出 ReplicaSet 失败")
	}
	targetRS, err := pickReplicaSetForUndo(&dep, rsList, req.Revision)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	toRev := targetRS.Annotations["deployment.kubernetes.io/revision"]
	copyObj := dep.DeepCopy()
	copyObj.Spec.Template = targetRS.Spec.Template
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(ns).Name(name).Update(copyObj).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "update", err, "回滚 Deployment 失败")
	}
	return &RolloutUndoResult{
		Kind:         "Deployment",
		Namespace:    ns,
		Name:         name,
		FromRevision: fromRev,
		ToRevision:   toRev,
		Message:      "Deployment 已回滚",
	}, nil
}

// StatefulSetRolloutUndo 基于 ControllerRevision 回滚 StatefulSet（更新 pod template 注解触发滚动并记录目标版本）。
func (s *K8sWorkloadService) StatefulSetRolloutUndo(ctx context.Context, req RolloutUndoRequest) (*RolloutUndoResult, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	name := strings.TrimSpace(req.Name)
	var sts appsv1.StatefulSet
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(ns).Name(name).Get(&sts).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "get", err, "获取 StatefulSet 失败")
	}
	fromRev := sts.Status.CurrentRevision
	var crList []appsv1.ControllerRevision
	if err := k.WithContext(ctx).Resource(&appsv1.ControllerRevision{}).Namespace(ns).List(&crList).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "list_cr", err, "列出 ControllerRevision 失败")
	}
	owned := filterOwnedControllerRevisions(crList, &sts)
	target, err := pickControllerRevision(owned, req.Revision)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	toRev := target.Name
	copyObj := sts.DeepCopy()
	if copyObj.Spec.Template.Annotations == nil {
		copyObj.Spec.Template.Annotations = map[string]string{}
	} else {
		copyObj.Spec.Template.Annotations = copyStrMap(copyObj.Spec.Template.Annotations)
	}
	copyObj.Spec.Template.Annotations["yunshu.io/rollback-to"] = toRev
	copyObj.Spec.Template.Annotations["yunshu.io/rollback-at"] = metav1.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(ns).Name(name).Update(copyObj).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.workload.undo", "update", err, "回滚 StatefulSet 失败")
	}
	return &RolloutUndoResult{
		Kind:         "StatefulSet",
		Namespace:    ns,
		Name:         name,
		FromRevision: fromRev,
		ToRevision:   toRev,
		Message:      "已标记 StatefulSet 回滚目标（ControllerRevision），请确认滚动状态",
	}, nil
}

// WorkloadReady 供 CI/CD verify 注入。
func (s *K8sWorkloadService) WorkloadReady(ctx context.Context, clusterIDStr, namespace, kind, name string) (*bool, string) {
	id, err := strconv.ParseUint(strings.TrimSpace(clusterIDStr), 10, 32)
	if err != nil || id == 0 {
		f := false
		return &f, "无效 cluster_id"
	}
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "deployment", "deployments":
		st, err := s.DeploymentRolloutStatus(ctx, DeploymentRolloutQuery{
			ClusterID: uint(id),
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			f := false
			return &f, err.Error()
		}
		ok := st.Complete && st.AvailableReplicas >= st.Replicas && st.Replicas > 0
		msg := fmt.Sprintf("ready=%d/%d complete=%v", st.ReadyReplicas, st.Replicas, st.Complete)
		return &ok, msg
	case "statefulset", "statefulsets":
		_, k, err := s.runtime.GetClusterKubectl(ctx, uint(id))
		if err != nil {
			f := false
			return &f, err.Error()
		}
		var sts appsv1.StatefulSet
		if err := k.WithContext(ctx).Resource(&appsv1.StatefulSet{}).Namespace(namespace).Name(name).Get(&sts).Error; err != nil {
			f := false
			return &f, err.Error()
		}
		want := int32(1)
		if sts.Spec.Replicas != nil {
			want = *sts.Spec.Replicas
		}
		ok := sts.Status.ReadyReplicas >= want && want > 0
		msg := fmt.Sprintf("ready=%d/%d", sts.Status.ReadyReplicas, want)
		return &ok, msg
	default:
		f := false
		return &f, "不支持的 kind: " + kind
	}
}

func pickReplicaSetForUndo(dep *appsv1.Deployment, items []appsv1.ReplicaSet, revision int64) (*appsv1.ReplicaSet, error) {
	owned := make([]appsv1.ReplicaSet, 0)
	for i := range items {
		rs := items[i]
		if !metav1.IsControlledBy(&rs, dep) {
			continue
		}
		owned = append(owned, rs)
	}
	if len(owned) == 0 {
		return nil, fmt.Errorf("未找到 Deployment 关联的 ReplicaSet")
	}
	if revision > 0 {
		want := strconv.FormatInt(revision, 10)
		for i := range owned {
			if owned[i].Annotations["deployment.kubernetes.io/revision"] == want {
				return &owned[i], nil
			}
		}
		return nil, fmt.Errorf("未找到 revision=%d 的 ReplicaSet", revision)
	}
	sortReplicaSetsByRevisionDesc(owned)
	cur := dep.Annotations["deployment.kubernetes.io/revision"]
	for i := range owned {
		if owned[i].Annotations["deployment.kubernetes.io/revision"] != cur {
			return &owned[i], nil
		}
	}
	if len(owned) >= 2 {
		return &owned[1], nil
	}
	return nil, fmt.Errorf("没有可回滚的历史 ReplicaSet")
}

func sortReplicaSetsByRevisionDesc(items []appsv1.ReplicaSet) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			ri, _ := strconv.ParseInt(items[i].Annotations["deployment.kubernetes.io/revision"], 10, 64)
			rj, _ := strconv.ParseInt(items[j].Annotations["deployment.kubernetes.io/revision"], 10, 64)
			if rj > ri {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func filterOwnedControllerRevisions(items []appsv1.ControllerRevision, sts *appsv1.StatefulSet) []appsv1.ControllerRevision {
	out := make([]appsv1.ControllerRevision, 0)
	for i := range items {
		if metav1.IsControlledBy(&items[i], sts) {
			out = append(out, items[i])
		}
	}
	return out
}

func pickControllerRevision(items []appsv1.ControllerRevision, revision int64) (*appsv1.ControllerRevision, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("无 ControllerRevision")
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Revision > items[i].Revision {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if revision > 0 {
		for i := range items {
			if items[i].Revision == revision {
				return &items[i], nil
			}
		}
		return nil, fmt.Errorf("未找到 ControllerRevision=%d", revision)
	}
	if len(items) < 2 {
		return nil, fmt.Errorf("没有可回滚的历史版本")
	}
	return &items[1], nil
}

func copyStrMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
