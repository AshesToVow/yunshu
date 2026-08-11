package logplatform

import (
	"strings"
	"testing"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

func TestClusterRuleExcludeFiles_WideCollect(t *testing.T) {
	ex := clusterRuleExcludeFiles(model.ClusterLogRule{})
	if len(ex) == 0 {
		t.Fatal("wide collect should emit excludeFiles")
	}
	joined := strings.Join(ex, "\n")
	for _, ns := range []string{"kube-system", "kube-public", "yunshu-logging"} {
		if !strings.Contains(joined, ns+"_") {
			t.Fatalf("missing exclude for %s: %s", ns, joined)
		}
	}
}

func TestClusterRuleExcludeFiles_ScopedNS(t *testing.T) {
	ex := clusterRuleExcludeFiles(model.ClusterLogRule{
		MatchNamespaces: `["default"]`,
	})
	if len(ex) != 0 {
		t.Fatalf("scoped ns should not need excludeFiles, got %v", ex)
	}
}

func TestBuildClusterPipelinesYAML_RateLimitAndExclude(t *testing.T) {
	yml := BuildClusterPipelinesYAML(
		1,
		2,
		nil,
		config.ElasticsearchConfig{},
		config.KafkaConfig{},
		1500,
	)
	if !strings.Contains(yml, "type: rateLimit") {
		t.Fatal("missing rateLimit interceptor")
	}
	if !strings.Contains(yml, "qps: 1500") {
		t.Fatal("missing project qps")
	}
	if !strings.Contains(yml, "excludeFiles:") {
		t.Fatal("default wide pipeline should exclude system ns")
	}
	if !strings.Contains(yml, "collector_mode: \"k8s\"") {
		t.Fatal("missing collector_mode field")
	}
}

func TestBuildClusterPipelinesYAML_RuleQPSOverride(t *testing.T) {
	yml := BuildClusterPipelinesYAML(
		1,
		2,
		[]model.ClusterLogRule{{
			ID:              9,
			Name:            "app",
			Enabled:         true,
			ParseProfile:    "cri",
			MatchNamespaces: `["default"]`,
			RateLimitQPS:    500,
		}},
		config.ElasticsearchConfig{},
		config.KafkaConfig{},
		2000,
	)
	if !strings.Contains(yml, "qps: 500") {
		t.Fatal("rule qps should override project qps")
	}
	if strings.Contains(yml, "excludeFiles:") {
		t.Fatal("scoped rule should not use excludeFiles")
	}
}

func TestAllocateRuleRateLimits_EqualSplit(t *testing.T) {
	got := allocateRuleRateLimits([]model.ClusterLogRule{
		{ID: 1, Enabled: true, MatchNamespaces: `["a"]`},
		{ID: 2, Enabled: true, MatchNamespaces: `["b"]`},
		{ID: 3, Enabled: false, MatchNamespaces: `["c"]`},
	}, 2000)
	if got[1] != 1000 || got[2] != 1000 {
		t.Fatalf("want equal 1000/1000, got %v", got)
	}
	if _, ok := got[3]; ok {
		t.Fatal("disabled rule should not allocate")
	}
}

func TestAllocateRuleRateLimits_fixedPlusShare(t *testing.T) {
	got := allocateRuleRateLimits([]model.ClusterLogRule{
		{ID: 1, Enabled: true, MatchNamespaces: `["a"]`, RateLimitQPS: 800},
		{ID: 2, Enabled: true, MatchNamespaces: `["b"]`},
		{ID: 3, Enabled: true, MatchNamespaces: `["c"]`},
	}, 2000)
	if got[1] != 800 {
		t.Fatalf("fixed want 800, got %d", got[1])
	}
	if got[2] != 600 || got[3] != 600 {
		t.Fatalf("remain 1200 should split to 600/600, got %v", got)
	}
}

func TestBuildClusterPipelinesYAML_SplitBudget(t *testing.T) {
	yml := BuildClusterPipelinesYAML(
		1,
		2,
		[]model.ClusterLogRule{
			{ID: 1, Name: "a", Enabled: true, ParseProfile: "cri", MatchNamespaces: `["ns-a"]`},
			{ID: 2, Name: "b", Enabled: true, ParseProfile: "cri", MatchNamespaces: `["ns-b"]`},
		},
		config.ElasticsearchConfig{},
		config.KafkaConfig{},
		2000,
	)
	if strings.Count(yml, "qps: 1000") < 2 {
		t.Fatalf("two inheriting rules should each get 1000, yaml:\n%s", yml)
	}
}
