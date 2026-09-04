package logplatform

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
)

type preparedSearch struct {
	indices string
	must    []map[string]any
	filters []map[string]any
	mustNot []map[string]any
	cfg     config.ElasticsearchConfig
}

func (s *LogSearchService) prepareSearch(ctx context.Context, q LogSearchQuery) (*preparedSearch, error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	if s.es == nil {
		return nil, constants.ErrBadRequestWithMsg("Elasticsearch 未配置")
	}
	cliCfg, err := s.es.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	_ = cliCfg // client resolved by caller via s.es.Client

	ApplySimplifiedDQL(&q)

	projectID := strconv.FormatUint(uint64(q.ProjectID), 10)
	must := []map[string]any{
		termIDFilter("project_id", projectID),
	}
	filters := make([]map[string]any, 0, 8)
	mustNot := make([]map[string]any, 0)
	if q.ServerID != nil && *q.ServerID > 0 {
		filters = append(filters, termIDFilter("server_id", strconv.FormatUint(uint64(*q.ServerID), 10)))
	}
	if q.ServiceID != nil && *q.ServiceID > 0 {
		filters = append(filters, termIDFilter("service_id", strconv.FormatUint(uint64(*q.ServiceID), 10)))
	}
	if q.LogSourceID != nil && *q.LogSourceID > 0 {
		filters = append(filters, termIDFilter("log_source_id", strconv.FormatUint(uint64(*q.LogSourceID), 10)))
	}
	if lv := strings.TrimSpace(q.Level); lv != "" {
		if clause := levelFilter(lv); clause != nil {
			filters = append(filters, clause)
		}
	}
	if fp := strings.TrimSpace(q.FilePath); fp != "" {
		filters = append(filters, filePathFilter(fp))
	}
	if sn := strings.TrimSpace(q.ServiceName); sn != "" {
		filters = append(filters, termIDFilter("service_name", sn))
	}
	if h := strings.TrimSpace(q.Host); h != "" {
		filters = append(filters, multiFieldTermFilter([]string{"host", "server_host", "hostname"}, h))
	}
	if ef := strings.TrimSpace(q.ExtraField); ef != "" {
		if ev := strings.TrimSpace(q.ExtraValue); ev != "" {
			candidates := []string{ef, ef + ".keyword", "fields." + ef, "fields." + ef + ".keyword"}
			filters = append(filters, multiFieldTermFilter(candidates, ev))
		}
	}
	if q.ClusterID != nil && *q.ClusterID > 0 {
		filters = append(filters, termIDFilter("cluster_id", strconv.FormatUint(uint64(*q.ClusterID), 10)))
	}
	switch normalizeCollectorMode(q.CollectorMode) {
	case "k8s":
		filters = append(filters, termIDFilter("collector_mode", "k8s"))
	case "host":
		filters = append(filters, hostCollectorModeFilter())
	}
	if ns := strings.TrimSpace(q.Namespace); ns != "" {
		filters = append(filters, termIDFilter("namespace", ns))
	}
	if pod := strings.TrimSpace(q.Pod); pod != "" {
		filters = append(filters, multiFieldTermFilter([]string{"podname", "pod"}, pod))
	}
	if ct := strings.TrimSpace(q.Container); ct != "" {
		filters = append(filters, multiFieldTermFilter([]string{"containername", "container"}, ct))
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		must = append(must, map[string]any{
			"simple_query_string": map[string]any{
				"query":            kw,
				"fields":           messageFieldsForQuery(cliCfg.MessageFields),
				"default_operator": "and",
			},
		})
	}
	if timeFilter := timeRangeFilter(q.From, q.To, cliCfg.TimestampField); timeFilter != nil {
		filters = append(filters, timeFilter)
	}
	if q.TraceID != "" {
		filters = append(filters, multiFieldTermFilter([]string{"trace_id", "traceId", "traceid"}, strings.TrimSpace(q.TraceID)))
	}

	if !q.SkipDropRules && s.db != nil {
		rules, err := NewLogDropRuleService(s.db).ListEnabled(ctx, q.ProjectID)
		if err == nil && len(rules) > 0 {
			mustNot = append(mustNot, dropRulesToMustNot(rules)...)
		}
	}

	return &preparedSearch{
		indices: s.resolveIndices(ctx, q),
		must:    must,
		filters: filters,
		mustNot: mustNot,
		cfg:     cliCfg,
	}, nil
}

func (p *preparedSearch) boolQuery() map[string]any {
	bq := map[string]any{
		"must":   p.must,
		"filter": p.filters,
	}
	if len(p.mustNot) > 0 {
		bq["must_not"] = p.mustNot
	}
	return map[string]any{"bool": bq}
}
