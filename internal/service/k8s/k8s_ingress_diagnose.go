package k8s

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

type IngressDiagnoseQuery = IngressDetailQuery

type IngressDiagnoseCheck struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type IngressDiagnoseResult struct {
	Checks []IngressDiagnoseCheck `json:"checks"`
}

func (s *K8sIngressService) Diagnose(ctx context.Context, q IngressDiagnoseQuery) (*IngressDiagnoseResult, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	var ing networkingv1.Ingress
	if err := k.WithContext(ctx).Resource(&networkingv1.Ingress{}).Namespace(q.Namespace).Name(q.Name).Get(&ing).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgf6d026c4bc20)
		}
		return nil, bizerrors.Internalf(ctx, "k8s.ingress", "api", err, constants.ErrFmt38cc1640ac12)
	}
	checks := []IngressDiagnoseCheck{
		{Level: "info", Code: "ingress_found", Message: fmt.Sprintf("Ingress %s/%s 存在", q.Namespace, q.Name)},
	}
	if len(ing.Spec.Rules) == 0 {
		checks = append(checks, IngressDiagnoseCheck{Level: "warn", Code: "no_rules", Message: "未配置 rules，无 HTTP 路由"})
	}
	backends := collectIngressBackends(&ing)
	if len(backends) == 0 {
		checks = append(checks, IngressDiagnoseCheck{Level: "error", Code: "no_backends", Message: "未解析到 Service 后端"})
	}
	for _, b := range backends {
		ns := b.Namespace
		if ns == "" {
			ns = q.Namespace
		}
		var svc corev1.Service
		if err := k.WithContext(ctx).Resource(&corev1.Service{}).Namespace(ns).Name(b.ServiceName).Get(&svc).Error; err != nil {
			if apierrors.IsNotFound(err) {
				checks = append(checks, IngressDiagnoseCheck{Level: "error", Code: "service_missing", Message: fmt.Sprintf("后端 Service %s/%s 不存在", ns, b.ServiceName)})
				continue
			}
			checks = append(checks, IngressDiagnoseCheck{Level: "error", Code: "service_error", Message: fmt.Sprintf("查询 Service %s/%s 失败: %v", ns, b.ServiceName, err)})
			continue
		}
		checks = append(checks, IngressDiagnoseCheck{Level: "info", Code: "service_ok", Message: fmt.Sprintf("Service %s/%s 存在 (ClusterIP=%s)", ns, b.ServiceName, svc.Spec.ClusterIP)})

		var eps corev1.Endpoints
		if err := k.WithContext(ctx).Resource(&corev1.Endpoints{}).Namespace(ns).Name(b.ServiceName).Get(&eps).Error; err != nil {
			checks = append(checks, IngressDiagnoseCheck{Level: "warn", Code: "endpoints_missing", Message: fmt.Sprintf("Endpoints %s/%s 不可用: %v", ns, b.ServiceName, err)})
			continue
		}
		ready := countReadyEndpoints(&eps, b.Port)
		if ready == 0 {
			checks = append(checks, IngressDiagnoseCheck{Level: "error", Code: "no_ready_endpoints", Message: fmt.Sprintf("Service %s/%s 无就绪 Endpoint (port=%d)", ns, b.ServiceName, b.Port)})
		} else {
			checks = append(checks, IngressDiagnoseCheck{Level: "info", Code: "endpoints_ready", Message: fmt.Sprintf("Service %s/%s 有 %d 个就绪 Endpoint", ns, b.ServiceName, ready)})
		}
	}
	return &IngressDiagnoseResult{Checks: checks}, nil
}

type ingressBackendRef struct {
	Namespace   string
	ServiceName string
	Port        int32
}

func collectIngressBackends(ing *networkingv1.Ingress) []ingressBackendRef {
	var out []ingressBackendRef
	add := func(ns, svc string, port int32) {
		if strings.TrimSpace(svc) == "" {
			return
		}
		out = append(out, ingressBackendRef{Namespace: ns, ServiceName: svc, Port: port})
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.Backend.Service != nil {
				port := int32(0)
				if p.Backend.Service.Port.Number > 0 {
					port = p.Backend.Service.Port.Number
				}
				add(ing.Namespace, p.Backend.Service.Name, port)
			}
		}
	}
	for _, tls := range ing.Spec.TLS {
		_ = tls
	}
	return out
}

func countReadyEndpoints(eps *corev1.Endpoints, wantPort int32) int {
	if eps == nil {
		return 0
	}
	n := 0
	for _, subset := range eps.Subsets {
		for _, addr := range subset.Addresses {
			if addr.IP != "" {
				if wantPort <= 0 {
					n++
					continue
				}
				for _, p := range subset.Ports {
					if p.Port == wantPort {
						n++
						break
					}
				}
			}
		}
	}
	return n
}
