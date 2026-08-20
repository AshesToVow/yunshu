package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/api/meta"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/yaml"
)

// ServerSideDryRunApply 对多文档 YAML 执行 apiserver DryRun=All（不落库）。
func (s *DynamicResourceService) ServerSideDryRunApply(ctx context.Context, clusterID uint, manifest string) error {
	if s.runtime == nil {
		return fmt.Errorf("runtime nil")
	}
	if err := s.ensureManifestNamespacesAllowed(ctx, manifest); err != nil {
		return err
	}
	_, cfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	disco, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(disco))

	for _, doc := range k8sutil.SplitYAMLDocs(manifest) {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		obj := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(doc), &obj.Object); err != nil {
			return fmt.Errorf("解析 YAML 失败: %w", err)
		}
		gvk := obj.GroupVersionKind()
		if gvk.Empty() {
			continue
		}
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("RESTMapping %s: %w", gvk.String(), err)
		}
		var ri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = metav1.NamespaceDefault
			}
			ri = dc.Resource(mapping.Resource).Namespace(ns)
		} else {
			ri = dc.Resource(mapping.Resource)
		}
		name := obj.GetName()
		if name == "" {
			return fmt.Errorf("%s 缺少 metadata.name", gvk.Kind)
		}
		_, getErr := ri.Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(getErr) {
			if _, err := ri.Create(ctx, obj, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}); err != nil {
				return err
			}
			continue
		}
		if getErr != nil {
			return getErr
		}
		if _, err := ri.Update(ctx, obj, metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}}); err != nil {
			return err
		}
	}
	return nil
}
