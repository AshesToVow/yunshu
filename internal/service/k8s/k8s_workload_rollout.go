package k8s

import (
	"context"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type DeploymentRolloutQuery = NamespacedDetailQuery

type DeploymentRolloutCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type DeploymentRolloutStatus struct {
	Namespace           string                       `json:"namespace"`
	Name                string                       `json:"name"`
	Generation          int64                        `json:"generation"`
	ObservedGeneration  int64                        `json:"observed_generation"`
	Synced              bool                         `json:"synced"`
	Replicas            int32                        `json:"replicas"`
	UpdatedReplicas     int32                        `json:"updated_replicas"`
	ReadyReplicas       int32                        `json:"ready_replicas"`
	AvailableReplicas   int32                        `json:"available_replicas"`
	UnavailableReplicas int32                        `json:"unavailable_replicas"`
	StrategyType        string                       `json:"strategy_type"`
	MaxSurge            string                       `json:"max_surge,omitempty"`
	MaxUnavailable      string                       `json:"max_unavailable,omitempty"`
	MinReadySeconds     int32                        `json:"min_ready_seconds"`
	ProgressDeadline    int32                        `json:"progress_deadline_seconds"`
	Complete            bool                         `json:"complete"`
	Progressing         bool                         `json:"progressing"`
	Conditions          []DeploymentRolloutCondition `json:"conditions"`
}

// DeploymentRolloutStatus 返回 Deployment 滚动发布进度，便于更新过程中确认业务仍可访问。
func (s *K8sWorkloadService) DeploymentRolloutStatus(ctx context.Context, q DeploymentRolloutQuery) (*DeploymentRolloutStatus, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var dep appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(q.Namespace).Name(q.Name).Get(&dep).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		return nil, bizerrors.Internalf(ctx, "k8s.workload.rollout", "api", err, constants.ErrFmta3018a66177e)
	}
	st := dep.Status
	spec := dep.Spec
	out := &DeploymentRolloutStatus{
		Namespace:           dep.Namespace,
		Name:                dep.Name,
		Generation:          dep.Generation,
		ObservedGeneration:  st.ObservedGeneration,
		Synced:              st.ObservedGeneration >= dep.Generation,
		Replicas:            derefInt32(spec.Replicas),
		UpdatedReplicas:     st.UpdatedReplicas,
		ReadyReplicas:       st.ReadyReplicas,
		AvailableReplicas:   st.AvailableReplicas,
		UnavailableReplicas: st.UnavailableReplicas,
		StrategyType:        string(spec.Strategy.Type),
		MinReadySeconds:     spec.MinReadySeconds,
		ProgressDeadline:    derefInt32(spec.ProgressDeadlineSeconds),
	}
	if spec.Strategy.RollingUpdate != nil {
		if spec.Strategy.RollingUpdate.MaxSurge != nil {
			out.MaxSurge = spec.Strategy.RollingUpdate.MaxSurge.String()
		}
		if spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			out.MaxUnavailable = spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
	}
	for _, c := range st.Conditions {
		out.Conditions = append(out.Conditions, DeploymentRolloutCondition{
			Type:    string(c.Type),
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
		if c.Type == appsv1.DeploymentProgressing && c.Status == "True" {
			out.Progressing = true
		}
	}
	desired := derefInt32(spec.Replicas)
	if out.StrategyType == "" {
		out.StrategyType = string(appsv1.RollingUpdateDeploymentStrategyType)
	}
	// 必须等控制器观察到最新 spec（observedGeneration >= metadata.generation），
	// 否则刚改完镜像时旧副本仍全 Ready 会被误判为发布完成。
	out.Complete = deploymentRolloutComplete(out.StrategyType, out.Synced,
		st.Replicas, st.UpdatedReplicas, st.ReadyReplicas, st.AvailableReplicas, st.UnavailableReplicas, desired)
	return out, nil
}

func deploymentRolloutComplete(strategyType string, synced bool, replicas, updated, ready, available, unavailable, desired int32) bool {
	if !synced || replicas != updated {
		return false
	}
	if strings.EqualFold(strategyType, "Recreate") {
		return ready >= desired && available >= desired
	}
	return updated >= desired && ready >= desired && available >= desired && unavailable == 0
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}
