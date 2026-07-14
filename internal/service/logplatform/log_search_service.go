package logplatform

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	bizerrors "yunshu/internal/pkg/errors"
)

type LogSearchService struct {
	es *ElasticsearchProvider
}

func NewLogSearchService(es *ElasticsearchProvider) *LogSearchService {
	return &LogSearchService{es: es}
}

type LogSearchQuery struct {
	ProjectID   uint   `form:"project_id"`
	ServerID    *uint  `form:"server_id"`
	ServiceID   *uint  `form:"service_id"`
	LogSourceID *uint  `form:"log_source_id"`
	Keyword     string `form:"keyword"`
	Level       string `form:"level"`
	From        string `form:"from"`
	To          string `form:"to"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type LogSearchItem struct {
	Timestamp   string `json:"timestamp"`
	Message     string `json:"message"`
	Highlight   string `json:"highlight,omitempty"`
	Level       string `json:"level,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
	ServerID    uint   `json:"server_id,omitempty"`
	ServiceID   uint   `json:"service_id,omitempty"`
	LogSourceID uint   `json:"log_source_id,omitempty"`
	Host        string `json:"host,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Pod         string `json:"pod,omitempty"`
	Container   string `json:"container,omitempty"`
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

	raw, err := cli.Search(ctx, cfg.IndexPattern, body)
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

// termIDFilter 兼容顶层与 Loggie fields.* 嵌套、ES7 keyword 映射。
func termIDFilter(field, value string) map[string]any {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]any{"match_all": map[string]any{}}
	}
	paths := []string{field, "fields." + field}
	should := make([]map[string]any, 0, len(paths)*2)
	for _, path := range paths {
		should = append(should,
			map[string]any{"term": map[string]any{path + ".keyword": value}},
			map[string]any{"term": map[string]any{path: value}},
		)
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
		Timestamp: pickString(src, cfg.TimestampField, "@timestamp", "timestamp", "ts", "time"),
		Message:   pickMessage(src, cfg.MessageFields),
		Level:     pickLevel(src),
		FilePath:  pickFilePath(src),
		Host:      pickString(src, "host", "hostname", "node"),
		Namespace: pickString(src, "namespace", "k8s.namespace"),
		Pod:       pickString(src, "pod", "pod_name", "kubernetes.pod.name"),
		Container: pickString(src, "container", "container_name", "kubernetes.container.name"),
	}
	if item.Level == "" {
		item.Level = extractLevelFromMessage(item.Message)
	}
	item.ServerID = pickUint(src, "server_id")
	if item.ServerID == 0 {
		item.ServerID = pickUintMap(meta, "server_id")
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
		if v := pickString(src, f); v != "" {
			return v
		}
	}
	return pickString(src, "message", "body", "log")
}

func pickString(src map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := src[k]; ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
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
