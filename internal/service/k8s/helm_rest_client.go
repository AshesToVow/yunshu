package k8s

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// helmRESTClientGetter 将 yunshu 集群 rest.Config 适配为 Helm action 所需的 RESTClientGetter。
type helmRESTClientGetter struct {
	kubeConfig *rest.Config
}

var _ genericclioptions.RESTClientGetter = (*helmRESTClientGetter)(nil)

func newHelmRESTClientGetter(kubeConfig *rest.Config) *helmRESTClientGetter {
	return &helmRESTClientGetter{kubeConfig: kubeConfig}
}

func (h *helmRESTClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	cfg := rest.CopyConfig(h.kubeConfig)
	cfg.Burst = 100
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(discoveryClient), nil
}

func (h *helmRESTClientGetter) ToRESTConfig() (*rest.Config, error) {
	return h.kubeConfig, nil
}

func (h *helmRESTClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	discoveryClient, err := h.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryClient)
	return restmapper.NewShortcutExpander(mapper, discoveryClient, func(string) {}), nil
}

func (h *helmRESTClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
}
