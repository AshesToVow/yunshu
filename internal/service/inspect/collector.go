package inspect

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/promapi"
)

// MetricSample 单条采集样本。
type MetricSample struct {
	Instance    string            `json:"instance"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	Unit        string            `json:"unit"`
	Status      string            `json:"status"` // normal|warning|critical
	Labels      map[string]string `json:"labels"`
	Timestamp   time.Time         `json:"timestamp"`
	Error       string            `json:"error,omitempty"`
}

// CollectResult 一次采集结果。
type CollectResult struct {
	Samples []MetricSample
	Total   int
	Critical int
	Warning  int
	Normal   int
}

func collectItems(ctx context.Context, cli *promapi.Client, items []model.InspectItem, concurrency int) CollectResult {
	if concurrency <= 0 {
		concurrency = 8
	}
	out := CollectResult{}
	if len(items) == 0 || cli == nil {
		return out
	}
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup
	now := time.Now()

	for _, it := range items {
		if !it.Enabled {
			continue
		}
		it := it
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			samples := queryItem(ctx, cli, it, now)
			mu.Lock()
			out.Samples = append(out.Samples, samples...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	for _, s := range out.Samples {
		out.Total++
		switch s.Status {
		case "critical":
			out.Critical++
		case "warning":
			out.Warning++
		default:
			out.Normal++
		}
	}
	return out
}

func queryItem(ctx context.Context, cli *promapi.Client, it model.InspectItem, now time.Time) []MetricSample {
	q := strings.TrimSpace(it.Query)
	if q == "" {
		return []MetricSample{{
			Name: it.Name, Description: it.Description, Type: it.Type,
			Threshold: it.Threshold, Unit: it.Unit, Status: "critical",
			Timestamp: now, Error: "empty query",
		}}
	}
	body, _, err := cli.QueryInstant(ctx, q, "")
	if err != nil {
		return []MetricSample{{
			Name: it.Name, Description: it.Description, Type: it.Type,
			Threshold: it.Threshold, Unit: it.Unit, Status: "critical",
			Timestamp: now, Error: err.Error(),
		}}
	}
	rows, err := parsePromVector(body)
	if err != nil {
		return []MetricSample{{
			Name: it.Name, Description: it.Description, Type: it.Type,
			Threshold: it.Threshold, Unit: it.Unit, Status: "critical",
			Timestamp: now, Error: err.Error(),
		}}
	}
	if len(rows) == 0 {
		return []MetricSample{{
			Name: it.Name, Description: it.Description, Type: it.Type,
			Threshold: it.Threshold, Unit: it.Unit, Status: "warning",
			Timestamp: now, Instance: "无数据", Labels: map[string]string{},
			Error: "Prometheus 无返回样本（检查指标名/job 是否与 Telegraf、Blackbox 一致）",
		}}
	}
	out := make([]MetricSample, 0, len(rows))
	for _, r := range rows {
		if math.IsNaN(r.Value) || math.IsInf(r.Value, 0) {
			continue
		}
		inst := resolveInstance(r.Metric)
		status := getStatus(r.Value, it.Threshold, it.ThresholdType)
		out = append(out, MetricSample{
			Instance:    inst,
			Name:        it.Name,
			Description: it.Description,
			Type:        it.Type,
			Value:       r.Value,
			Threshold:   it.Threshold,
			Unit:        it.Unit,
			Status:      status,
			Labels:      r.Metric,
			Timestamp:   now,
		})
	}
	if len(out) == 0 {
		return []MetricSample{{
			Name: it.Name, Description: it.Description, Type: it.Type,
			Threshold: it.Threshold, Unit: it.Unit, Status: "warning",
			Timestamp: now, Instance: "无数据", Labels: map[string]string{},
			Error: "样本均为 NaN/Inf",
		}}
	}
	return out
}

type promSample struct {
	Metric map[string]string
	Value  float64
}

func parsePromVector(body json.RawMessage) ([]promSample, error) {
	var wrap struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	if wrap.Status != "success" {
		return nil, fmt.Errorf("prometheus status: %s", wrap.Status)
	}
	if wrap.Data.ResultType != "vector" {
		return nil, fmt.Errorf("unsupported resultType: %s", wrap.Data.ResultType)
	}
	out := make([]promSample, 0, len(wrap.Data.Result))
	for _, r := range wrap.Data.Result {
		v := 0.0
		if len(r.Value) >= 2 {
			switch x := r.Value[1].(type) {
			case string:
				v, _ = strconv.ParseFloat(strings.TrimSpace(x), 64)
			case float64:
				v = x
			default:
				v, _ = strconv.ParseFloat(strings.TrimSpace(fmt.Sprintf("%v", r.Value[1])), 64)
			}
		}
		m := r.Metric
		if m == nil {
			m = map[string]string{}
		}
		out = append(out, promSample{Metric: m, Value: v})
	}
	return out, nil
}

func firstLabel(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	return ""
}

// resolveInstance 从 Prometheus 样本标签解析实例标识。
// Telegraf 常见 host/hostname；Blackbox/Exporter 常见 instance；自定义常有 ip。
func resolveInstance(m map[string]string) string {
	if m == nil {
		return "-"
	}
	if v := firstLabel(m,
		"instance", "host", "hostname", "ip", "agent_host", "agent_hostname",
		"exported_instance", "exported_host", "node", "pod", "server", "target",
	); v != "" {
		return stripInstancePort(v)
	}
	// 兜底：除 __name__/job/quantile 外取第一个非空标签
	skip := map[string]bool{
		"__name__": true, "job": true, "quantile": true, "le": true,
		"cpu": true, "mode": true, "path": true, "fstype": true, "device": true,
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if skip[k] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := strings.TrimSpace(m[k]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(m["job"]); v != "" {
		return v
	}
	return "-"
}

// stripInstancePort 展示时去掉 :9273 这类 scrape 端口，保留 IP/主机名（探测目标含端口则保留）。
func stripInstancePort(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	// IPv6 [::1]:9100
	if strings.HasPrefix(v, "[") {
		if i := strings.LastIndex(v, "]:"); i > 0 {
			return v[:i+1]
		}
		return v
	}
	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return v
	}
	// Blackbox 目标常为 host:业务端口，保留；Telegraf scrape 端口常见 9126/9273/9100 可去掉
	switch port {
	case "9100", "9126", "9273", "8080", "9101", "9102":
		return host
	default:
		return v
	}
}

// getStatus 对齐 PromAI：越界为 critical；接近阈值为 warning（上界 90%、下界 110%）。
func getStatus(value, threshold float64, thresholdType string) string {
	tt := strings.TrimSpace(strings.ToLower(thresholdType))
	if tt == "" {
		tt = "greater"
	}
	thresholdStatus := "critical"
	warningFactor := 0.9

	switch tt {
	case "equal", "=":
		if value == threshold {
			return "normal"
		}
		return thresholdStatus
	case "not_equal", "!=":
		if value != threshold {
			return "normal"
		}
		return thresholdStatus
	case "greater", ">":
		if value > threshold {
			return thresholdStatus
		}
		if value > threshold*warningFactor {
			return "warning"
		}
		return "normal"
	case "greater_equal", ">=":
		if value >= threshold {
			return thresholdStatus
		}
		if value >= threshold*warningFactor {
			return "warning"
		}
		return "normal"
	case "less", "<":
		if value < threshold {
			return thresholdStatus
		}
		if value < threshold*(1+(1-warningFactor)) {
			return "warning"
		}
		return "normal"
	case "less_equal", "<=":
		if value <= threshold {
			return thresholdStatus
		}
		if value <= threshold*(1+(1-warningFactor)) {
			return "warning"
		}
		return "normal"
	default:
		if value > threshold {
			return thresholdStatus
		}
		if value > threshold*warningFactor {
			return "warning"
		}
		return "normal"
	}
}
