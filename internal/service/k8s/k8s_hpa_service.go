package k8s

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/k8sutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"

	kom "github.com/weibaohui/kom/kom"
)

type HPAListQuery = ClusterNamespaceKeywordQuery
type HPADetailQuery = ClusterNamespaceNameQuery
type HPAApplyRequest = ClusterManifestApplyRequest
type HPADeleteRequest = ClusterNamespaceNameQuery

type HPAItem struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	MinReplicas    string            `json:"min_replicas,omitempty"`
	MaxReplicas    string            `json:"max_replicas,omitempty"`
	ScaleTargetRef string            `json:"scale_target_ref,omitempty"`
	MetricsSummary string            `json:"metrics_summary,omitempty"`
	ConditionsText string            `json:"conditions_text,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Age            string            `json:"age,omitempty"`
	CreationTime   string            `json:"creation_time"`
}

type HPADetail struct {
	YAML string `json:"yaml"`
}

type K8sHPAService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

func NewK8sHPAService(runtime *K8sRuntimeService) *K8sHPAService {
	return &K8sHPAService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var hpaGVK = schema.GroupVersionKind{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}

// resolveHPAGVK 解析集群实际提供的 HPA 版本（v2 → v2beta2 → v2beta1 → v1）。
func (s *K8sHPAService) resolveHPAGVK(k *kom.Kubectl) (schema.GroupVersionKind, error) {
	return resolveGVKForCluster(k, hpaGVK)
}

func (s *K8sHPAService) List(ctx context.Context, q HPAListQuery) ([]HPAItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, err := s.resolveHPAGVK(k)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, gvk, q.Namespace)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.hpa", "api", err, constants.ErrFmte5f4df2bc9c2)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]HPAItem, 0, len(listU))
	for i := range listU {
		// 集群可能只提供 autoscaling/v1 或 v2beta*，统一按 unstructured 读取，避免转换到 v2 丢字段。
		h := listU[i]
		name := strings.ToLower(h.GetName())
		if kw != "" && !strings.Contains(name, kw) {
			continue
		}
		out = append(out, hpaToItem(&h))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sHPAService) Detail(ctx context.Context, q HPADetailQuery) (*HPADetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	gvk, err := s.resolveHPAGVK(k)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, gvk, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsge64b05879667)
		}
		return nil, bizerrors.Internalf(ctx, "k8s.hpa", "api", err, constants.ErrFmtd28ea35ac553)
	}
	// 直接序列化集群返回的对象，保留 v1/v2beta* 各自的字段（如 targetCPUUtilizationPercentage），
	// 避免经 v2 结构体转换后丢失字段、误导用户编辑。
	obj := u.DeepCopy()
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
	obj.SetAPIVersion(gvk.GroupVersion().String())
	obj.SetKind(gvk.Kind)
	y, err := yaml.Marshal(obj.Object)
	if err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.hpa", "marshal", err, constants.ErrFmtd28ea35ac553)
	}
	return &HPADetail{YAML: string(y)}, nil
}

func (s *K8sHPAService) Apply(ctx context.Context, req HPAApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	if err := s.dyn.ApplyManifest(ctx, k, req.Manifest); err != nil {
		return k8sFail(ctx, "k8s.hpa", "api", err)
	}
	return nil
}

func (s *K8sHPAService) Delete(ctx context.Context, req HPADeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	gvk, err := s.resolveHPAGVK(k)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, gvk, req.Namespace, req.Name, req.K8sDeleteOptions); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.hpa", "api", err)
	}
	return nil
}

func hpaToItem(h *unstructured.Unstructured) HPAItem {
	min := "-"
	if v, ok, _ := unstructured.NestedInt64(h.Object, "spec", "minReplicas"); ok {
		min = strconv.FormatInt(v, 10)
	}
	max := "-"
	if v, ok, _ := unstructured.NestedInt64(h.Object, "spec", "maxReplicas"); ok {
		max = strconv.FormatInt(v, 10)
	}
	ref := ""
	refKind, _, _ := unstructured.NestedString(h.Object, "spec", "scaleTargetRef", "kind")
	refName, _, _ := unstructured.NestedString(h.Object, "spec", "scaleTargetRef", "name")
	if name := strings.TrimSpace(refName); name != "" {
		ref = strings.TrimSpace(refKind) + "/" + name
	}
	created := h.GetCreationTimestamp().Time
	return HPAItem{
		Name:           h.GetName(),
		Namespace:      h.GetNamespace(),
		MinReplicas:    min,
		MaxReplicas:    max,
		ScaleTargetRef: ref,
		MetricsSummary: summarizeHPAMetrics(h.Object),
		ConditionsText: summarizeHPAConditions(h.Object),
		Labels:         h.GetLabels(),
		Age:            k8sutil.HumanAge(created),
		CreationTime:   created.Format("2006-01-02 15:04:05"),
	}
}

// summarizeHPAMetrics 兼容 autoscaling/v1（targetCPUUtilizationPercentage）与 v2*（metrics 列表）。
func summarizeHPAMetrics(obj map[string]any) string {
	metrics, ok, _ := unstructured.NestedSlice(obj, "spec", "metrics")
	if !ok || len(metrics) == 0 {
		if v, found, _ := unstructured.NestedInt64(obj, "spec", "targetCPUUtilizationPercentage"); found {
			return "Resource/cpu(" + strconv.FormatInt(v, 10) + "%)"
		}
		return "-"
	}
	parts := make([]string, 0, len(metrics))
	for _, raw := range metrics {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		mType, _ := m["type"].(string)
		switch mType {
		case "Resource":
			parts = append(parts, metricNamePart("Resource", m, "resource"))
		case "ContainerResource":
			parts = append(parts, metricNamePart("ContainerResource", m, "containerResource"))
		case "Pods":
			parts = append(parts, metricNamePart("Pods", m, "pods"))
		case "Object":
			parts = append(parts, metricNamePart("Object", m, "object"))
		case "External":
			parts = append(parts, metricNamePart("External", m, "external"))
		default:
			if mType != "" {
				parts = append(parts, mType)
			}
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

func metricNamePart(prefix string, metric map[string]any, sourceKey string) string {
	source, ok := metric[sourceKey].(map[string]any)
	if !ok {
		return prefix
	}
	name, _ := source["name"].(string) // resource / containerResource 直接带 name
	if strings.TrimSpace(name) == "" {
		if metricDef, ok := source["metric"].(map[string]any); ok {
			name, _ = metricDef["name"].(string)
		}
	}
	if strings.TrimSpace(name) == "" {
		return prefix
	}
	return prefix + "/" + name
}

func summarizeHPAConditions(obj map[string]any) string {
	conds, ok, _ := unstructured.NestedSlice(obj, "status", "conditions")
	if !ok || len(conds) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(conds))
	for _, raw := range conds {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := c["type"].(string)
		st, _ := c["status"].(string)
		if typ == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", typ, st))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "; ")
}
