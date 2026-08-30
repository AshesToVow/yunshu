package inspect

import (
	"embed"
	"html/template"
	"math"
	"sort"
	"strings"
	"time"
)

//go:embed templates/*.html
var reportFS embed.FS

// ReportData HTML 报告数据。
type ReportData struct {
	Timestamp       time.Time
	Project         string
	Datasource      string
	InspectionUser  string
	Score           float64
	Grade           string
	GradeLabel      string // 优秀/良好/需关注/高风险
	RiskLevel       string // 低风险…高风险
	Summary         string
	Verdict         string // 面向客户的一句话结论
	ReportListMode  string
	ReportListHint  string
	Total           int
	Critical        int
	Warning         int
	Normal          int
	CategoryCount   int // 检查分类数
	CheckItemCount  int // 去重检查项数
	DistCriticalPct int
	DistWarningPct  int
	DistNormalPct   int
	AbnormalPct     int
	DistBarSVG      template.HTML  // 预渲染分布条，避免模板内联 style 触发 HTML 校验报错
	ContentGroups   []ContentGroup // 巡检内容（全量范围）
	Groups          []ReportGroup  // 各类巡检结果（受 listMode 过滤）
	Findings        []Finding

	// Criteria 评分与等级的判定依据，回答客户「这个分怎么来的」。
	Criteria ScoreCriteria
	// Ledger 风险台账（含责任人/期限/期次状态），报告的整改闭环载体。
	// buildReportData 阶段 State 全为 new，由 performRun 调 applyPeriodDiff 回填。
	Ledger []LedgerEntry
	// Diff 与上期的对比结论。HasBaseline=false 时模板不渲染该章节。
	Diff PeriodDiff
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
	categoryCount := countCheckCategories(collected.Samples)
	checkItemCount := countCheckItems(collected.Samples)
	hint := "展示全部样本"
	switch mode {
	case "abnormal_only":
		hint = "明细区仅展示异常样本（严重/警告），便于客户汇报阅读"
	case "summary":
		hint = "摘要模式：明细区仅展示需关注的异常样本"
	}
	return ReportData{
		Timestamp:       time.Now(),
		Project:         projectName,
		Datasource:      dsName,
		InspectionUser:  user,
		Score:           score,
		Grade:           grade,
		GradeLabel:      gradeLabelCN(grade),
		RiskLevel:       riskLevelCN(grade),
		Summary:         buildExecutiveSummary(collected, score, grade, categoryCount, checkItemCount),
		Verdict:         gradeVerdict(grade, collected.Critical, collected.Warning),
		ReportListMode:  mode,
		ReportListHint:  hint,
		Total:           collected.Total,
		Critical:        collected.Critical,
		Warning:         collected.Warning,
		Normal:          collected.Normal,
		CategoryCount:   categoryCount,
		CheckItemCount:  checkItemCount,
		DistCriticalPct: percentPart(collected.Critical, collected.Total),
		DistWarningPct:  percentPart(collected.Warning, collected.Total),
		DistNormalPct:   percentPart(collected.Normal, collected.Total),
		AbnormalPct:     percentPart(collected.Critical+collected.Warning, collected.Total),
		DistBarSVG:      buildDistBarSVG(collected.Critical, collected.Warning, collected.Normal),
		ContentGroups:   contentGroups,
		Groups:          groups,
		Findings:        findings,
		Criteria:        buildScoreCriteria(collected, score, grade, categoryCount, checkItemCount),
		Ledger:          buildLedgerEntries(collected.Samples),
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

// scorecard 计算健康分与等级。扣分权重与等级阈值统一取 report_criteria.go 的常量，
// 避免这里改了权重而报告「判定依据」章节仍展示旧规则（两处曾各自硬编码 8/3/100）。
func scorecard(c CollectResult) (float64, string) {
	if c.Total == 0 {
		return scoreBase, "A"
	}
	score := scoreBase - float64(c.Critical)*scoreCriticalDeduct - float64(c.Warning)*scoreWarningDeduct
	if score < 0 {
		score = 0
	}
	score = math.Round(score*10) / 10
	grade := "A"
	switch {
	case score < gradeThresholdD:
		grade = "D"
	case score < gradeThresholdC:
		grade = "C"
	case score < gradeThresholdB:
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
		hint := "请核查相关实例指标与阈值配置，并在下次巡检前完成复核。"
		if k.sev == "critical" {
			hint = "建议立即组织排查并处理，必要时扩容、切换或修复故障；处理结果请回填至运维台账。"
		}
		if noData[k] > 0 && noData[k] >= n {
			hint = "监控数据未采集到有效样本，请核对 Prometheus 指标名、job 标签是否与 Telegraf / Blackbox 配置一致。"
		}
		out = append(out, Finding{Type: k.t, Name: k.n, Severity: k.sev, Count: n, Hint: hint})
	}
	sort.Slice(out, func(i, j int) bool {
		si, sj := severityRank(out[i].Severity), severityRank(out[j].Severity)
		if si != sj {
			return si < sj
		}
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Name < out[j].Name
	})
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func renderHTML(data ReportData) ([]byte, error) {
	return renderHTMLWithTemplate("default", "", data)
}
