package dictmask

import (
	"strings"
	"testing"
)

// TestValueHash 守门：唯一性依赖 value_hash，必须保证「去首尾空白后相同即同哈希」，
// 且不同值不碰撞（kubeconfig 等大文本走同一函数）。
func TestValueHash(t *testing.T) {
	t.Parallel()

	const kubeconfig = "apiVersion: v1\nkind: Config\nclusters: []\n"
	tests := []struct {
		name string
		a    string
		b    string
		want bool // 两值哈希是否相同
	}{
		{name: "same value", a: "abc", b: "abc", want: true},
		{name: "trim spaces", a: "  abc \n", b: "abc\t", want: true},
		{name: "inner space differs", a: "a b", b: "a  b", want: false},
		{name: "case differs", a: "Abc", b: "abc", want: false},
		{name: "empty vs space", a: "", b: "   ", want: true},
		{name: "large text same", a: kubeconfig, b: "\n" + kubeconfig, want: true},
		{name: "large text differs", a: kubeconfig, b: kubeconfig + "x", want: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ha, hb := ValueHash(tc.a), ValueHash(tc.b)
			if got := ha == hb; got != tc.want {
				t.Fatalf("ValueHash(%q)=%s, ValueHash(%q)=%s, equal=%v want %v", tc.a, ha, tc.b, hb, got, tc.want)
			}
			if len(ha) != 64 {
				t.Fatalf("ValueHash(%q) length = %d, want 64", tc.a, len(ha))
			}
			if strings.ToLower(ha) != ha {
				t.Fatalf("ValueHash(%q) = %s, want lowercase hex", tc.a, ha)
			}
		})
	}
}

// TestValueHashKnownVector 固定向量：哈希算法一旦变更，存量 value_hash 会静默失配，必须显式改版并回填。
func TestValueHashKnownVector(t *testing.T) {
	t.Parallel()
	// echo -n "abc" | sha256sum
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := ValueHash(" abc "); got != want {
		t.Fatalf("ValueHash(\" abc \") = %s, want %s", got, want)
	}
}
