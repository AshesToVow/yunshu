package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	if len(ing.Spec.Rules) == 0 && ing.Spec.DefaultBackend == nil {
		checks = append(checks, IngressDiagnoseCheck{Level: "warn", Code: "no_rules", Message: "未配置 rules 或 defaultBackend，无 HTTP 路由"})
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

		svcPort, ok := resolveIngressServicePort(&svc, b)
		if !ok {
			checks = append(checks, IngressDiagnoseCheck{
				Level:   "error",
				Code:    "service_port_missing",
				Message: fmt.Sprintf("Service %s/%s 未找到 Ingress 引用的端口 %s", ns, b.ServiceName, b.portDesc()),
			})
			continue
		}
		checks = append(checks, IngressDiagnoseCheck{
			Level:   "info",
			Code:    "service_port_ok",
			Message: fmt.Sprintf("Service %s/%s 端口 %s 对应 targetPort %s", ns, b.ServiceName, svcPort.display(), svcPort.targetDesc()),
		})

		var eps corev1.Endpoints
		if err := k.WithContext(ctx).Resource(&corev1.Endpoints{}).Namespace(ns).Name(b.ServiceName).Get(&eps).Error; err != nil {
			checks = append(checks, IngressDiagnoseCheck{Level: "warn", Code: "endpoints_missing", Message: fmt.Sprintf("Endpoints %s/%s 不可用: %v", ns, b.ServiceName, err)})
			continue
		}
		ready, mismatch := countReadyEndpointsForServicePort(&eps, svcPort)
		switch {
		case ready > 0:
			checks = append(checks, IngressDiagnoseCheck{Level: "info", Code: "endpoints_ready", Message: fmt.Sprintf("Service %s/%s 有 %d 个就绪 Endpoint", ns, b.ServiceName, ready)})
		case mismatch:
			checks = append(checks, IngressDiagnoseCheck{
				Level:   "error",
				Code:    "endpoint_port_mismatch",
				Message: fmt.Sprintf("Endpoints %s/%s 的端口与 Service 端口不匹配（期望 %s，实际 %s）", ns, b.ServiceName, svcPort.display(), endpointPortsDesc(&eps)),
			})
		default:
			checks = append(checks, IngressDiagnoseCheck{
				Level:   "error",
				Code:    "no_ready_endpoints",
				Message: fmt.Sprintf("Service %s/%s 无就绪 Endpoint（端口 %s）", ns, b.ServiceName, svcPort.display()),
			})
		}
	}
	return &IngressDiagnoseResult{Checks: checks}, nil
}

type ingressBackendRef struct {
	Namespace   string
	ServiceName string
	Port        int32
	PortName    string
}

func (b ingressBackendRef) portDesc() string {
	if b.PortName != "" {
		return b.PortName
	}
	return strconv.Itoa(int(b.Port))
}

func collectIngressBackends(ing *networkingv1.Ingress) []ingressBackendRef {
	var out []ingressBackendRef
	add := func(ns, svc string, port int32, portName string) {
		if strings.TrimSpace(svc) == "" {
			return
		}
		out = append(out, ingressBackendRef{Namespace: ns, ServiceName: svc, Port: port, PortName: portName})
	}
	addBackend := func(ns string, backend *networkingv1.IngressBackend) {
		if backend == nil || backend.Service == nil {
			return
		}
		port := int32(0)
		portName := ""
		if backend.Service.Port.Number > 0 {
			port = backend.Service.Port.Number
		} else {
			portName = strings.TrimSpace(backend.Service.Port.Name)
		}
		add(ns, backend.Service.Name, port, portName)
	}
	addBackend(ing.Namespace, ing.Spec.DefaultBackend)
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			addBackend(ing.Namespace, &p.Backend)
		}
	}
	return out
}

type resolvedIngressServicePort struct {
	Name           string
	Port           int32
	TargetPort     int32
	TargetPortName string
}

func (p resolvedIngressServicePort) display() string {
	if p.Name != "" {
		return fmt.Sprintf("%s(%d)", p.Name, p.Port)
	}
	return strconv.Itoa(int(p.Port))
}

func (p resolvedIngressServicePort) targetDesc() string {
	if p.TargetPortName != "" && p.TargetPort == 0 {
		return fmt.Sprintf("%s(命名端口)", p.TargetPortName)
	}
	if p.TargetPort > 0 {
		return strconv.Itoa(int(p.TargetPort))
	}
	return strconv.Itoa(int(p.Port))
}

func resolveIngressServicePort(svc *corev1.Service, b ingressBackendRef) (resolvedIngressServicePort, bool) {
	if svc == nil {
		return resolvedIngressServicePort{}, false
	}
	for _, sp := range svc.Spec.Ports {
		if b.PortName != "" {
			if strings.TrimSpace(sp.Name) != b.PortName {
				continue
			}
		} else if b.Port <= 0 || sp.Port != b.Port {
			continue
		}
		out := resolvedIngressServicePort{Name: sp.Name, Port: sp.Port}
		switch sp.TargetPort.Type {
		case intstr.Int:
			if sp.TargetPort.IntVal > 0 {
				out.TargetPort = sp.TargetPort.IntVal
			}
		case intstr.String:
			out.TargetPortName = sp.TargetPort.StrVal
		}
		return out, true
	}
	return resolvedIngressServicePort{}, false
}

// countReadyEndpointsForServicePort 统计就绪地址。Endpoint 端口是 targetPort 解析后的容器端口，
// 不能拿 Service 的 port 直接比较；优先按服务端口名匹配，其次按数值 targetPort 匹配。
func countReadyEndpointsForServicePort(eps *corev1.Endpoints, sp resolvedIngressServicePort) (ready int, mismatch bool) {
	if eps == nil {
		return 0, false
	}
	for _, subset := range eps.Subsets {
		matched := len(subset.Ports) == 0
		for _, p := range subset.Ports {
			if endpointPortMatches(p, sp) {
				matched = true
				break
			}
		}
		if !matched {
			mismatch = true
			continue
		}
		ready += len(subset.Addresses)
	}
	return ready, mismatch
}

func endpointPortMatches(p corev1.EndpointPort, sp resolvedIngressServicePort) bool {
	if sp.Name != "" {
		if p.Name == sp.Name {
			return true
		}
		if p.Name != "" {
			return false
		}
	}
	expected := sp.Port
	if sp.TargetPort > 0 {
		expected = sp.TargetPort
	}
	if sp.TargetPort == 0 && sp.TargetPortName != "" && sp.Name == "" {
		// 单端口 Service 的命名 targetPort：Endpoint 端口名也是空，无法再细分，视为命中。
		return true
	}
	return p.Port == expected
}

func endpointPortsDesc(eps *corev1.Endpoints) string {
	if eps == nil {
		return "-"
	}
	var parts []string
	for _, subset := range eps.Subsets {
		for _, p := range subset.Ports {
			if p.Name != "" {
				parts = append(parts, fmt.Sprintf("%s(%d)", p.Name, p.Port))
			} else {
				parts = append(parts, strconv.Itoa(int(p.Port)))
			}
		}
	}
	if len(parts) == 0 {
		return "无端口"
	}
	return strings.Join(parts, ",")
}
