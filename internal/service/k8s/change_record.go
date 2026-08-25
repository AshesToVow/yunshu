package k8s

import (
	"context"
	"fmt"

	"yunshu/internal/model"
	"yunshu/internal/service/changeevent"
)

func recordK8sChange(ctx context.Context, cluster *model.K8sCluster, action, kind, namespace, name string, payload any) {
	if cluster == nil {
		return
	}
	projectID := uint(0)
	if cluster.OwningProjectID != nil {
		projectID = *cluster.OwningProjectID
	}
	summary := fmt.Sprintf("K8s %s %s/%s/%s", action, kind, namespace, name)
	changeevent.Record(ctx, changeevent.Input{
		ProjectID: projectID,
		Source:    model.ChangeSourceK8s,
		Action:    action,
		RiskLevel: model.ChangeRiskHigh,
		Status:    model.ChangeStatusSucceeded,
		Summary:   summary,
		Payload: map[string]any{
			"cluster_id": cluster.ID,
			"kind":       kind,
			"namespace":  namespace,
			"name":       name,
			"detail":     payload,
		},
	})
}
