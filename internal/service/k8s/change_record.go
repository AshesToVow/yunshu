package k8s

import (
	"context"
	"fmt"

	"yunshu/internal/model"
	"yunshu/internal/service/changeevent"
)

func recordK8sChange(ctx context.Context, cluster *model.K8sCluster, action, kind, namespace, name string, payload any) {
	if cluster == nil || cluster.OwningProjectID == nil || *cluster.OwningProjectID == 0 {
		return
	}
	summary := fmt.Sprintf("K8s %s %s/%s/%s", action, kind, namespace, name)
	changeevent.Record(ctx, changeevent.Input{
		ProjectID: *cluster.OwningProjectID,
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
