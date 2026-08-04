package k8s

import (
	"context"

	"yunshu/internal/model"
	"log/slog"
	"yunshu/internal/plugin"
	"yunshu/internal/service"
	"yunshu/internal/service/k8s/eventforward"

	"gorm.io/gorm"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "k8s" }
func (m *module) Description() string { return "Kubernetes 多集群管理与资源控制台" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{
			"/clusters", "/cluster", "/pods", "/namespaces", "/nodes", "/component-status",
			"/cluster-api-resources", "/horizontal-pod-autoscalers", "/k8s-resource-topology",
			"/deployments", "/statefulsets", "/daemonsets", "/cronjobs", "/jobs",
			"/helm/releases", "/helm/charts",
			"/configmaps", "/secrets", "/ingresses", "/ingress-classes", "/events",
			"/k8s-services", "/persistentvolumes", "/persistentvolumeclaims", "/storageclasses",
			"/crds", "/crs", "/rbac", "/serviceaccounts", "/k8s-scoped-policies",
			"/network-policies", "/k8s/",
		},
		APIPrefixes: []string{
			"/api/v1/clusters", "/api/v1/pods", "/api/v1/namespaces", "/api/v1/nodes",
			"/api/v1/deployments", "/api/v1/statefulsets", "/api/v1/daemonsets", "/api/v1/cronjobs",
			"/api/v1/jobs", "/api/v1/configmaps", "/api/v1/secrets", "/api/v1/services",
			"/api/v1/ingresses", "/api/v1/ingress-classes", "/api/v1/events", "/api/v1/persistentvolumes",
			"/api/v1/persistentvolumeclaims", "/api/v1/storageclasses", "/api/v1/crds", "/api/v1/crs",
			"/api/v1/rbac", "/api/v1/serviceaccounts", "/api/v1/k8s-policies",
			"/api/v1/k8s-namespace-deny-rules", "/api/v1/k8s-namespace-allow-rules",
			"/api/v1/network-policies", "/api/v1/k8s/", "/api/v1/horizontal-pod-autoscalers",
			"/api/v1/helm/", "/api/v1/component-status", "/api/v1/k8s-event-forward",
		},
		Workers: []string{"k8s_event_forward"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.K8sCluster{},
		&model.K8sNamespaceDenyRule{},
		&model.K8sNamespaceAllowRule{},
		&model.K8sClusterAccessGrant{},
		&model.K8sForwardedEvent{},
		&model.K8sEventForwardRule{},
		&model.K8sEventForwardSetting{},
		&model.K8sWorkloadSnapshot{},
	}
}

func (m *module) PostMigrate(db *gorm.DB) error {
	if err := migrateK8sLegacyRoleCodeToPrincipal(db); err != nil {
		return err
	}
	return migrateDropLegacyK8sCasbinPolicies(db)
}

func (m *module) StartWorkers(bgCtx context.Context, rt *plugin.Runtime) error {
	if bgCtx == nil || rt == nil || rt.DB == nil || rt.Config == nil {
		return nil
	}
	runtimeSvc, _ := rt.K8sRuntime.(*service.K8sRuntimeService)
	if runtimeSvc == nil {
		return nil
	}
	mgr, err := eventforward.NewManager(
		nil,
		runtimeSvc,
		rt.YamlK8sEventForwardBase,
		rt.Config.Alert,
		rt.Config.App.Port,
		rt.DB,
	)
	if err != nil {
		slog.Default().With("component", "k8s.event_forward").Error("Failed to init K8s event forward manager", "error", err)
		return nil
	}
	mgr.Start()
	eventforward.SetActive(mgr)
	return nil
}
