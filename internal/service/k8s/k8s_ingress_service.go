package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8sutil"
	bizerrors "yunshu/internal/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	kom "github.com/weibaohui/kom/kom"
)

type IngressListQuery = ClusterNamespaceKeywordQuery
type IngressDetailQuery = ClusterNamespaceNameQuery
type IngressApplyRequest = ClusterManifestApplyRequest
type IngressDeleteRequest = ClusterNamespaceNameQuery
type IngressClassListQuery = ClusterKeywordQuery
type IngressClassDetailQuery = ClusterNameQuery
type IngressClassApplyRequest = ClusterManifestApplyRequest
type IngressClassDeleteRequest = ClusterNameQuery

type IngressNginxRestartRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace"`
	Selector  string `json:"selector"`
	// Confirm 须为 true，防止误触批量删除 controller Pod。
	Confirm bool `json:"confirm"`
}

type IngressNginxRestartResult struct {
	DeletedCount int      `json:"deleted_count"`
	DeletedNames []string `json:"deleted_names"`
	FailedNames  []string `json:"failed_names,omitempty"`
}

type IngressItem struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	ClassName    string            `json:"class_name,omitempty"`
	RulesText    string            `json:"rules_text,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	HostCount    int               `json:"host_count"`
	TLSCount     int               `json:"tls_count"`
	LoadBalancer string            `json:"load_balancer,omitempty"`
	Age          string            `json:"age,omitempty"`
	CreationTime string            `json:"creation_time"`
}

type IngressClassItem struct {
	Name         string            `json:"name"`
	Controller   string            `json:"controller,omitempty"`
	IngressCount int               `json:"ingress_count"`
	IsDefault    bool              `json:"is_default"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	Age          string            `json:"age,omitempty"`
	CreationTime string            `json:"creation_time"`
}

type IngressClassDetail struct {
	Item IngressClassItem `json:"item"`
	YAML string           `json:"yaml"`
}

type IngressDetail struct {
	Item IngressItem `json:"item"`
	YAML string      `json:"yaml"`
}

type K8sIngressService struct {
	runtime    *K8sRuntimeService
	dyn        *DynamicResourceService
	accessRepo interfaces.K8sClusterAccessRepository
}

// NewK8sIngressService 创建相关逻辑。
func NewK8sIngressService(runtime *K8sRuntimeService, accessRepo interfaces.K8sClusterAccessRepository) *K8sIngressService {
	return &K8sIngressService{runtime: runtime, dyn: NewDynamicResourceService(runtime), accessRepo: accessRepo}
}

var ingressGVK = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}
var ingressClassGVK = schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"}

// resolveIngressGVK 解析集群实际提供的 Ingress 版本（networking.k8s.io/v1 → v1beta1 → extensions/v1beta1）。
// 注意：v1beta1 的 spec.backend 字段与 v1 不同，转换为 networkingv1.Ingress 时后端信息可能缺失，
// 但名称/命名空间/规则主机等列表展示字段仍可用。
func (s *K8sIngressService) resolveIngressGVK(k *kom.Kubectl) (schema.GroupVersionKind, error) {
	return resolveGVKForCluster(k, ingressGVK)
}

// resolveIngressClassGVK 解析集群实际提供的 IngressClass 版本。
func (s *K8sIngressService) resolveIngressClassGVK(k *kom.Kubectl) (schema.GroupVersionKind, error) {
	return resolveGVKForCluster(k, ingressClassGVK)
}

// List 查询列表相关的业务逻辑。
func (s *K8sIngressService) List(ctx context.Context, q IngressListQuery) ([]IngressItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ingGVK, err := s.resolveIngressGVK(k)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, ingGVK, q.Namespace)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt7f0818fd6f52)
	}
	list := make([]networkingv1.Ingress, 0, len(listU))
	for _, item := range listU {
		var ing networkingv1.Ingress
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ing); e != nil {
			continue
		}
		list = append(list, ing)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]IngressItem, 0, len(list))
	for _, ing := range list {
		if kw != "" && !strings.Contains(strings.ToLower(ing.Name), kw) {
			continue
		}
		out = append(out, ingressToItem(&ing))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sIngressService) Detail(ctx context.Context, q IngressDetailQuery) (*IngressDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	ingGVK, err := s.resolveIngressGVK(k)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, ingGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg82a55c47e927)
		}
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmtd0e7b9970841)
	}
	var obj networkingv1.Ingress
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	// 回填集群实际版本，避免把 v1beta1 对象标成 networking.k8s.io/v1 误导用户编辑。
	copyObj.APIVersion = ingGVK.GroupVersion().String()
	copyObj.Kind = ingGVK.Kind
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &IngressDetail{Item: ingressToItem(copyObj), YAML: string(y)}, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sIngressService) Apply(ctx context.Context, req IngressApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	refs := extractIngressRefs(req.Manifest)
	// Apply 本身以 manifest 内声明的 apiVersion 为准；此处仅为「就绪回读」解析集群实际版本。
	ingGVK, gvkErr := s.resolveIngressGVK(k)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 || gvkErr != nil {
			return false
		}
		for _, r := range refs {
			if strings.TrimSpace(r.Name) == "" {
				continue
			}
			ns := strings.TrimSpace(r.Namespace)
			if ns == "" {
				ns = "default"
			}
			if _, e := s.dyn.GetByGVK(c, k, ingGVK, ns, r.Name); e != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return k8sFail(ctx, "k8s.ingress", "api", err)
	}
	return nil
}

// Delete 删除相关的业务逻辑。
func (s *K8sIngressService) Delete(ctx context.Context, req IngressDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	ingGVK, err := s.resolveIngressGVK(k)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, ingGVK, req.Namespace, req.Name, req.K8sDeleteOptions); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.ingress", "api", err)
	}
	return nil
}

// ListClasses 查询列表相关的业务逻辑。
func (s *K8sIngressService) ListClasses(ctx context.Context, q IngressClassListQuery) ([]IngressClassItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	clsGVK, err := s.resolveIngressClassGVK(k)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, clsGVK, "")
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt6c250f47f18b)
	}
	ingGVK, err := s.resolveIngressGVK(k)
	if err != nil {
		return nil, err
	}
	ingsU, err := s.dyn.ListByGVK(ctx, k, ingGVK, "")
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt7f0818fd6f52)
	}
	classCounter := map[string]int{}
	for _, item := range ingsU {
		var ing networkingv1.Ingress
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &ing); e != nil {
			continue
		}
		className := ""
		if ing.Spec.IngressClassName != nil {
			className = strings.TrimSpace(*ing.Spec.IngressClassName)
		}
		if className == "" {
			continue
		}
		classCounter[className]++
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]IngressClassItem, 0, len(listU))
	for _, item := range listU {
		var cls networkingv1.IngressClass
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cls); e != nil {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(cls.Name), kw) {
			continue
		}
		out = append(out, ingressClassToItem(&cls, classCounter[cls.Name]))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// DetailClass 查询详情相关的业务逻辑。
func (s *K8sIngressService) DetailClass(ctx context.Context, q IngressClassDetailQuery) (*IngressClassDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	clsGVK, err := s.resolveIngressClassGVK(k)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, clsGVK, "", q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgeb6e8490034b)
		}
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt829d798aa9fb)
	}
	var obj networkingv1.IngressClass
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = clsGVK.GroupVersion().String()
	copyObj.Kind = clsGVK.Kind
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &IngressClassDetail{Item: ingressClassToItem(copyObj, 0), YAML: string(y)}, nil
}

// ApplyClass 提交申请相关的业务逻辑。
func (s *K8sIngressService) ApplyClass(ctx context.Context, req IngressClassApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	refs := extractIngressClassRefs(req.Manifest)
	// 同 Apply：仅用于就绪回读。
	clsGVK, gvkErr := s.resolveIngressClassGVK(k)
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 || gvkErr != nil {
			return false
		}
		for _, name := range refs {
			if strings.TrimSpace(name) == "" {
				continue
			}
			if _, e := s.dyn.GetByGVK(c, k, clsGVK, "", name); e != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return k8sFail(ctx, "k8s.ingress", "api", err)
	}
	return nil
}

// DeleteClass 删除相关的业务逻辑。
func (s *K8sIngressService) DeleteClass(ctx context.Context, req IngressClassDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	clsGVK, err := s.resolveIngressClassGVK(k)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, clsGVK, "", req.Name, req.K8sDeleteOptions); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.ingress", "api", err)
	}
	return nil
}

// RestartIngressNginxPods 删除 ingress-nginx controller Pods，使其自动重建，从而刷新默认证书等运行态资源。
// 说明：不同安装方式 label 可能不同，这里支持自定义 selector，并提供默认 selector。
func (s *K8sIngressService) RestartIngressNginxPods(ctx context.Context, req IngressNginxRestartRequest) (*IngressNginxRestartResult, error) {
	if !req.Confirm {
		return nil, constants.ErrBadRequestWithMsg("请设置 confirm=true 以确认重启 Ingress-Nginx 控制器")
	}
	if err := s.ensureIngressNginxRestartAccess(ctx, req.ClusterID); err != nil {
		return nil, err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" {
		ns = "ingress-nginx"
	}
	selector := strings.TrimSpace(req.Selector)
	if selector == "" {
		// 官方 chart/controller 常见标签
		selector = "app.kubernetes.io/name=ingress-nginx,app.kubernetes.io/component=controller"
	}

	var pods []corev1.Pod
	q := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).WithLabelSelector(selector)
	if err := q.List(&pods).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt0cbe9766f7af)
	}
	// 兜底：如果 selector 太严格导致空，再尝试兼容历史 label
	if len(pods) == 0 && strings.TrimSpace(req.Selector) == "" {
		fallback := "app=ingress-nginx"
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).WithLabelSelector(fallback).List(&pods).Error; err != nil {
			return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt0cbe9766f7af)
		}
	}
	if len(pods) == 0 {
		return nil, constants.ErrBadRequestWithMsg("未找到匹配的 Ingress-Nginx 控制器 Pod，请检查 namespace 与 selector")
	}

	deleted := make([]string, 0, len(pods))
	failed := make([]string, 0)
	for _, p := range pods {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			continue
		}
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(ns).Name(name).Delete().Error; err != nil {
			failed = append(failed, name)
			continue
		}
		deleted = append(deleted, name)
	}
	sort.Strings(deleted)
	sort.Strings(failed)
	if len(deleted) == 0 {
		return nil, constants.ErrBadRequestWithMsg("未能删除任何 Ingress-Nginx Pod")
	}
	return &IngressNginxRestartResult{DeletedCount: len(deleted), DeletedNames: deleted, FailedNames: failed}, nil
}

func (s *K8sIngressService) ensureIngressNginxRestartAccess(ctx context.Context, clusterID uint) error {
	if clusterID == 0 {
		return constants.ErrBadRequestWithMsg(constants.ErrMsgba2a155d1253)
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.accessRepo == nil {
		return constants.ErrInternal
	}
	pack := k8sauth.PackFromCurrentUser(u)
	if s.accessRepo.EffectiveTier(ctx, pack, clusterID) < K8sAccessRankAdmin {
		return constants.ErrForbiddenWithMsg("重启 Ingress-Nginx 需要集群 admin 档位授权")
	}
	return nil
}

type ingressRef struct {
	Name      string
	Namespace string
}

func extractIngressRefs(manifest string) []ingressRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]ingressRef, 0)
	for _, doc := range docs {
		docTrim := strings.TrimSpace(doc)
		if docTrim == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(docTrim), &m); err != nil {
			continue
		}
		kind, _ := m["kind"].(string)
		if strings.TrimSpace(kind) != "Ingress" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		ns, _ := meta["namespace"].(string)
		name = strings.TrimSpace(name)
		ns = strings.TrimSpace(ns)
		if name != "" && ns != "" {
			out = append(out, ingressRef{Name: name, Namespace: ns})
		}
	}
	return out
}

func extractIngressClassRefs(manifest string) []string {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]string, 0)
	for _, doc := range docs {
		docTrim := strings.TrimSpace(doc)
		if docTrim == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(docTrim), &m); err != nil {
			continue
		}
		kind, _ := m["kind"].(string)
		if strings.TrimSpace(kind) != "IngressClass" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func ingressToItem(ing *networkingv1.Ingress) IngressItem {
	className := ""
	if ing.Spec.IngressClassName != nil {
		className = strings.TrimSpace(*ing.Spec.IngressClassName)
	}
	hostSet := map[string]bool{}
	for _, r := range ing.Spec.Rules {
		h := strings.TrimSpace(r.Host)
		if h != "" {
			hostSet[h] = true
		}
	}
	lb := ""
	if len(ing.Status.LoadBalancer.Ingress) > 0 {
		first := ing.Status.LoadBalancer.Ingress[0]
		if strings.TrimSpace(first.Hostname) != "" {
			lb = first.Hostname
		} else {
			lb = first.IP
		}
	}
	rules := make([]string, 0, len(ing.Spec.Rules))
	for _, r := range ing.Spec.Rules {
		host := strings.TrimSpace(r.Host)
		if host == "" {
			host = "*"
		}
		if r.HTTP == nil || len(r.HTTP.Paths) == 0 {
			rules = append(rules, fmt.Sprintf("%s -> -", host))
			continue
		}
		for _, p := range r.HTTP.Paths {
			path := strings.TrimSpace(p.Path)
			if path == "" {
				path = "/"
			}
			svc := ""
			if p.Backend.Service != nil {
				svc = strings.TrimSpace(p.Backend.Service.Name)
				if p.Backend.Service.Port.Number > 0 {
					svc = fmt.Sprintf("%s:%d", svc, p.Backend.Service.Port.Number)
				} else if strings.TrimSpace(p.Backend.Service.Port.Name) != "" {
					svc = fmt.Sprintf("%s:%s", svc, strings.TrimSpace(p.Backend.Service.Port.Name))
				}
			}
			if svc == "" {
				svc = "-"
			}
			rules = append(rules, fmt.Sprintf("%s%s -> %s", host, path, svc))
		}
	}
	rulesText := strings.Join(rules, "\n")
	if strings.TrimSpace(rulesText) == "" {
		rulesText = "-"
	}
	return IngressItem{
		Name:         ing.Name,
		Namespace:    ing.Namespace,
		ClassName:    className,
		RulesText:    rulesText,
		Labels:       ing.Labels,
		Annotations:  ing.Annotations,
		HostCount:    len(hostSet),
		TLSCount:     len(ing.Spec.TLS),
		LoadBalancer: lb,
		Age:          k8sutil.HumanAge(ing.CreationTimestamp.Time),
		CreationTime: ing.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}

func ingressClassToItem(cls *networkingv1.IngressClass, ingressCount int) IngressClassItem {
	isDefault := false
	if cls.Annotations != nil {
		v := strings.TrimSpace(cls.Annotations["ingressclass.kubernetes.io/is-default-class"])
		isDefault = strings.EqualFold(v, "true")
	}
	return IngressClassItem{
		Name:         cls.Name,
		Controller:   strings.TrimSpace(cls.Spec.Controller),
		IngressCount: ingressCount,
		IsDefault:    isDefault,
		Labels:       cls.Labels,
		Annotations:  cls.Annotations,
		Age:          k8sutil.HumanAge(cls.CreationTimestamp.Time),
		CreationTime: cls.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
	}
}
