package k8s

import (
	"context"

	"yunshu/internal/model"
)

func projectIDOf(cluster *model.K8sCluster) uint {
	if cluster == nil || cluster.OwningProjectID == nil {
		return 0
	}
	return *cluster.OwningProjectID
}

func assertK8sWritable(_ context.Context, _ *model.K8sCluster, _, _ string) error {
	return nil
}
