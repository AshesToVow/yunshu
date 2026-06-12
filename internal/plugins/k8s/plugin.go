package k8s

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/logutil"
	"yunshu/internal/plugin"
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

func (m *module) Models() []any {
	return []any{
		&model.K8sCluster{},
		&model.K8sNamespaceDenyRule{},
		&model.K8sNamespaceAllowRule{},
		&model.K8sClusterAccessGrant{},
		&model.K8sForwardedEvent{},
		&model.K8sEventForwardRule{},
		&model.K8sEventForwardSetting{},
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
	runtimeSvc := rt.K8sRuntimeSvc()
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
		logutil.Worker("k8s.event_forward").Errorw(err, "Failed to init K8s event forward manager")
		return nil
	}
	mgr.Start()
	eventforward.SetActive(mgr)
	return nil
}
