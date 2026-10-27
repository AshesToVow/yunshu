package k8s

import "testing"

func TestDeploymentRolloutComplete(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		strategy  string
		synced    bool
		replicas  int32
		updated   int32
		ready     int32
		available int32
		unavail   int32
		desired   int32
		want      bool
	}{
		{"rolling complete", "RollingUpdate", true, 3, 3, 3, 3, 0, 3, true},
		{"stale generation", "RollingUpdate", false, 3, 3, 3, 3, 0, 3, false},
		{"old replicas remain", "RollingUpdate", true, 4, 3, 4, 4, 0, 3, false},
		{"not all updated", "RollingUpdate", true, 3, 2, 3, 3, 0, 3, false},
		{"unavailable remains", "RollingUpdate", true, 3, 3, 3, 2, 1, 3, false},
		{"recreate complete", "Recreate", true, 3, 3, 3, 3, 0, 3, true},
		{"recreate stale", "Recreate", false, 3, 3, 3, 3, 0, 3, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deploymentRolloutComplete(tc.strategy, tc.synced, tc.replicas, tc.updated, tc.ready, tc.available, tc.unavail, tc.desired)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
