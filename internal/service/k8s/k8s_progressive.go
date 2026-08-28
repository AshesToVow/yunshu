package k8s

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"yunshu/internal/pkg/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	progressiveColorLabel = "yunshu.io/color"
	progressiveTrackLabel = "yunshu.io/track"
)

// ProgressiveScaleRequest 扩缩副本。
type ProgressiveScaleRequest struct {
	ClusterID uint
	Namespace string
	Name      string
	Replicas  int32
}

// ProgressivePatchImageRequest 更新 Deployment 容器镜像。
type ProgressivePatchImageRequest struct {
	ClusterID uint
	Namespace string
	Name      string
	Image     string
}

// ProgressiveSwitchServiceRequest 切换 Service selector 的 color。
type ProgressiveSwitchServiceRequest struct {
	ClusterID   uint
	Namespace   string
	ServiceName string
	Color       string // blue|green
	AppLabel    string // optional app label key=value match; if empty keep existing + set color
}

// ProgressiveEnsureCanaryRequest 从稳定 Deployment 克隆/更新金丝雀副本。
type ProgressiveEnsureCanaryRequest struct {
	ClusterID      uint
	Namespace      string
	StableName     string
	CanaryName     string
	Image          string
	CanaryReplicas int32
}

func (s *K8sWorkloadService) ProgressiveScaleDeployment(ctx context.Context, req ProgressiveScaleRequest) error {
	return s.DeploymentScale(ctx, WorkloadScaleRequest{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		Name:      req.Name,
		Replicas:  req.Replicas,
	})
}

func (s *K8sWorkloadService) ProgressivePatchDeploymentImage(ctx context.Context, req ProgressivePatchImageRequest) (map[string]any, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	image := strings.TrimSpace(req.Image)
	if image == "" {
		return nil, constants.ErrBadRequestWithMsg("镜像地址不能为空")
	}
	var obj appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(req.Name).Get(&obj).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg("Deployment 不存在: " + req.Name)
		}
		return nil, k8sFail(ctx, "k8s.progressive", "get", err)
	}
	copyObj := obj.DeepCopy()
	if len(copyObj.Spec.Template.Spec.Containers) == 0 {
		return nil, constants.ErrBadRequestWithMsg("Deployment 无容器")
	}
	before := copyObj.Spec.Template.Spec.Containers[0].Image
	copyObj.Spec.Template.Spec.Containers[0].Image = image
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return nil, k8sFail(ctx, "k8s.progressive", "update", err)
	}
	return map[string]any{
		"name":   req.Name,
		"before": before,
		"after":  image,
	}, nil
}

func (s *K8sWorkloadService) ProgressiveEnsureCanaryDeployment(ctx context.Context, req ProgressiveEnsureCanaryRequest) (map[string]any, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	stableName := strings.TrimSpace(req.StableName)
	canaryName := strings.TrimSpace(req.CanaryName)
	if stableName == "" || canaryName == "" {
		return nil, constants.ErrBadRequestWithMsg("稳定版/金丝雀工作负载名不能为空")
	}
	var stable appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(stableName).Get(&stable).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg("稳定版 Deployment 不存在: " + stableName)
		}
		return nil, k8sFail(ctx, "k8s.progressive", "get_stable", err)
	}
	replicas := max(req.CanaryReplicas, 1)
	image := strings.TrimSpace(req.Image)

	var existing appsv1.Deployment
	err = k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Name(canaryName).Get(&existing).Error
	if err == nil {
		copyObj := existing.DeepCopy()
		copyObj.Spec.Replicas = &replicas
		if image != "" && len(copyObj.Spec.Template.Spec.Containers) > 0 {
			copyObj.Spec.Template.Spec.Containers[0].Image = image
		}
		ensureProgressiveLabels(copyObj.Labels, "canary")
		ensureProgressiveLabels(copyObj.Spec.Template.Labels, "canary")
		if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
			return nil, k8sFail(ctx, "k8s.progressive", "update_canary", err)
		}
		return map[string]any{"action": "updated", "name": canaryName, "replicas": replicas}, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, k8sFail(ctx, "k8s.progressive", "get_canary", err)
	}

	canary := stable.DeepCopy()
	canary.ObjectMeta = metav1.ObjectMeta{
		Name:        canaryName,
		Namespace:   req.Namespace,
		Labels:      cloneStringMap(stable.Labels),
		Annotations: cloneStringMap(stable.Annotations),
	}
	delete(canary.Annotations, "deployment.kubernetes.io/revision")
	canary.ResourceVersion = ""
	canary.UID = ""
	canary.CreationTimestamp = metav1.Time{}
	canary.Status = appsv1.DeploymentStatus{}
	canary.Spec.Replicas = &replicas
	ensureProgressiveLabels(canary.Labels, "canary")
	ensureProgressiveLabels(canary.Spec.Template.Labels, "canary")
	if canary.Spec.Selector != nil {
		if canary.Spec.Selector.MatchLabels == nil {
			canary.Spec.Selector.MatchLabels = map[string]string{}
		}
		canary.Spec.Selector.MatchLabels[progressiveTrackLabel] = "canary"
	}
	if image != "" && len(canary.Spec.Template.Spec.Containers) > 0 {
		canary.Spec.Template.Spec.Containers[0].Image = image
	}
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(req.Namespace).Create(canary).Error; err != nil {
		return nil, k8sFail(ctx, "k8s.progressive", "create_canary", err)
	}
	return map[string]any{"action": "created", "name": canaryName, "replicas": replicas}, nil
}

func (s *K8sWorkloadService) ProgressiveSwitchServiceColor(ctx context.Context, req ProgressiveSwitchServiceRequest) (map[string]any, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	svcName := strings.TrimSpace(req.ServiceName)
	color := strings.ToLower(strings.TrimSpace(req.Color))
	if svcName == "" || (color != "blue" && color != "green") {
		return nil, constants.ErrBadRequestWithMsg("Service 名与 color(blue|green) 必填")
	}
	var svc corev1.Service
	if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(req.Namespace).Name(svcName).Get(&svc).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg("Service 不存在: " + svcName)
		}
		return nil, k8sFail(ctx, "k8s.progressive", "get_svc", err)
	}
	copyObj := svc.DeepCopy()
	if copyObj.Spec.Selector == nil {
		copyObj.Spec.Selector = map[string]string{}
	}
	before := copyObj.Spec.Selector[progressiveColorLabel]
	copyObj.Spec.Selector[progressiveColorLabel] = color
	if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(req.Namespace).Update(copyObj).Error; err != nil {
		return nil, k8sFail(ctx, "k8s.progressive", "update_svc", err)
	}
	return map[string]any{
		"service": svcName,
		"before":  before,
		"after":   color,
	}, nil
}

func (s *K8sWorkloadService) ProgressiveEnsureColorDeployment(ctx context.Context, clusterID uint, namespace, baseName, color, image string, replicas int32) (map[string]any, error) {
	name := fmt.Sprintf("%s-%s", strings.TrimSpace(baseName), color)
	_, k, err := s.runtime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	var stable appsv1.Deployment
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(namespace).Name(baseName).Get(&stable).Error; err != nil {
		// 若基线不存在，尝试用 blue 作为基线
		if apierrors.IsNotFound(err) {
			alt := baseName + "-blue"
			if err2 := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(namespace).Name(alt).Get(&stable).Error; err2 != nil {
				return nil, constants.ErrBadRequestWithMsg("找不到基线 Deployment: " + baseName)
			}
		} else {
			return nil, k8sFail(ctx, "k8s.progressive", "get_base", err)
		}
	}
	if replicas < 1 {
		replicas = 1
	}
	var existing appsv1.Deployment
	err = k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(namespace).Name(name).Get(&existing).Error
	if err == nil {
		copyObj := existing.DeepCopy()
		copyObj.Spec.Replicas = &replicas
		if image != "" && len(copyObj.Spec.Template.Spec.Containers) > 0 {
			copyObj.Spec.Template.Spec.Containers[0].Image = image
		}
		ensureColorLabels(copyObj.Labels, color)
		ensureColorLabels(copyObj.Spec.Template.Labels, color)
		if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(namespace).Update(copyObj).Error; err != nil {
			return nil, k8sFail(ctx, "k8s.progressive", "update_color", err)
		}
		return map[string]any{"action": "updated", "name": name, "color": color}, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, k8sFail(ctx, "k8s.progressive", "get_color", err)
	}
	dep := stable.DeepCopy()
	dep.ObjectMeta = metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      cloneStringMap(stable.Labels),
		Annotations: cloneStringMap(stable.Annotations),
	}
	dep.ResourceVersion = ""
	dep.UID = ""
	dep.Status = appsv1.DeploymentStatus{}
	dep.Spec.Replicas = &replicas
	ensureColorLabels(dep.Labels, color)
	ensureColorLabels(dep.Spec.Template.Labels, color)
	if dep.Spec.Selector != nil {
		if dep.Spec.Selector.MatchLabels == nil {
			dep.Spec.Selector.MatchLabels = map[string]string{}
		}
		dep.Spec.Selector.MatchLabels[progressiveColorLabel] = color
	}
	if image != "" && len(dep.Spec.Template.Spec.Containers) > 0 {
		dep.Spec.Template.Spec.Containers[0].Image = image
	}
	if err := k.WithContext(ctx).Resource(&appsv1.Deployment{}).Namespace(namespace).Create(dep).Error; err != nil {
		return nil, k8sFail(ctx, "k8s.progressive", "create_color", err)
	}
	return map[string]any{"action": "created", "name": name, "color": color}, nil
}

func ensureProgressiveLabels(m map[string]string, track string) {
	if m == nil {
		return
	}
	m[progressiveTrackLabel] = track
}

func ensureColorLabels(m map[string]string, color string) {
	if m == nil {
		return
	}
	m[progressiveColorLabel] = color
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
