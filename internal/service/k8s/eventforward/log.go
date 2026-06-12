package eventforward

import "yunshu/internal/pkg/logutil"

func forwardLog() *logutil.Component {
	return logutil.Worker("k8s.event_forward")
}
