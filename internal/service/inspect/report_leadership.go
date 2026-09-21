package inspect

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// AssetBucket 报告首页/统计页的资产大类（按实际巡检 Type 动态生成，不写死）。
type AssetBucket struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Total    int     `json:"total"`
	Normal   int     `json:"normal"`
	Abnormal int     `json:"abnormal"`
	Rate     float64 `json:"rate"` // 正常率 0-100
	Kind     string  `json:"kind"` // host/db/middleware/k8s/security/app/other（仅辅助展示）
}

// CategorySection 某一巡检分类的明细页（有哪些分类就生成哪些页）。
type CategorySection struct {
	Key      string
	Title    string
	Kind     string
	Total    int
	Normal   int
	Abnormal int
	Rate     float64
	Items    []ContentItem
	Metrics  []MetricSample
	HostAvg  *HostResourceAvg
	HostRows []HostRow
}

// RiskTierSummary 风险等级汇总（用于风险页 / 总结页）。
type RiskTierSummary struct {
	Severe     int
	High       int
	Medium     int
	Low        int
	SevereDone int
	HighDone   int
	MediumDone int
	LowDone    int
	SevereOpen int
	HighOpen   int
	MediumOpen int
	LowOpen    int
	TotalOpen  int
	TotalDone  int
}

// HostResourceAvg 主机类资源均值。
type HostResourceAvg struct {
	CPUPct  float64
	MemPct  float64
	DiskPct float64
	HasCPU  bool
	HasMem  bool
	HasDisk bool
	Hosts   int
}

// HostRow 主机实例摘要行。
type HostRow struct {
	Instance string
	CPU      string
	Mem      string
	Disk     string
	Status   string
}

// categoryKind 仅用于可选控件（如主机 CPU/内存卡），不决定是否出页。
func categoryKind(typ string) string {
	t := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case strings.Contains(t, "基础") || strings.Contains(t, "主机") || strings.Contains(t, "服务器") ||
		strings.Contains(t, "infra") || strings.Contains(t, "host"):
		return "host"
	case strings.Contains(t, "数据库") || strings.Contains(t, "mysql") || strings.Contains(t, "postgres") ||
		strings.Contains(t, "mongo") || t == "db":
		return "db"
	case strings.Contains(t, "k8s") || strings.Contains(t, "kube") || strings.Contains(t, "容器"):
		return "k8s"
	case strings.Contains(t, "中间件") || strings.Contains(t, "redis") || strings.Contains(t, "nginx") ||
		strings.Contains(t, "kafka") || strings.Contains(t, "mq") || strings.Contains(t, "elasticsearch") ||
		strings.Contains(t, "minio") || strings.Contains(t, "tomcat"):
		return "middleware"
	case strings.Contains(t, "安全") || strings.Contains(t, "security") || strings.Contains(t, "ssh") ||
		strings.Contains(t, "漏洞"):
		return "security"
	case strings.Contains(t, "应用") || strings.Contains(t, "jvm") || strings.Contains(t, "app"):
		return "app"
	default:
		return "other"
	}
}

func categoryTitle(typ string) string {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return "未分类"
	}
	return typ
}

func buildAssetBuckets(samples []MetricSample, contentGroups []ContentGroup) []AssetBucket {
	type agg struct {
		label, kind      string
		total, abn, norm int
	}
	byKey := map[string]*agg{}

	abnByTypeItem := map[string]map[string]bool{}
	for _, s := range samples {
		if s.Status != "critical" && s.Status != "warning" {
			continue
		}
		typ := categoryTitle(s.Type)
		if abnByTypeItem[typ] == nil {
			abnByTypeItem[typ] = map[string]bool{}
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = "未命名项"
		}
		abnByTypeItem[typ][name] = true
	}

	if len(contentGroups) > 0 {
		for _, g := range contentGroups {
			typ := categoryTitle(g.Type)
			a := byKey[typ]
			if a == nil {
				a = &agg{label: typ, kind: categoryKind(typ)}
				byKey[typ] = a
			}
			for _, it := range g.Items {
				a.total++
				name := strings.TrimSpace(it.Name)
				if abnByTypeItem[typ][name] {
					a.abn++
				} else {
					a.norm++
				}
			}
		}
	} else {
		instStatus := map[string]map[string]string{}
		for _, s := range samples {
			typ := categoryTitle(s.Type)
			if byKey[typ] == nil {
				byKey[typ] = &agg{label: typ, kind: categoryKind(typ)}
			}
			if instStatus[typ] == nil {
				instStatus[typ] = map[string]string{}
			}
			id := strings.TrimSpace(s.Instance)
			if id == "" || id == "无数据" {
				id = strings.TrimSpace(s.Name)
			}
			prev := instStatus[typ][id]
			if prev == "" || severityRank(s.Status) < severityRank(prev) {
				instStatus[typ][id] = s.Status
			}
		}
		for typ, m := range instStatus {
			a := byKey[typ]
			for _, st := range m {
				a.total++
				if st == "critical" || st == "warning" {
					a.abn++
				} else {
					a.norm++
				}
			}
		}
	}

	// 顺序跟随 ContentGroups；否则按 label 排序，保证稳定
	out := make([]AssetBucket, 0, len(byKey))
	seen := map[string]bool{}
	appendOne := func(typ string, a *agg) {
		if a == nil || a.total == 0 || seen[typ] {
			return
		}
		seen[typ] = true
		rate := 0.0
		if a.total > 0 {
			rate = float64(a.norm) * 100 / float64(a.total)
		}
		out = append(out, AssetBucket{
			Key: typ, Label: a.label, Kind: a.kind,
			Total: a.total, Normal: a.norm, Abnormal: a.abn, Rate: round1(rate),
		})
	}
	for _, g := range contentGroups {
		appendOne(categoryTitle(g.Type), byKey[categoryTitle(g.Type)])
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		appendOne(k, byKey[k])
	}
	return out
}

func buildCategorySections(contentGroups []ContentGroup, groups []ReportGroup, samples []MetricSample, maxMetrics int) []CategorySection {
	if maxMetrics <= 0 {
		maxMetrics = 60
	}
	groupByType := map[string]ReportGroup{}
	for _, g := range groups {
		groupByType[categoryTitle(g.Type)] = g
	}
	// 若 Groups 被 listMode 过滤为空，用全量 samples 兜底
	samplesByType := map[string][]MetricSample{}
	for _, s := range samples {
		typ := categoryTitle(s.Type)
		samplesByType[typ] = append(samplesByType[typ], s)
	}

	sections := make([]CategorySection, 0, len(contentGroups))
	seen := map[string]bool{}
	appendSection := func(typ string, items []ContentItem, st GroupStats) {
		if seen[typ] {
			return
		}
		seen[typ] = true
		kind := categoryKind(typ)
		abn := st.Critical + st.Warning
		norm := st.Normal
		total := st.Total
		if total == 0 && len(items) > 0 {
			// ContentGroup.Stats 是样本级；Assets 用检查项级。明细页 KPI 优先检查项。
			abnItems := 0
			for _, it := range items {
				for _, s := range samplesByType[typ] {
					if strings.TrimSpace(s.Name) == strings.TrimSpace(it.Name) &&
						(s.Status == "critical" || s.Status == "warning") {
						abnItems++
						break
					}
				}
			}
			total = len(items)
			abn = abnItems
			norm = total - abn
		}
		rate := 0.0
		if total > 0 {
			rate = float64(norm) * 100 / float64(total)
		}

		metrics := pickCategoryMetrics(groupByType[typ].Metrics, samplesByType[typ], maxMetrics)
		sec := CategorySection{
			Key: typ, Title: typ, Kind: kind,
			Total: total, Normal: norm, Abnormal: abn, Rate: round1(rate),
			Items: items, Metrics: metrics,
		}
		if kind == "host" {
			avg := buildHostResourceAvg(samplesByType[typ])
			if avg.HasCPU || avg.HasMem || avg.HasDisk || avg.Hosts > 0 {
				sec.HostAvg = &avg
			}
			rows := buildHostRows(samplesByType[typ], 40)
			if len(rows) > 0 {
				sec.HostRows = rows
			}
		}
		sections = append(sections, sec)
	}

	for _, g := range contentGroups {
		appendSection(categoryTitle(g.Type), g.Items, g.Stats)
	}
	// 有样本但 ContentGroups 未覆盖的分类
	keys := make([]string, 0, len(samplesByType))
	for k := range samplesByType {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, typ := range keys {
		if seen[typ] {
			continue
		}
		st := GroupStats{}
		for _, s := range samplesByType[typ] {
			st.Total++
			switch s.Status {
			case "critical":
				st.Critical++
			case "warning":
				st.Warning++
			default:
				st.Normal++
			}
		}
		appendSection(typ, nil, st)
	}
	return sections
}

func pickCategoryMetrics(preferred, fallback []MetricSample, max int) []MetricSample {
	src := preferred
	if len(src) == 0 {
		src = fallback
	}
	if len(src) == 0 {
		return nil
	}
	cp := append([]MetricSample(nil), src...)
	sort.SliceStable(cp, func(i, j int) bool {
		return severityRank(cp[i].Status) < severityRank(cp[j].Status)
	})
	if len(cp) > max {
		cp = cp[:max]
	}
	return cp
}

func buildRiskTierSummary(ledger []LedgerEntry) RiskTierSummary {
	var s RiskTierSummary
	for _, e := range ledger {
		done := e.State == "recovered" || strings.Contains(e.StateLabel, "已整改") || strings.Contains(e.StateLabel, "已恢复")
		high := e.Severity == "critical" || e.Priority == "P0"
		if high {
			s.High++
			if done {
				s.HighDone++
			} else {
				s.HighOpen++
			}
			continue
		}
		s.Medium++
		if done {
			s.MediumDone++
		} else {
			s.MediumOpen++
		}
	}
	s.TotalOpen = s.SevereOpen + s.HighOpen + s.MediumOpen + s.LowOpen
	s.TotalDone = s.SevereDone + s.HighDone + s.MediumDone + s.LowDone
	return s
}

func buildHostResourceAvg(samples []MetricSample) HostResourceAvg {
	var out HostResourceAvg
	var cpuSum, memSum, diskSum float64
	var cpuN, memN, diskN int
	hosts := map[string]struct{}{}
	for _, s := range samples {
		if inst := strings.TrimSpace(s.Instance); inst != "" && inst != "无数据" {
			hosts[inst] = struct{}{}
		}
		name := strings.ToLower(s.Name)
		v := s.Value
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		switch {
		case strings.Contains(name, "cpu"):
			cpuSum += v
			cpuN++
			out.HasCPU = true
		case strings.Contains(name, "内存") || strings.Contains(name, "mem"):
			memSum += v
			memN++
			out.HasMem = true
		case strings.Contains(name, "磁盘") || strings.Contains(name, "disk"):
			diskSum += v
			diskN++
			out.HasDisk = true
		}
	}
	out.Hosts = len(hosts)
	if cpuN > 0 {
		out.CPUPct = round1(cpuSum / float64(cpuN))
	}
	if memN > 0 {
		out.MemPct = round1(memSum / float64(memN))
	}
	if diskN > 0 {
		out.DiskPct = round1(diskSum / float64(diskN))
	}
	return out
}

func buildHostRows(samples []MetricSample, maxRows int) []HostRow {
	type cell struct {
		cpu, mem, disk string
		status         string
	}
	byInst := map[string]*cell{}
	order := make([]string, 0)
	for _, s := range samples {
		inst := strings.TrimSpace(s.Instance)
		if inst == "" || inst == "无数据" {
			continue
		}
		c := byInst[inst]
		if c == nil {
			c = &cell{status: "normal"}
			byInst[inst] = c
			order = append(order, inst)
		}
		if severityRank(s.Status) < severityRank(c.status) {
			c.status = s.Status
		}
		name := strings.ToLower(s.Name)
		val := fmt.Sprintf("%.0f%%", s.Value)
		switch {
		case strings.Contains(name, "cpu"):
			c.cpu = val
		case strings.Contains(name, "内存") || strings.Contains(name, "mem"):
			c.mem = val
		case strings.Contains(name, "磁盘") || strings.Contains(name, "disk"):
			c.disk = val
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		return severityRank(byInst[order[i]].status) < severityRank(byInst[order[j]].status)
	})
	if maxRows <= 0 {
		maxRows = 40
	}
	out := make([]HostRow, 0, min(len(order), maxRows))
	for _, inst := range order {
		if len(out) >= maxRows {
			break
		}
		c := byInst[inst]
		out = append(out, HostRow{
			Instance: inst,
			CPU:      dashOr(c.cpu),
			Mem:      dashOr(c.mem),
			Disk:     dashOr(c.disk),
			Status:   c.status,
		})
	}
	return out
}

func dashOr(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func buildLeadershipConclusion(data *ReportData) string {
	if data == nil {
		return ""
	}
	segs := make([]string, 0, len(data.Assets))
	for _, a := range data.Assets {
		segs = append(segs, fmt.Sprintf("%s %d 项", a.Label, a.Total))
	}
	if len(segs) == 0 {
		segs = append(segs, fmt.Sprintf("检查项 %d 项", data.CheckItemCount))
	}
	head := "本次巡检共覆盖" + strings.Join(segs, "、") + "。"
	riskTotal := data.RiskTiers.High + data.RiskTiers.Medium + data.RiskTiers.Low + data.RiskTiers.Severe
	var mid string
	if riskTotal == 0 {
		mid = "整体运行稳定，未发现需跟进的风险问题。"
	} else {
		mid = fmt.Sprintf("运行评价为「%s」，发现需关注问题%d项（严重%d / 高风险%d / 中风险%d / 低风险%d）。",
			data.GradeLabel, riskTotal, data.RiskTiers.Severe, data.RiskTiers.High, data.RiskTiers.Medium, data.RiskTiers.Low)
	}
	tail := strings.TrimSpace(data.Verdict)
	if tail != "" {
		return head + mid + "建议重点关注：" + tail
	}
	return head + mid
}

func starRating(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "A":
		return "★★★★★"
	case "B":
		return "★★★★☆"
	case "C":
		return "★★★☆☆"
	default:
		return "★★☆☆☆"
	}
}
