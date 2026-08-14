package logplatform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"
)

type LogSearchService struct {
	es         *ElasticsearchProvider
	serverRepo interfaces.ServerRepository
}

func NewLogSearchService(es *ElasticsearchProvider, serverRepo interfaces.ServerRepository) *LogSearchService {
	return &LogSearchService{es: es, serverRepo: serverRepo}
}

type LogSearchQuery struct {
	ProjectID     uint   `form:"project_id"`
	ServerID      *uint  `form:"server_id"`
	ServiceID     *uint  `form:"service_id"`
	LogSourceID   *uint  `form:"log_source_id"`
	ServiceName   string `form:"service_name"`
	CollectorMode string `form:"collector_mode"` // host|k8s|空=全部
	ClusterID     *uint  `form:"cluster_id"`
	Namespace     string `form:"namespace"`
	Pod           string `form:"pod"`
	Container     string `form:"container"`
	Keyword       string `form:"keyword"`
	Level         string `form:"level"`
	FilePath      string `form:"file_path"`
	From          string `form:"from"`
	To            string `form:"to"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}

type LogSearchItem struct {
	Timestamp     string `json:"timestamp"`
	Message       string `json:"message"`
	Highlight     string `json:"highlight,omitempty"`
	Level         string `json:"level,omitempty"`
	FilePath      string `json:"file_path,omitempty"`
	ServerID      uint   `json:"server_id,omitempty"`
	ServiceID     uint   `json:"service_id,omitempty"`
	LogSourceID   uint   `json:"log_source_id,omitempty"`
	ServiceName   string `json:"service_name,omitempty"`
	ServerHost    string `json:"server_host,omitempty"`
	Host          string `json:"host,omitempty"`
	CollectorMode string `json:"collector_mode,omitempty"`
	ClusterID     uint   `json:"cluster_id,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	Pod           string `json:"pod,omitempty"`
	PodName       string `json:"podname,omitempty"`
	Container     string `json:"container,omitempty"`
	ContainerName string `json:"containername,omitempty"`
}

func (s *LogSearchService) Search(ctx context.Context, q LogSearchQuery) (*pagination.Result[LogSearchItem], error) {
	if q.ProjectID == 0 {
		return nil, constants.ErrProjectIDRequired
	}
	if s.es == nil {
		return nil, constants.ErrBadRequestWithMsg("Elasticsearch 未配置")
	}
	cli, cfg, err := s.es.Client(ctx)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if pageSize > cfg.MaxSize {
		pageSize = cfg.MaxSize
	}
	from := (page - 1) * pageSize

	projectID := strconv.FormatUint(uint64(q.ProjectID), 10)
	must := []map[string]any{
		termIDFilter("project_id", projectID),
	}
	filters := make([]map[string]any, 0, 8)
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
	if q.ClusterID != nil && *q.ClusterID > 0 {
		filters = append(filters, termIDFilter("cluster_id", strconv.FormatUint(uint64(*q.ClusterID), 10)))
	}
	// collector_mode：k8s 精确匹配；host 兼容未热更旧文档（字段缺失）
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
				"fields":           messageFieldsForQuery(cfg.MessageFields),
				"default_operator": "and",
			},
		})
	}
	if timeFilter := timeRangeFilter(q.From, q.To, cfg.TimestampField); timeFilter != nil {
		filters = append(filters, timeFilter)
	}

	body := map[string]any{
		"track_total_hits": true,
		"from":             from,
		"size":             pageSize,
		"sort":             searchSort(cfg.TimestampField),
		"query": map[string]any{
			"bool": map[string]any{
				"must":   must,
				"filter": filters,
			},
		},
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		body["highlight"] = map[string]any{
			"pre_tags":  []string{"<mark>"},
			"post_tags": []string{"</mark>"},
			"fields":    highlightFields(cfg.MessageFields),
		}
	}

	indices := s.resolveIndices(ctx, q)
	raw, err := cli.Search(ctx, indices, body)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "logsearch", "Search", err)
	}
	items, total := parseSearchHits(raw, cfg)
	return &pagination.Result[LogSearchItem]{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *LogSearchService) resolveIndices(ctx context.Context, q LogSearchQuery) string {
	k8sPrefix := ""
	if s.es != nil {
		if cfg, err := s.es.Resolve(ctx); err == nil {
			k8sPrefix = cfg.K8sIndexPrefix
		}
	}
	mode := normalizeCollectorMode(q.CollectorMode)
	if q.ClusterID != nil && *q.ClusterID > 0 && mode == "" {
		mode = "k8s"
	}
	switch mode {
	case "k8s":
		if q.ClusterID != nil && *q.ClusterID > 0 {
			return K8sIndexPattern(*q.ClusterID, k8sPrefix)
		}
		return GlobalK8sIndexPattern(k8sPrefix)
	case "host":
		return s.resolveHostIndices(ctx, q)
	default:
		host := s.resolveHostIndices(ctx, q)
		k8s := GlobalK8sIndexPattern(k8sPrefix)
		if strings.TrimSpace(host) == "" {
			return k8s
		}
		return host + "," + k8s
	}
}

func normalizeCollectorMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "k8s", "cluster", "kubernetes":
		return "k8s"
	case "host", "agent":
		return "host"
	default:
		return ""
	}
}

// hostCollectorModeFilter 匹配 collector_mode=host，或尚未热更、字段缺失的旧主机文档。
func hostCollectorModeFilter() map[string]any {
	return map[string]any{
		"bool": map[string]any{
			"should": []map[string]any{
				termIDFilter("collector_mode", "host"),
				{
					"bool": map[string]any{
						"must_not": []map[string]any{
							{
								"bool": map[string]any{
									"should": []map[string]any{
										{"exists": map[string]any{"field": "collector_mode"}},
										{"exists": map[string]any{"field": "fields.collector_mode"}},
									},
									"minimum_should_match": 1,
								},
							},
						},
					},
				},
			},
			"minimum_should_match": 1,
		},
	}
}

func (s *LogSearchService) resolveHostIndices(ctx context.Context, q LogSearchQuery) string {
	if q.ServerID != nil && *q.ServerID > 0 {
		if s.serverRepo != nil {
			if sv, err := s.serverRepo.GetByID(ctx, *q.ServerID); err == nil && sv != nil && strings.TrimSpace(sv.Host) != "" {
				return ResolveSearchIndicesByHosts([]string{sv.Host}, []uint{*q.ServerID})
			}
		}
		return AgentIndexPatternByServerID(*q.ServerID)
	}
	var ids []uint
	var hosts []string
	if s.serverRepo != nil {
		servers, _, err := s.serverRepo.List(ctx, repository.ServerListParams{
			ProjectID: q.ProjectID,
			Page:      1,
			PageSize:  maxSearchIndexServers + 1,
		})
		if err == nil {
			ids = make([]uint, 0, len(servers))
			hosts = make([]string, 0, len(servers))
			for _, sv := range servers {
				ids = append(ids, sv.ID)
				if h := strings.TrimSpace(sv.Host); h != "" {
					hosts = append(hosts, h)
				}
			}
		}
	}
	if len(hosts) > 0 {
		return ResolveSearchIndicesByHosts(hosts, ids)
	}
	return ResolveSearchIndices(nil, ids)
}

// multiFieldTermFilter 在多个字段上做 term 兼容（含 fields.* 嵌套）。
func multiFieldTermFilter(fields []string, value string) map[string]any {
	value = strings.TrimSpace(value)
	if value == "" || len(fields) == 0 {
		return map[string]any{"match_all": map[string]any{}}
	}
	should := make([]map[string]any, 0, len(fields)*8)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		for _, path := range []string{field, "fields." + field} {
			should = append(should,
				map[string]any{"term": map[string]any{path + ".keyword": value}},
				map[string]any{"term": map[string]any{path: value}},
			)
			if n, err := strconv.ParseInt(value, 10, 64); err == nil {
				should = append(should, map[string]any{"term": map[string]any{path: n}})
			}
		}
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func (s *LogSearchService) Export(ctx context.Context, q LogSearchQuery) (string, error) {
	q.Page = 1
	if q.PageSize <= 0 {
		if s.es != nil {
			if cfg, err := s.es.Resolve(ctx); err == nil {
				q.PageSize = cfg.MaxSize
			}
		}
		if q.PageSize <= 0 {
			q.PageSize = 1000
		}
	}
	res, err := s.Search(ctx, q)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, it := range res.List {
		prefix := it.Timestamp
		if fp := strings.TrimSpace(it.FilePath); fp != "" {
			prefix = fmt.Sprintf("%s [%s]", prefix, fp)
		}
		if lv := strings.TrimSpace(it.Level); lv != "" {
			prefix = fmt.Sprintf("%s <%s>", prefix, lv)
		}
		b.WriteString(prefix)
		b.WriteString(" ")
		msg := it.Message
		if h := strings.TrimSpace(it.Highlight); h != "" {
			msg = stripHTML(h)
		}
		b.WriteString(msg)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func highlightFields(fields []string) map[string]any {
	out := map[string]any{}
	for _, f := range messageFieldsForQuery(fields) {
		out[f] = map[string]any{}
	}
	if len(out) == 0 {
		out["message"] = map[string]any{}
	}
	return out
}

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<mark>", "")
	s = strings.ReplaceAll(s, "</mark>", "")
	return s
}

// termIDFilter 兼容顶层与 Loggie fields.* 嵌套、keyword/long 映射。
func termIDFilter(field, value string) map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]any{"match_all": map[string]any{}}
	}
	paths := []string{field, "fields." + field}
	should := make([]map[string]any, 0, len(paths)*4)
	for _, path := range paths {
		should = append(should,
			map[string]any{"term": map[string]any{path + ".keyword": value}},
			map[string]any{"term": map[string]any{path: value}},
		)
		// ES 动态映射常把纯数字打成 long，字符串 term 会没命中
		if n, err := strconv.ParseInt(value, 10, 64); err == nil {
			should = append(should, map[string]any{"term": map[string]any{path: n}})
		}
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

// timeRangeFilter 按 @timestamp/timestamp 过滤；无时间字段的历史文档仍可命中。
func timeRangeFilter(from, to, primaryField string) map[string]any {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" && to == "" {
		return nil
	}
	buildRange := func(field string) map[string]any {
		r := map[string]any{}
		if from != "" {
			r["gte"] = from
		}
		if to != "" {
			r["lte"] = to
		}
		return map[string]any{"range": map[string]any{field: r}}
	}
	should := make([]map[string]any, 0, 4)
	seen := map[string]struct{}{}
	for _, f := range []string{primaryField, "@timestamp", "timestamp"} {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		should = append(should, buildRange(f))
	}
	should = append(should, map[string]any{
		"bool": map[string]any{
			"must_not": []map[string]any{
				{"exists": map[string]any{"field": primaryField}},
				{"exists": map[string]any{"field": "@timestamp"}},
				{"exists": map[string]any{"field": "timestamp"}},
			},
		},
	})
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func searchSort(primaryField string) []map[string]any {
	primaryField = strings.TrimSpace(primaryField)
	if primaryField == "" {
		primaryField = "@timestamp"
	}
	return []map[string]any{
		{primaryField: map[string]any{"order": "desc", "unmapped_type": "date", "missing": "_last"}},
		{"_doc": map[string]any{"order": "desc"}},
	}
}

func messageFieldsForQuery(fields []string) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		out = append(out, f)
	}
	if len(out) == 0 {
		return []string{"message"}
	}
	return out
}

func parseSearchHits(raw map[string]any, cfg config.ElasticsearchConfig) ([]LogSearchItem, int64) {
	hitsRoot, _ := raw["hits"].(map[string]any)
	var total int64
	if tv, ok := hitsRoot["total"].(map[string]any); ok {
		if v, ok := tv["value"].(float64); ok {
			total = int64(v)
		}
	} else if v, ok := hitsRoot["total"].(float64); ok {
		total = int64(v)
	}
	hits, _ := hitsRoot["hits"].([]any)
	out := make([]LogSearchItem, 0, len(hits))
	for _, h := range hits {
		hm, ok := h.(map[string]any)
		if !ok {
			continue
		}
		src, _ := hm["_source"].(map[string]any)
		item := mapHit(src, cfg)
		if hl, ok := hm["highlight"].(map[string]any); ok {
			item.Highlight = pickHighlight(hl, cfg.MessageFields)
		}
		out = append(out, item)
	}
	return out, total
}

func pickHighlight(hl map[string]any, fields []string) string {
	for _, f := range messageFieldsForQuery(fields) {
		if arr, ok := hl[f].([]any); ok && len(arr) > 0 {
			if s, ok := arr[0].(string); ok {
				return s
			}
		}
	}
	for _, arr := range hl {
		if a, ok := arr.([]any); ok && len(a) > 0 {
			if s, ok := a[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

func mapHit(src map[string]any, cfg config.ElasticsearchConfig) LogSearchItem {
	meta := nestedFields(src)
	item := LogSearchItem{
		Timestamp:     pickString(src, cfg.TimestampField, "@timestamp", "timestamp", "ts", "time"),
		Message:       pickMessage(src, cfg.MessageFields),
		Level:         pickLevel(src),
		FilePath:      pickFilePath(src),
		Host:          pickString(src, "host", "hostname", "node"),
		CollectorMode: pickString(src, "collector_mode"),
		Namespace:     pickString(src, "namespace", "k8s.namespace"),
		Pod:           pickString(src, "pod", "pod_name", "podname", "kubernetes.pod.name"),
		PodName:       pickString(src, "podname", "pod", "pod_name"),
		Container:     pickString(src, "container", "container_name", "containername", "kubernetes.container.name"),
		ContainerName: pickString(src, "containername", "container", "container_name"),
		ServiceName:   pickString(src, "service_name"),
		ServerHost:    pickString(src, "server_host"),
	}
	if item.Level == "" {
		item.Level = extractLevelFromMessage(item.Message)
	}
	if meta != nil {
		if item.CollectorMode == "" {
			item.CollectorMode = pickString(meta, "collector_mode")
		}
		if item.Namespace == "" {
			item.Namespace = pickString(meta, "namespace")
		}
		if item.Pod == "" {
			item.Pod = pickString(meta, "pod", "podname")
		}
		if item.PodName == "" {
			item.PodName = pickString(meta, "podname", "pod")
		}
		if item.Container == "" {
			item.Container = pickString(meta, "container", "containername")
		}
		if item.ContainerName == "" {
			item.ContainerName = pickString(meta, "containername", "container")
		}
		if item.FilePath == "" {
			item.FilePath = pickString(meta, "file_path", "log_file", "filename")
		}
		if item.ServiceName == "" {
			item.ServiceName = pickString(meta, "service_name")
		}
		if item.ServerHost == "" {
			item.ServerHost = pickString(meta, "server_host")
		}
		if item.Host == "" {
			item.Host = pickString(meta, "host", "hostname")
		}
	}
	if item.PodName == "" {
		item.PodName = item.Pod
	}
	if item.Pod == "" {
		item.Pod = item.PodName
	}
	if item.ContainerName == "" {
		item.ContainerName = item.Container
	}
	if item.Container == "" {
		item.Container = item.ContainerName
	}
	item.ServerID = pickUint(src, "server_id")
	if item.ServerID == 0 {
		item.ServerID = pickUintMap(meta, "server_id")
	}
	item.ClusterID = pickUint(src, "cluster_id")
	if item.ClusterID == 0 {
		item.ClusterID = pickUintMap(meta, "cluster_id")
	}
	item.ServiceID = pickUint(src, "service_id")
	if item.ServiceID == 0 {
		item.ServiceID = pickUintMap(meta, "service_id")
	}
	item.LogSourceID = pickUint(src, "log_source_id")
	if item.LogSourceID == 0 {
		item.LogSourceID = pickUintMap(meta, "log_source_id")
	}
	if item.Timestamp == "" {
		item.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	return item
}

func nestedFields(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	raw, ok := src["fields"].(map[string]any)
	if !ok {
		return nil
	}
	return raw
}

func pickUintMap(src map[string]any, key string) uint {
	if src == nil {
		return 0
	}
	return pickUint(src, key)
}

func pickMessage(src map[string]any, fields []string) string {
	for _, f := range fields {
		if v := anyToLogString(src[f]); v != "" {
			return v
		}
	}
	for _, key := range []string{"message", "body", "log", "msg", "content"} {
		if v := anyToLogString(src[key]); v != "" {
			return v
		}
	}
	if meta := nestedFields(src); meta != nil {
		for _, key := range []string{"message", "body", "log", "msg"} {
			if v := anyToLogString(meta[key]); v != "" {
				return v
			}
		}
	}
	return ""
}

func pickString(src map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			switch t := v.(type) {
			case string:
				s := strings.TrimSpace(t)
				if s != "" && s != "<nil>" {
					return t
				}
			case float64:
				return strconv.FormatInt(int64(t), 10)
			}
		}
	}
	return ""
}

func pickUint(src map[string]any, key string) uint {
	v, ok := src[key]
	if !ok {
		return 0
	}
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return uint(t)
		}
	case string:
		n, _ := strconv.ParseUint(strings.TrimSpace(t), 10, 32)
		return uint(n)
	}
	return 0
}
