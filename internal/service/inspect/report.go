package inspect

import (
	"embed"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

//go:embed templates/*.html
var reportFS embed.FS

// ReportData HTML 报告数据。
type ReportData struct {
	Timestamp      time.Time
	Project        string
	Datasource     string
	InspectionUser string
	Score          float64
	Grade          string
	Summary        string
	ReportListMode string
	ReportListHint string
	Total          int
	Critical       int
	Warning        int
	Normal         int
	ContentGroups  []ContentGroup // 巡检内容（全量范围）
	Groups         []ReportGroup  // 各类巡检结果（受 listMode 过滤）
	Findings       []Finding
}

type ContentGroup struct {
	Type  string
	Items []ContentItem
	Stats GroupStats
}

// ContentItem 某分类下的一条巡检项（去重后的检查名称）。
type ContentItem struct {
	Name        string
	Description string
	SampleCount int
}

type ReportGroup struct {
	Type    string
	Metrics []MetricSample
	Stats   GroupStats
}

type GroupStats struct {
	Total    int
	Critical int
	Warning  int
	Normal   int
}

type Finding struct {
	Type     string
	Name     string
	Severity string
	Count    int
	Hint     string
}

func buildReportData(projectName, dsName, user, listMode string, collected CollectResult) ReportData {
	mode := strings.TrimSpace(listMode)
	if mode == "" {
		mode = "abnormal_only"
	}
	filtered := filterSamples(collected.Samples, mode)
	byType := map[string][]MetricSample{}
	for _, s := range filtered {
		t := s.Type
		if t == "" {
			t = "未分类"
		}
		byType[t] = append(byType[t], s)
	}
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	groups := make([]ReportGroup, 0, len(types))
	for _, t := range types {
		ms := byType[t]
		st := GroupStats{}
		for _, m := range ms {
			st.Total++
			switch m.Status {
			case "critical":
				st.Critical++
			case "warning":
				st.Warning++
			default:
				st.Normal++
			}
		}
		groups = append(groups, ReportGroup{Type: t, Metrics: ms, Stats: st})
	}
	score, grade := scorecard(collected)
	findings := buildFindings(collected.Samples)
	contentGroups := buildContentGroups(collected.Samples)
	hint := "展示全部样本"
	switch mode {
	case "abnormal_only":
		hint = "仅展示异常（critical/warning）样本"
	case "summary":
		hint = "摘要模式：每类仅展示异常样本"
	}
	return ReportData{
		Timestamp:      time.Now(),
		Project:        projectName,
		Datasource:     dsName,
		InspectionUser: user,
		Score:          score,
		Grade:          grade,
		Summary:        fmt.Sprintf("共 %d 条样本：严重 %d、警告 %d、正常 %d。健康分 %.0f（%s）。", collected.Total, collected.Critical, collected.Warning, collected.Normal, score, grade),
		ReportListMode: mode,
		ReportListHint: hint,
		Total:          collected.Total,
		Critical:       collected.Critical,
		Warning:        collected.Warning,
		Normal:         collected.Normal,
		ContentGroups:  contentGroups,
		Groups:         groups,
		Findings:       findings,
	}
}

func buildContentGroups(samples []MetricSample) []ContentGroup {
	byType := map[string]map[string]*ContentItem{}
	statsByType := map[string]*GroupStats{}
	for _, s := range samples {
		t := s.Type
		if t == "" {
			t = "未分类"
		}
		if byType[t] == nil {
			byType[t] = map[string]*ContentItem{}
			statsByType[t] = &GroupStats{}
		}
		st := statsByType[t]
		st.Total++
		switch s.Status {
		case "critical":
			st.Critical++
		case "warning":
			st.Warning++
		default:
			st.Normal++
		}
		key := strings.TrimSpace(s.Name)
		if key == "" {
			key = "未命名项"
		}
		if byType[t][key] == nil {
			byType[t][key] = &ContentItem{Name: key, Description: strings.TrimSpace(s.Description)}
		}
		byType[t][key].SampleCount++
	}
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)
	out := make([]ContentGroup, 0, len(types))
	for _, t := range types {
		items := make([]ContentItem, 0, len(byType[t]))
		for _, it := range byType[t] {
			items = append(items, *it)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		out = append(out, ContentGroup{Type: t, Items: items, Stats: *statsByType[t]})
	}
	return out
}

func filterSamples(samples []MetricSample, mode string) []MetricSample {
	switch mode {
	case "all":
		return samples
	case "summary", "abnormal_only":
		out := make([]MetricSample, 0, len(samples))
		for _, s := range samples {
			if s.Status == "critical" || s.Status == "warning" || s.Error != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return samples
	}
}

func scorecard(c CollectResult) (float64, string) {
	if c.Total == 0 {
		return 100, "A"
	}
	// 严重扣 8 分、警告扣 3 分，下限 0
	score := 100.0 - float64(c.Critical)*8 - float64(c.Warning)*3
	if score < 0 {
		score = 0
	}
	score = math.Round(score*10) / 10
	grade := "A"
	switch {
	case score < 60:
		grade = "D"
	case score < 75:
		grade = "C"
	case score < 90:
		grade = "B"
	}
	return score, grade
}

func buildFindings(samples []MetricSample) []Finding {
	type key struct{ t, n, sev string }
	counts := map[key]int{}
	noData := map[key]int{}
	for _, s := range samples {
		if s.Status != "critical" && s.Status != "warning" {
			continue
		}
		k := key{t: s.Type, n: s.Name, sev: s.Status}
		counts[k]++
		if s.Error != "" && (strings.Contains(s.Error, "无返回样本") || s.Instance == "无数据") {
			noData[k]++
		}
	}
	out := make([]Finding, 0, len(counts))
	for k, n := range counts {
		hint := "请核查相关实例指标与阈值配置。"
		if k.sev == "critical" {
			hint = "建议立即排查并处理，必要时扩容或修复故障。"
		}
		if noData[k] > 0 && noData[k] >= n {
			hint = "采集无数据：请核对 Prometheus 指标名 / job 是否与 Telegraf、Blackbox 一致。"
		}
		out = append(out, Finding{Type: k.t, Name: k.n, Severity: k.sev, Count: n, Hint: hint})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity < out[j].Severity // critical before warning alphabetically? critical < warning
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func renderHTML(data ReportData) ([]byte, error) {
	return renderHTMLWithTemplate("default", "", data)
}
