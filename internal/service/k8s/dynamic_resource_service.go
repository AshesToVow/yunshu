package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type DynamicResourceService struct {
	runtime *K8sRuntimeService
}

// NewDynamicResourceService 创建相关逻辑。
func NewDynamicResourceService(runtime *K8sRuntimeService) *DynamicResourceService {
	return &DynamicResourceService{runtime: runtime}
}

// gvkClusterScoped 集群级 GVK 兜底表：仅在集群 discovery 尚未就绪（kom 未缓存 APIResources）时使用。
// 正常路径请用 namespacedForGVK，以集群实际返回的 APIResource.Namespaced 为准。
func gvkClusterScoped(gvk schema.GroupVersionKind) bool {
	switch strings.TrimSpace(gvk.Kind) {
	case "Namespace", "Node", "PersistentVolume", "StorageClass",
		"ClusterRole", "ClusterRoleBinding", "CustomResourceDefinition",
		"IngressClass", "NodeMetrics", "ComponentStatus", "PriorityClass",
		"RuntimeClass", "CSIDriver", "CSINode", "VolumeAttachment",
		"MutatingWebhookConfiguration", "ValidatingWebhookConfiguration",
		"APIService", "PodSecurityPolicy":
		return true
	default:
		return false
	}
}

func applyListNamespaceScope(q *kom.Kubectl, gvk schema.GroupVersionKind, namespace string) *kom.Kubectl {
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		return q.Namespace(ns)
	}
	if namespacedForGVK(q, gvk) {
		return q.AllNamespace()
	}
	return q
}

// ListByGVK 查询列表相关的业务逻辑。白名单激活且未指定 namespace 时按允许 NS 分别 List，避免全集群拉取。
func (s *DynamicResourceService) ListByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace string) ([]unstructured.Unstructured, error) {
	if nsList := s.scopedNamespacesForList(ctx, k, namespace, gvk); len(nsList) > 0 {
		var all []unstructured.Unstructured
		for _, ns := range nsList {
			part, err := s.listByGVKRaw(ctx, k, gvk, ns)
			if err != nil {
				return nil, err
			}
			all = append(all, part...)
		}
		return s.filterUnstructuredByNSPolicy(ctx, k, gvk, namespace, all)
	}
	list, err := s.listByGVKRaw(ctx, k, gvk, namespace)
	if err != nil {
		return nil, err
	}
	return s.filterUnstructuredByNSPolicy(ctx, k, gvk, namespace, list)
}

func (s *DynamicResourceService) listByGVKRaw(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace string) ([]unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	var list []unstructured.Unstructured
	q := applyListNamespaceScope(k.WithContext(ctx).Resource(u), gvk, namespace)
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *DynamicResourceService) scopedNamespacesForList(ctx context.Context, k *kom.Kubectl, namespace string, gvk schema.GroupVersionKind) []string {
	if strings.TrimSpace(namespace) != "" || !namespacedForGVK(k, gvk) {
		return nil
	}
	if s.runtime == nil || s.runtime.nsAllow == nil {
		return nil
	}
	scope, ok := k8sauth.RequestScopeFromContext(ctx)
	if !ok || scope.ClusterID == 0 {
		return nil
	}
	active, err := s.runtime.nsAllow.WhitelistActiveForCluster(ctx, scope.Pack, scope.ClusterID)
	if err != nil || !active {
		return nil
	}
	nsList, err := s.runtime.nsAllow.WhitelistUnionNamespaces(ctx, scope.Pack, scope.ClusterID)
	if err != nil || len(nsList) == 0 {
		return nil
	}
	return nsList
}

// ListByGVKWithSelector 查询列表相关的业务逻辑。
func (s *DynamicResourceService) ListByGVKWithSelector(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, labelSelector string) ([]unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	var list []unstructured.Unstructured
	q := applyListNamespaceScope(k.WithContext(ctx).Resource(u), gvk, namespace)
	if ls := strings.TrimSpace(labelSelector); ls != "" {
		q = q.WithLabelSelector(ls)
	}
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	return s.filterUnstructuredByNSPolicy(ctx, k, gvk, namespace, list)
}

// GetByGVK 获取相关的业务逻辑。
func (s *DynamicResourceService) GetByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, name string) (*unstructured.Unstructured, error) {
	ns := strings.TrimSpace(namespace)
	if err := s.ensureNamespaceAllowed(ctx, ns); err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	q := k.WithContext(ctx).Resource(u).Name(strings.TrimSpace(name))
	if ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.Get(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func (s *DynamicResourceService) ensureNamespaceAllowed(ctx context.Context, namespace string) error {
	ns := strings.TrimSpace(namespace)
	if ns == "" || s == nil || s.runtime == nil {
		return nil
	}
	if k8sauth.SkipNamespacePolicy(ctx) {
		return nil
	}
	clusterID := k8sauth.ClusterIDFromContext(ctx)
	if clusterID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	pack := k8sauth.PackFromCurrentUser(u)
	allowed, err := NamespaceAllowedByPolicy(
		ctx,
		s.runtime.NamespaceDenyRepo(),
		s.runtime.NamespaceAllowRepo(),
		pack,
		clusterID,
		ns,
	)
	if err != nil {
		return bizerrors.Pass(ctx, "k8s.dyn", "ensureNamespaceAllowed", err)
	}
	if !allowed {
		return constants.ErrForbiddenWithMsg("当前主体在此集群下禁止访问命名空间「" + ns + "」")
	}
	return nil
}

func (s *DynamicResourceService) filterUnstructuredByNSPolicy(
	ctx context.Context,
	k *kom.Kubectl,
	gvk schema.GroupVersionKind,
	namespace string,
	list []unstructured.Unstructured,
) ([]unstructured.Unstructured, error) {
	if !namespacedForGVK(k, gvk) || len(list) == 0 {
		return list, nil
	}
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		if err := s.ensureNamespaceAllowed(ctx, ns); err != nil {
			return nil, err
		}
		return list, nil
	}
	clusterID := k8sauth.ClusterIDFromContext(ctx)
	if clusterID == 0 || s == nil || s.runtime == nil {
		return list, nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return list, nil
	}
	pack := k8sauth.PackFromCurrentUser(u)
	names := make([]string, 0, len(list))
	seen := map[string]struct{}{}
	for i := range list {
		n := list[i].GetNamespace()
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	allowedNames, err := FilterNamespaceNamesByPolicy(
		ctx,
		s.runtime.NamespaceDenyRepo(),
		s.runtime.NamespaceAllowRepo(),
		pack,
		clusterID,
		names,
	)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "k8s.dyn", "filterUnstructuredByNSPolicy", err)
	}
	okSet := make(map[string]struct{}, len(allowedNames))
	for _, n := range allowedNames {
		okSet[n] = struct{}{}
	}
	out := make([]unstructured.Unstructured, 0, len(list))
	for i := range list {
		n := list[i].GetNamespace()
		if n == "" {
			out = append(out, list[i])
			continue
		}
		if _, yes := okSet[n]; yes {
			out = append(out, list[i])
		}
	}
	return out, nil
}

// DeleteByGVK 删除相关的业务逻辑。
func (s *DynamicResourceService) DeleteByGVK(ctx context.Context, k *kom.Kubectl, gvk schema.GroupVersionKind, namespace, name string, opts K8sDeleteOptions) error {
	deleteOptions, err := opts.ToMetav1()
	if err != nil {
		return err
	}
	// 先按集群 discovery 解析真实存在的 GVK/GVR，再退回 kom Tools；
	// 避免同 Kind 多版本时 GetGVRByKind 任意挑选一个版本。
	// 仅对登记了降级表的 Kind 做严格解析（见 hasVersionCandidates 注释）。
	var (
		gvr        schema.GroupVersionResource
		namespaced bool
		resolveErr error
	)
	if hasVersionCandidates(gvk) {
		var capability resourceCapability
		capability, resolveErr = resolveClusterResource(k, gvk)
		if resolveErr == nil && capability.Discovered && !capability.GVR.Empty() {
			gvr = capability.GVR
			namespaced = capability.Namespaced
		}
	}
	if gvr.Empty() {
		var ok bool
		gvr, namespaced, ok = k.Tools().GetGVRByGVK(gvk)
		if !ok || gvr.Empty() {
			if resolveErr != nil {
				return resolveErr
			}
			gvr, namespaced = k.Tools().GetGVRByKind(gvk.Kind)
		}
	}
	if gvr.Empty() {
		return fmt.Errorf("unknown GVK: %v", gvk)
	}
	ns := strings.TrimSpace(namespace)
	name = strings.TrimSpace(name)
	if name == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsge278df185255)
	}
	if namespaced {
		if ns == "" {
			ns = metav1.NamespaceDefault
		}
		if err := s.ensureNamespaceAllowed(ctx, ns); err != nil {
			return err
		}
	}
	dc := k.DynamicClient()
	if namespaced {
		return dc.Resource(gvr).Namespace(ns).Delete(ctx, name, deleteOptions)
	}
	return dc.Resource(gvr).Delete(ctx, name, deleteOptions)
}

// ResolveCRKindFromCRD 执行对应的业务逻辑。
func (s *DynamicResourceService) ResolveCRKindFromCRD(ctx context.Context, k *kom.Kubectl, group, version, resource string) (string, error) {
	kind, _, err := s.resolveCRDMeta(ctx, k, group, version, resource)
	return kind, err
}

func (s *DynamicResourceService) resolveCRDMeta(ctx context.Context, k *kom.Kubectl, group, version, resource string) (kind string, namespaced bool, err error) {
	group = strings.TrimSpace(group)
	version = strings.TrimSpace(version)
	resource = strings.TrimSpace(resource)
	if group == "" || version == "" || resource == "" {
		return "", false, constants.ErrBadRequestWithMsg(constants.ErrMsgf757c3be22a2)
	}
	list, err := s.ListByGVK(ctx, k, schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}, "")
	if err != nil {
		return "", false, bizerrors.Internalf(ctx, "k8s.dynamic", "list_crd", err, constants.ErrFmt2b30d4949c98)
	}
	for _, item := range list {
		var crd apiextv1.CustomResourceDefinition
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &crd); e != nil {
			continue
		}
		if strings.TrimSpace(crd.Spec.Group) != group {
			continue
		}
		if strings.TrimSpace(crd.Spec.Names.Plural) != resource {
			continue
		}
		for _, v := range crd.Spec.Versions {
			if strings.TrimSpace(v.Name) == version && v.Served {
				ns := crd.Spec.Scope != apiextv1.ClusterScoped
				return strings.TrimSpace(crd.Spec.Names.Kind), ns, nil
			}
		}
	}
	return "", false, constants.ErrBadRequestWithMsg(constants.ErrMsg1a5f8ce82917)
}

// ListCR 查询列表相关的业务逻辑。
func (s *DynamicResourceService) ListCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace string) ([]unstructured.Unstructured, error) {
	kind, namespaced, err := s.resolveCRDMeta(ctx, k, group, version, resource)
	if err != nil {
		return nil, err
	}
	var list []unstructured.Unstructured
	q := k.WithContext(ctx).CRD(group, version, kind)
	ns := strings.TrimSpace(namespace)
	if ns != "" {
		q = q.Namespace(ns)
	} else if namespaced {
		q = q.AllNamespace()
	}
	if err := q.List(&list).Error; err != nil {
		return nil, err
	}
	if !namespaced {
		return list, nil
	}
	gvk := schema.GroupVersionKind{Group: strings.TrimSpace(group), Version: strings.TrimSpace(version), Kind: kind}
	return s.filterUnstructuredByNSPolicy(ctx, k, gvk, ns, list)
}

// GetCR 获取相关的业务逻辑。
func (s *DynamicResourceService) GetCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {
	kind, namespaced, err := s.resolveCRDMeta(ctx, k, group, version, resource)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(namespace)
	if namespaced {
		if ns == "" {
			ns = metav1.NamespaceDefault
		}
		if err := s.ensureNamespaceAllowed(ctx, ns); err != nil {
			return nil, err
		}
	}
	var obj unstructured.Unstructured
	q := k.WithContext(ctx).CRD(group, version, kind).Name(strings.TrimSpace(name))
	if namespaced && ns != "" {
		q = q.Namespace(ns)
	}
	if err := q.Get(&obj).Error; err != nil {
		return nil, err
	}
	return &obj, nil
}

// DeleteCR 删除相关的业务逻辑。
func (s *DynamicResourceService) DeleteCR(ctx context.Context, k *kom.Kubectl, group, version, resource, namespace, name string, opts K8sDeleteOptions) error {
	kind, err := s.ResolveCRKindFromCRD(ctx, k, group, version, resource)
	if err != nil {
		return err
	}
	gvk := schema.GroupVersionKind{Group: strings.TrimSpace(group), Version: strings.TrimSpace(version), Kind: kind}
	return s.DeleteByGVK(ctx, k, gvk, namespace, name, opts)
}

// ApplyManifest 提交申请相关的业务逻辑；Apply 前校验 YAML 中各命名空间是否允许。
func (s *DynamicResourceService) ApplyManifest(ctx context.Context, k *kom.Kubectl, manifest string, exists func(context.Context) bool) error {
	if err := s.ensureManifestNamespacesAllowed(ctx, manifest); err != nil {
		return err
	}
	if err := k.WithContext(ctx).Applier().Apply(manifest); err != nil {
		if k8sutil.IsLikelySuccessfulApplyError(err) {
			return nil
		}
		if exists != nil && exists(ctx) {
			return nil
		}
		return fmt.Errorf("%v", err)
	}
	return nil
}

func (s *DynamicResourceService) ensureManifestNamespacesAllowed(ctx context.Context, manifest string) error {
	for _, ns := range extractManifestNamespaces(manifest) {
		if err := s.ensureNamespaceAllowed(ctx, ns); err != nil {
			return err
		}
	}
	return nil
}

// extractManifestNamespaces 从多文档 YAML 提取命名空间资源目标 NS（缺省则视为 default）。
func extractManifestNamespaces(manifest string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(ns string) {
		ns = strings.TrimSpace(ns)
		if ns == "" {
			ns = metav1.NamespaceDefault
		}
		if _, ok := seen[ns]; ok {
			return
		}
		seen[ns] = struct{}{}
		out = append(out, ns)
	}
	for _, doc := range k8sutil.SplitYAMLDocs(manifest) {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(doc), &m); err != nil || m == nil {
			continue
		}
		kind, _ := m["kind"].(string)
		kind = strings.TrimSpace(kind)
		if kind == "" || gvkClusterScoped(schema.GroupVersionKind{Kind: kind}) {
			// Namespace 资源本身：校验其 metadata.name（创建 NS 不视为「往某 NS 写」）
			if kind == "Namespace" {
				continue
			}
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			add("")
			continue
		}
		ns, _ := meta["namespace"].(string)
		add(ns)
	}
	return out
}

// GVKByKind 执行对应的业务逻辑。
func (s *DynamicResourceService) GVKByKind(kind string) (schema.GroupVersionKind, bool) {
	switch strings.TrimSpace(kind) {
	case "Namespace":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}, true
	case "ConfigMap":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}, true
	case "Secret":
		return schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}, true
	case "Ingress":
		return schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}, true
	case "Deployment":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}, true
	case "StatefulSet":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}, true
	case "DaemonSet":
		return schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}, true
	case "Job":
		return schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}, true
	case "CronJob":
		return schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"}, true
	case "CustomResourceDefinition":
		return schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}, true
	default:
		return schema.GroupVersionKind{}, false
	}
}

// ExistsByKind 执行对应的业务逻辑。
func (s *DynamicResourceService) ExistsByKind(ctx context.Context, k *kom.Kubectl, kind, namespace, name string) bool {
	gvk, ok := s.GVKByKind(kind)
	if !ok || strings.TrimSpace(name) == "" {
		return false
	}
	_, err := s.GetByGVK(ctx, k, gvk, strings.TrimSpace(namespace), strings.TrimSpace(name))
	return err == nil
}
