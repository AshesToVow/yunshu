package k8s

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterOwnedControllerRevisions(t *testing.T) {
	t.Parallel()
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{UID: "sts-uid", Name: "demo", Namespace: "ns"},
	}
	owned := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-1",
			OwnerReferences: []metav1.OwnerReference{{
				UID:        "sts-uid",
				Controller: boolPtr(true),
			}},
		},
		Revision: 1,
	}
	other := appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{Name: "other-1"},
		Revision:   9,
	}
	got := filterOwnedControllerRevisions([]appsv1.ControllerRevision{owned, other}, sts)
	if len(got) != 1 || got[0].Name != "demo-1" {
		t.Fatalf("got %#v", got)
	}
}

func boolPtr(v bool) *bool { return &v }
