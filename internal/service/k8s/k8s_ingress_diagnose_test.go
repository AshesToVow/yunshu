package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestResolveIngressServicePortNamed(t *testing.T) {
	t.Parallel()
	svc := &corev1.Service{Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{
		{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)},
		{Name: "metrics", Port: 9090, TargetPort: intstr.FromString("metrics")},
	}}}
	got, ok := resolveIngressServicePort(svc, ingressBackendRef{PortName: "http"})
	if !ok || got.Port != 80 || got.TargetPort != 8080 {
		t.Fatalf("named backend port: ok=%v got=%+v", ok, got)
	}
	got, ok = resolveIngressServicePort(svc, ingressBackendRef{PortName: "metrics"})
	if !ok || got.TargetPort != 0 || got.TargetPortName != "metrics" {
		t.Fatalf("named targetPort: ok=%v got=%+v", ok, got)
	}
	if _, ok := resolveIngressServicePort(svc, ingressBackendRef{PortName: "missing"}); ok {
		t.Fatal("missing named port must not resolve")
	}
	if _, ok := resolveIngressServicePort(svc, ingressBackendRef{Port: 8080}); ok {
		t.Fatal("backend port must match service port, not targetPort")
	}
}

func TestCountReadyEndpointsUsesTargetPort(t *testing.T) {
	t.Parallel()
	sp := resolvedIngressServicePort{Name: "", Port: 80, TargetPort: 8080}
	eps := &corev1.Endpoints{Subsets: []corev1.EndpointSubset{{
		Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1"}},
		Ports:     []corev1.EndpointPort{{Port: 8080}},
	}}}
	ready, mismatch := countReadyEndpointsForServicePort(eps, sp)
	if ready != 1 || mismatch {
		t.Fatalf("targetPort match: ready=%d mismatch=%v", ready, mismatch)
	}

	wrong := &corev1.Endpoints{Subsets: []corev1.EndpointSubset{{
		Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1"}},
		Ports:     []corev1.EndpointPort{{Port: 9090}},
	}}}
	ready, mismatch = countReadyEndpointsForServicePort(wrong, sp)
	if ready != 0 || !mismatch {
		t.Fatalf("port mismatch: ready=%d mismatch=%v", ready, mismatch)
	}
}

func TestCountReadyEndpointsByName(t *testing.T) {
	t.Parallel()
	sp := resolvedIngressServicePort{Name: "http", Port: 80, TargetPort: 8080}
	eps := &corev1.Endpoints{Subsets: []corev1.EndpointSubset{{
		Addresses: []corev1.EndpointAddress{{IP: "10.0.0.1"}, {IP: "10.0.0.2"}},
		Ports: []corev1.EndpointPort{
			{Name: "http", Port: 8080},
			{Name: "metrics", Port: 9090},
		},
	}}}
	ready, mismatch := countReadyEndpointsForServicePort(eps, sp)
	if ready != 2 || mismatch {
		t.Fatalf("named endpoint match: ready=%d mismatch=%v", ready, mismatch)
	}
}

func TestCollectIngressBackendsIncludesDefaultAndNamedPort(t *testing.T) {
	t.Parallel()
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
				Name: "fallback",
				Port: networkingv1.ServiceBackendPort{Number: 80},
			}},
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
							Name: "web",
							Port: networkingv1.ServiceBackendPort{Name: "http"},
						}},
					}},
				}},
			}},
		},
	}
	got := collectIngressBackends(ing)
	if len(got) != 2 {
		t.Fatalf("backends=%+v", got)
	}
	if got[0].ServiceName != "fallback" || got[0].Port != 80 {
		t.Fatalf("default backend=%+v", got[0])
	}
	if got[1].ServiceName != "web" || got[1].PortName != "http" {
		t.Fatalf("rule backend=%+v", got[1])
	}
}
