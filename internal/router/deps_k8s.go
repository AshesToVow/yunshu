package router

import (
	"yunshu/internal/handler"
	"yunshu/internal/routedeps"
)

// K8sRouteDeps 插件/模块路由窄依赖。

func (d *RouteDeps) K8sScopedPolicyHandler() *handler.K8sScopedPolicyHandler {
	if d == nil {
		return nil
	}
	return d.k8sScopedPolicyHandler
}

func (d *RouteDeps) K8sNamespaceDenyHandler() *handler.K8sNamespaceDenyHandler {
	if d == nil {
		return nil
	}
	return d.k8sNamespaceDenyHandler
}

func (d *RouteDeps) K8sNamespaceAllowHandler() *handler.K8sNamespaceAllowHandler {
	if d == nil {
		return nil
	}
	return d.k8sNamespaceAllowHandler
}

func (d *RouteDeps) ClusterHandler() *handler.ClusterHandler {
	if d == nil {
		return nil
	}
	return d.clusterHandler
}

func (d *RouteDeps) K8sDiscoveryHandler() *handler.K8sDiscoveryHandler {
	if d == nil {
		return nil
	}
	return d.k8sDiscoveryHandler
}

func (d *RouteDeps) K8sEventForwardHandler() *handler.K8sEventForwardHandler {
	if d == nil {
		return nil
	}
	return d.k8sEventForwardHandler
}

func (d *RouteDeps) PodHandler() *handler.PodHandler {
	if d == nil {
		return nil
	}
	return d.podHandler
}

func (d *RouteDeps) NamespaceHandler() *handler.NamespaceHandler {
	if d == nil {
		return nil
	}
	return d.namespaceHandler
}

func (d *RouteDeps) NodeHandler() *handler.NodeHandler {
	if d == nil {
		return nil
	}
	return d.nodeHandler
}

func (d *RouteDeps) WorkloadHandler() *handler.WorkloadHandler {
	if d == nil {
		return nil
	}
	return d.workloadHandler
}

func (d *RouteDeps) ConfigHandler() *handler.ConfigHandler {
	if d == nil {
		return nil
	}
	return d.configHandler
}

func (d *RouteDeps) StorageHandler() *handler.StorageHandler {
	if d == nil {
		return nil
	}
	return d.storageHandler
}

func (d *RouteDeps) ServiceResourceHandler() *handler.ServiceResourceHandler {
	if d == nil {
		return nil
	}
	return d.serviceResourceHandler
}

func (d *RouteDeps) IngressHandler() *handler.IngressHandler {
	if d == nil {
		return nil
	}
	return d.ingressHandler
}

func (d *RouteDeps) NetworkPolicyHandler() *handler.NetworkPolicyHandler {
	if d == nil {
		return nil
	}
	return d.networkPolicyHandler
}

func (d *RouteDeps) K8sHPAHandler() *handler.K8sHPAHandler {
	if d == nil {
		return nil
	}
	return d.k8sHPAHandler
}

func (d *RouteDeps) HelmHandler() *handler.HelmHandler {
	if d == nil {
		return nil
	}
	return d.helmHandler
}

func (d *RouteDeps) EventHandler() *handler.EventHandler {
	if d == nil {
		return nil
	}
	return d.eventHandler
}

func (d *RouteDeps) CRDHandler() *handler.CRDHandler {
	if d == nil {
		return nil
	}
	return d.crdHandler
}

func (d *RouteDeps) CRHandler() *handler.CRHandler {
	if d == nil {
		return nil
	}
	return d.crHandler
}

func (d *RouteDeps) RBACHandler() *handler.RBACHandler {
	if d == nil {
		return nil
	}
	return d.rbacHandler
}

func (d *RouteDeps) ServiceAccountHandler() *handler.ServiceAccountHandler {
	if d == nil {
		return nil
	}
	return d.serviceAccountHandler
}

func (d *RouteDeps) OverviewHandler() *handler.OverviewHandler {
	if d == nil {
		return nil
	}
	return d.overviewHandler
}

func (d *RouteDeps) K8sSearchHandler() *handler.K8sSearchHandler {
	if d == nil {
		return nil
	}
	return d.k8sSearchHandler
}

func (d *RouteDeps) K8sResourceWatchHandler() *handler.K8sResourceWatchHandler {
	if d == nil {
		return nil
	}
	return d.k8sResourceWatchHandler
}

var _ K8sRouteDeps = (*RouteDeps)(nil)

var _ routedeps.K8sRouteDeps = (*RouteDeps)(nil)
