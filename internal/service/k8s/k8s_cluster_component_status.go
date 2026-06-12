package k8s

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func isKubeControlPlanePod(p corev1.Pod) bool {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return false
	}
	return strings.HasPrefix(name, "kube-apiserver-") ||
		strings.HasPrefix(name, "kube-controller-manager-") ||
		strings.HasPrefix(name, "kube-scheduler-") ||
		strings.HasPrefix(name, "etcd-")
}

func nodeToComponentStatusItem(n corev1.Node, probedAt string) ComponentStatusItem {
	healthy, state, msg, reason := nodeReadySummary(n)
	return ComponentStatusItem{
		Name:        "node/" + n.Name,
		Status:      state,
		Healthy:     healthy,
		Message:     msg,
		Error:       reason,
		LastProbeAt: probedAt,
	}
}

func nodeReadySummary(n corev1.Node) (healthy bool, state, message, reason string) {
	state = "Unknown"
	for _, c := range n.Status.Conditions {
		if c.Type != corev1.NodeReady {
			continue
		}
		switch c.Status {
		case corev1.ConditionTrue:
			state = "Ready"
			healthy = true
		case corev1.ConditionFalse:
			state = "NotReady"
		default:
			state = "Unknown"
		}
		message = strings.TrimSpace(c.Message)
		reason = strings.TrimSpace(string(c.Reason))
		break
	}
	return healthy, state, message, reason
}

func podToComponentStatusItem(p corev1.Pod, probedAt string) ComponentStatusItem {
	healthy := p.Status.Phase == corev1.PodRunning
	for _, c := range p.Status.ContainerStatuses {
		if !c.Ready {
			healthy = false
		}
	}
	state := string(p.Status.Phase)
	if state == "" {
		state = "Unknown"
	}
	msg := ""
	if !healthy {
		for _, cs := range p.Status.ContainerStatuses {
			if w := cs.State.Waiting; w != nil && w.Message != "" {
				msg = w.Message
				break
			}
			if t := cs.State.Terminated; t != nil && t.Message != "" {
				msg = t.Message
				break
			}
		}
	}
	return ComponentStatusItem{
		Name:        "pod/" + p.Namespace + "/" + p.Name,
		Status:      state,
		Healthy:     healthy,
		Message:     msg,
		LastProbeAt: probedAt,
	}
}
