package inspect

import "testing"

func TestBuildInspectAlertExpr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		q, tt string
		th    float64
		want  string
	}{
		{"cpu_usage", "greater", 80, "(cpu_usage) > 80"},
		{"mem_free", "less", 0.1, "(mem_free) < 0.1"},
		{"up == 0", "greater", 1, "up == 0"},
	}
	for _, c := range cases {
		got, err := buildInspectAlertExpr(c.q, c.th, c.tt)
		if err != nil {
			t.Fatalf("%q: %v", c.q, err)
		}
		if got != c.want {
			t.Fatalf("got %q want %q", got, c.want)
		}
	}
}
