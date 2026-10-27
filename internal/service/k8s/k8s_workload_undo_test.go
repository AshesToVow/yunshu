package k8s

import (
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestApplyControllerRevisionTemplateToStatefulSet(t *testing.T) {
	t.Parallel()
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "current", Image: "v2"}}},
			},
		},
	}
	hist := appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"version": "v1"}},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Name: "current", Image: "v1", Env: []corev1.EnvVar{{Name: "MODE", Value: "old"}},
				}}},
			},
		},
	}
	raw, err := json.Marshal(hist)
	if err != nil {
		t.Fatal(err)
	}
	cr := &appsv1.ControllerRevision{Data: runtime.RawExtension{Raw: raw}}
	if err := applyControllerRevisionTemplateToStatefulSet(sts, cr); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got := sts.Spec.Template.Spec.Containers[0].Image; got != "v1" {
		t.Fatalf("image=%q want v1", got)
	}
	if got := sts.Spec.Template.Labels["version"]; got != "v1" {
		t.Fatalf("template labels not restored: %#v", sts.Spec.Template.Labels)
	}
}

func TestApplyControllerRevisionTemplateToStatefulSet_Errors(t *testing.T) {
	t.Parallel()
	sts := &appsv1.StatefulSet{}
	cases := []struct {
		name string
		cr   *appsv1.ControllerRevision
		want string
	}{
		{name: "nil_cr", cr: nil, want: "历史版本数据为空"},
		{name: "empty_raw", cr: &appsv1.ControllerRevision{}, want: "历史版本数据为空"},
		{name: "bad_json", cr: &appsv1.ControllerRevision{Data: runtime.RawExtension{Raw: []byte("{")}}, want: "unexpected end"},
		{
			name: "no_containers",
			cr: &appsv1.ControllerRevision{Data: runtime.RawExtension{
				Raw: []byte(`{"spec":{"template":{"spec":{}}}}`),
			}},
			want: "未包含 Pod 模板",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := applyControllerRevisionTemplateToStatefulSet(sts, tc.cr)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want contains %q", err, tc.want)
			}
		})
	}
}
