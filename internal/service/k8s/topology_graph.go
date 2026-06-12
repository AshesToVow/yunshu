package k8s

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type graphBuilder struct {
	nodes   []TopologyNode
	edges   []TopologyEdge
	nodeSet map[string]struct{}
	edgeSet map[string]struct{}
}

func newGraphBuilder() *graphBuilder {
	return &graphBuilder{
		nodeSet: map[string]struct{}{},
		edgeSet: map[string]struct{}{},
	}
}

func (b *graphBuilder) addNode(id, label, kind, state string) {
	if _, ok := b.nodeSet[id]; ok {
		return
	}
	b.nodeSet[id] = struct{}{}
	b.nodes = append(b.nodes, TopologyNode{
		ID:         id,
		Label:      label,
		Kind:       kind,
		State:      state,
		StateLevel: classifyTopologyState(kind, state),
	})
}

func (b *graphBuilder) addEdge(from, to, kind string) {
	key := from + "->" + to + ":" + kind
	if _, ok := b.edgeSet[key]; ok {
		return
	}
	b.edgeSet[key] = struct{}{}
	b.edges = append(b.edges, TopologyEdge{From: from, To: to, Kind: kind})
}

func (b *graphBuilder) build() *TopologyGraph {
	return &TopologyGraph{Nodes: b.nodes, Edges: b.edges}
}

// classifyTopologyState 参考 KubeVela：蓝=正常、黄=进行中、红=异常。
func classifyTopologyState(kind, state string) string {
	s := strings.ToLower(strings.TrimSpace(state))
	switch {
	case strings.Contains(s, "fail"), strings.Contains(s, "error"), strings.Contains(s, "crash"),
		strings.Contains(s, "backoff"), strings.Contains(s, "unknown"), s == "0", s == "0/0":
		return "abnormal"
	case strings.Contains(s, "pend"), strings.Contains(s, "progress"), strings.Contains(s, "terminat"),
		strings.Contains(s, "init"), strings.Contains(s, "creating"):
		return "progressing"
	default:
		return "normal"
	}
}

func isOwnedBy(obj metav1.Object, ownerKind, ownerName string) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if strings.EqualFold(ref.Kind, ownerKind) && ref.Name == ownerName {
			return true
		}
	}
	return false
}
