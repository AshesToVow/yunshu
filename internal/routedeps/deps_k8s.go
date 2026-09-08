package routedeps

import (
	"yunshu/internal/handler"
)

type K8sRouteDeps interface {
	RouteMiddleware
	K8sScopedPolicyHandler() *handler.K8sScopedPolicyHandler
	K8sNamespaceDenyHandler() *handler.K8sNamespaceDenyHandler
	K8sNamespaceAllowHandler() *handler.K8sNamespaceAllowHandler
	ClusterHandler() *handler.ClusterHandler
	K8sDiscoveryHandler() *handler.K8sDiscoveryHandler
	K8sEventForwardHandler() *handler.K8sEventForwardHandler
	PodHandler() *handler.PodHandler
	NamespaceHandler() *handler.NamespaceHandler
	NodeHandler() *handler.NodeHandler
	WorkloadHandler() *handler.WorkloadHandler
	ConfigHandler() *handler.ConfigHandler
	StorageHandler() *handler.StorageHandler
	ServiceResourceHandler() *handler.ServiceResourceHandler
	IngressHandler() *handler.IngressHandler
	NetworkPolicyHandler() *handler.NetworkPolicyHandler
	K8sHPAHandler() *handler.K8sHPAHandler
	HelmHandler() *handler.HelmHandler
	EventHandler() *handler.EventHandler
	CRDHandler() *handler.CRDHandler
	CRHandler() *handler.CRHandler
	RBACHandler() *handler.RBACHandler
	ServiceAccountHandler() *handler.ServiceAccountHandler
	OverviewHandler() *handler.OverviewHandler
	K8sSearchHandler() *handler.K8sSearchHandler
	K8sResourceWatchHandler() *handler.K8sResourceWatchHandler
	PlatformFeaturesHandler() *handler.PlatformFeaturesHandler
}
