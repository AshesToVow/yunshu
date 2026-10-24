package logplatform

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"yunshu/internal/pkg/constants"
)

// LogTopNQuery 排行榜维度查询。
type LogTopNQuery struct {
	LogSearchQuery
	Dim  string `form:"dim"`  // service|pod|host|level|namespace|container|status
	Size int    `form:"size"` // 默认 20，最大 50
}

// LogTopNItem 排行项。
type LogTopNItem struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// LogTopNResult 排行榜结果。
type LogTopNResult struct {
	Dim   string        `json:"dim"`
	Items []LogTopNItem `json:"items"`
	Total int64         `json:"total"` // 当前筛选命中总数（近似）
}

// TopN 按维度返回 TopN 计数（复用 overview terms 聚合）。
func (s *LogSearchService) TopN(ctx context.Context, q LogTopNQuery) (*LogTopNResult, error) {
	dim := strings.ToLower(strings.TrimSpace(q.Dim))
	if dim == "" {
		dim = "service"
	}
	size := q.Size
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}

	prep, err := s.prepareSearch(ctx, q.LogSearchQuery)
	if err != nil {
		return nil, err
	}
	fields := topNFieldCandidates(dim)
	if len(fields) == 0 {
		return nil, constants.ErrBadRequestWithMsg("不支持的 dim，可选 service/pod/host/level/namespace/container/status")
	}
	counts := s.queryTermsFacet(ctx, prep, fields, size)
	type kv struct {
		k string
		v int64
	}
	arr := make([]kv, 0, len(counts))
	var sum int64
	for k, v := range counts {
		arr = append(arr, kv{k, v})
		sum += v
	}
	slices.SortFunc(arr, func(a, b kv) int {
		return cmp.Compare(b.v, a.v)
	})
	items := make([]LogTopNItem, 0, len(arr))
	for _, it := range arr {
		items = append(items, LogTopNItem{Key: it.k, Count: it.v})
	}
	return &LogTopNResult{Dim: dim, Items: items, Total: sum}, nil
}

func topNFieldCandidates(dim string) []string {
	switch dim {
	case "service", "service_name", "svc":
		return []string{"service_name.keyword", "service_name", "fields.service_name.keyword", "fields.service_name"}
	case "pod", "podname":
		return []string{"podname.keyword", "podname", "pod.keyword", "pod"}
	case "host", "hostname":
		return []string{"host.keyword", "host", "server_host.keyword", "server_host", "hostname"}
	case "level", "status", "severity":
		return []string{"level", "level.keyword", "fields.level", "fields.level.keyword"}
	case "namespace", "ns":
		return []string{"namespace.keyword", "namespace"}
	case "container", "containername":
		return []string{"containername.keyword", "containername", "container.keyword", "container"}
	default:
		return nil
	}
}
