package inspect

import "testing"

func TestBuildAssetBucketsDynamicTypes(t *testing.T) {
	t.Parallel()
	samples := []MetricSample{
		{Type: "基础设施层", Name: "CPU 使用率", Status: "normal", Instance: "h1"},
		{Type: "基础设施层", Name: "磁盘使用率", Status: "critical", Instance: "h1"},
		{Type: "数据库监控", Name: "MySQL 存活", Status: "normal", Instance: "db1"},
		{Type: "中间件层", Name: "Redis 存活", Status: "warning", Instance: "r1"},
		{Type: "k8s集群", Name: "Node Ready", Status: "normal", Instance: "node1"},
		{Type: "业务网关", Name: "网关可用性", Status: "critical", Instance: "gw1"},
	}
	groups := buildContentGroups(samples)
	buckets := buildAssetBuckets(samples, groups)
	wantLabels := map[string]bool{
		"基础设施层": true, "数据库监控": true, "中间件层": true, "k8s集群": true, "业务网关": true,
	}
	if len(buckets) != 5 {
		t.Fatalf("want 5 dynamic buckets, got %+v", buckets)
	}
	for _, b := range buckets {
		if !wantLabels[b.Label] {
			t.Fatalf("unexpected label %q", b.Label)
		}
		delete(wantLabels, b.Label)
	}
	if len(wantLabels) != 0 {
		t.Fatalf("missing labels: %v", wantLabels)
	}

	secs := buildCategorySections(groups, nil, samples, 60)
	if len(secs) != 5 {
		t.Fatalf("want 5 category sections, got %d", len(secs))
	}
	var sawK8s, sawCustom, sawHostAvg bool
	for _, s := range secs {
		switch s.Title {
		case "k8s集群":
			sawK8s = true
			if s.Kind != "k8s" {
				t.Fatalf("k8s kind=%s", s.Kind)
			}
		case "业务网关":
			sawCustom = true
			if s.Kind != "other" {
				t.Fatalf("custom kind=%s", s.Kind)
			}
		case "基础设施层":
			if s.HostAvg == nil {
				t.Fatal("host section should attach HostAvg")
			}
			sawHostAvg = true
		}
	}
	if !sawK8s || !sawCustom || !sawHostAvg {
		t.Fatalf("sawK8s=%v sawCustom=%v sawHostAvg=%v", sawK8s, sawCustom, sawHostAvg)
	}
}
