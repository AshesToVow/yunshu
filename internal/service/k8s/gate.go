package k8s

import (
	"context"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/service/changegate"
)

func projectIDOf(cluster *model.K8sCluster) uint {
	if cluster == nil || cluster.OwningProjectID == nil {
		return 0
	}
	return *cluster.OwningProjectID
}

func assertK8sWritable(ctx context.Context, cluster *model.K8sCluster, action, namespace string) error {
	pid := projectIDOf(cluster)
	if pid == 0 {
		return nil
	}
	return changegate.AssertWritable(ctx, changegate.CheckInput{
		ProjectID: pid,
		Source:    model.ChangeSourceK8s,
		Namespace: strings.TrimSpace(namespace),
		Action:    action,
	})
}
