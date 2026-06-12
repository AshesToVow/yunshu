package k8s

import (
	"strings"

	"yunshu/internal/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sDeleteOptions 对应 Kubernetes DeleteOptions 的可选字段。
type K8sDeleteOptions struct {
	GracePeriodSeconds *int64  `form:"grace_period_seconds" json:"grace_period_seconds,omitempty"`
	PropagationPolicy  *string `form:"propagation_policy" json:"propagation_policy,omitempty"`
}

// ToMetav1 将请求参数转换为 metav1.DeleteOptions；未传字段保持 K8s API 默认行为。
func (o K8sDeleteOptions) ToMetav1() (metav1.DeleteOptions, error) {
	opts := metav1.DeleteOptions{}
	if o.GracePeriodSeconds != nil {
		opts.GracePeriodSeconds = o.GracePeriodSeconds
	}
	if o.PropagationPolicy != nil {
		p := strings.TrimSpace(*o.PropagationPolicy)
		if p == "" {
			return opts, nil
		}
		switch metav1.DeletionPropagation(p) {
		case metav1.DeletePropagationBackground, metav1.DeletePropagationForeground, metav1.DeletePropagationOrphan:
			policy := metav1.DeletionPropagation(p)
			opts.PropagationPolicy = &policy
		default:
			return opts, constants.ErrBadRequestWithMsg("propagation_policy 取值不合法，允许：Background、Foreground、Orphan")
		}
	}
	return opts, nil
}
