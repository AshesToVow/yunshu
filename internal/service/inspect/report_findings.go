package inspect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	// ledgerMaxEntries 风险清单展示上限。超出部分不入台账，避免报告退化成指标转储。
	ledgerMaxEntries = 50
	// ledgerMaxInstances 单条风险展示的实例数上限，超出以「等 N 个」收敛。
	ledgerMaxInstances = 8
	// ledgerMaxServices 单条风险展示的受影响服务数上限。
	ledgerMaxServices = 3
)

// LedgerEntry 风险台账条目：报告「风险清单」章节与整改闭环的展示单元。
//
// 与 Finding 的区别：Finding 只有 类型/名称/数量/提示，是给运维看的技术聚合；
// LedgerEntry 补齐了受影响服务、现象、业务影响、责任人、期限、期次状态，
// 目标是让客户方管理者能直接据此派活与验收。
type LedgerEntry struct {
	Seq             int
	Fingerprint     string
	Type            string
	Name            string
	Severity        string
	Priority        string // P0/P1
	Count           int
	AffectedService string
	Instances       string
	Phenomenon      string
	Impact          string
	Suggestion      string
	State           string // new|persisting|recovered
	StateLabel      string
	Owner           string
	DueDate         *time.Time
	DueDateText     string
	FirstSeenAt     time.Time
	DurationDays    int
}

// findingFingerprint 由 分类+名称 生成跨期次稳定的风险指纹。
// 不含实例，因此「同一检查项换了实例」仍视为同一条风险，符合整改跟踪的粒度。
func findingFingerprint(typ, name string) string {
	key := strings.ToLower(strings.TrimSpace(typ)) + "\x00" + strings.ToLower(strings.TrimSpace(name))
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])[:32]
}

type ledgerAgg struct {
	entry     LedgerEntry
	instances []string
	services  []string
	peak      MetricSample
	hasPeak   bool
	noData    int
}

// buildLedgerEntries 将异常样本聚合为风险台账条目（State 统一为 new，
// 期次状态与责任人由 persistFindings 结合上一期结果回填）。
func buildLedgerEntries(samples []MetricSample) []LedgerEntry {
	byFP := map[string]*ledgerAgg{}
	order := make([]string, 0, 16)

	for _, s := range samples {
		if s.Status != "critical" && s.Status != "warning" {
			continue
		}
		typ := strings.TrimSpace(s.Type)
		if typ == "" {
			typ = "未分类"
		}
		name := strings.TrimSpace(s.Name)
		if name == "" {
			name = "未命名项"
		}
		fp := findingFingerprint(typ, name)
		a := byFP[fp]
		if a == nil {
			a = &ledgerAgg{entry: LedgerEntry{
				Fingerprint: fp,
				Type:        typ,
				Name:        name,
				Severity:    s.Status,
			}}
			byFP[fp] = a
			order = append(order, fp)
		}
		// 同一检查项跨实例出现不同严重度时，按最高严重度归档
		if severityRank(s.Status) < severityRank(a.entry.Severity) {
			a.entry.Severity = s.Status
		}
		a.entry.Count++
		if inst := strings.TrimSpace(s.Instance); inst != "" && inst != "无数据" {
			a.instances = append(a.instances, inst)
		}
		if svc := affectedServiceOf(s); svc != "" {
			a.services = append(a.services, svc)
		}
		if isNoDataSample(s) {
			a.noData++
		}
		if !a.hasPeak || sampleDeviation(s) > sampleDeviation(a.peak) {
			a.peak = s
			a.hasPeak = true
		}
	}

	out := make([]LedgerEntry, 0, len(byFP))
	for _, fp := range order {
		a := byFP[fp]
		e := a.entry
		allNoData := e.Count > 0 && a.noData >= e.Count

		e.Priority = findingPriorityLabel(e.Severity)
		e.Instances = joinCapped(dedupeStrings(a.instances), ledgerMaxInstances)
		e.AffectedService = joinCapped(dedupeStrings(a.services), ledgerMaxServices)
		if e.AffectedService == "" {
			e.AffectedService = "未能从指标标签识别，建议补充实例归属信息"
		}
		e.Phenomenon = ledgerPhenomenon(a.peak, a.hasPeak, allNoData, e.Count)
		e.Impact = ledgerImpact(e.Type, e.Severity, allNoData)
		e.Suggestion = ledgerSuggestion(e.Severity, allNoData)
		e.State = "new"
		e.StateLabel = ledgerStateLabel(e.State)
		out = append(out, e)
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
	if len(out) > ledgerMaxEntries {
		out = out[:ledgerMaxEntries]
	}
	for i := range out {
		out[i].Seq = i + 1
	}
	return out
}

// affectedServiceOf 从指标标签推导「受影响的业务对象」。
// 按 service > app > application > job > namespace > cluster 取第一个命中的标签，
// 目的是把 PromQL 视角的 instance 翻译成客户能对应到人的服务名。
func affectedServiceOf(s MetricSample) string {
	for _, k := range []string{"service", "app", "application", "job", "namespace", "cluster"} {
		if v := strings.TrimSpace(s.Labels[k]); v != "" {
			return v
		}
	}
	return ""
}

func isNoDataSample(s MetricSample) bool {
	if strings.TrimSpace(s.Instance) == "无数据" {
		return true
	}
	return s.Error != "" && strings.Contains(s.Error, "无返回样本")
}

// sampleDeviation 偏离度，用于在同一检查项的多个实例中挑出最有代表性的「峰值样本」。
func sampleDeviation(s MetricSample) float64 {
	if s.Threshold != 0 {
		return math.Abs(s.Value-s.Threshold) / math.Abs(s.Threshold)
	}
	return math.Abs(s.Value)
}

func ledgerPhenomenon(peak MetricSample, hasPeak, allNoData bool, count int) string {
	if allNoData {
		return fmt.Sprintf("监控未采集到有效样本，共 %d 个检查目标无数据返回。", count)
	}
	if !hasPeak {
		return fmt.Sprintf("共 %d 个检查目标触发告警条件。", count)
	}
	unit := strings.TrimSpace(peak.Unit)
	valText := formatMetricValue(peak.Value, unit)
	if peak.Threshold == 0 {
		return fmt.Sprintf("峰值 %s（共 %d 个目标越线）。", valText, count)
	}
	return fmt.Sprintf("峰值 %s，阈值 %s（共 %d 个目标越线）。",
		valText, formatMetricValue(peak.Threshold, unit), count)
}

func formatMetricValue(v float64, unit string) string {
	var num string
	if math.Abs(v-math.Round(v)) < 1e-9 {
		num = fmt.Sprintf("%.0f", v)
	} else {
		num = fmt.Sprintf("%.2f", v)
	}
	if unit == "" {
		return num
	}
	return num + unit
}

func ledgerImpact(typ, severity string, allNoData bool) string {
	if allNoData {
		return "该项处于监控盲区，异常发生时无法及时发现，影响故障响应时效。"
	}
	scope := impactScopeOf(typ)
	if severity == "critical" {
		return scope + "已进入高风险区间，可能造成请求超时、处理能力下降或服务不可用。"
	}
	return scope + "指标已偏离健康区间，当前尚可承载，但存在继续恶化的趋势。"
}

// impactScopeOf 按巡检分类给出影响面描述，让「业务影响」列不至于千篇一律。
func impactScopeOf(typ string) string {
	t := strings.ToLower(typ)
	switch {
	case strings.Contains(t, "数据库") || strings.Contains(t, "mysql") || strings.Contains(t, "redis"):
		return "数据层读写能力"
	case strings.Contains(t, "网络") || strings.Contains(t, "network") || strings.Contains(t, "拨测"):
		return "网络连通与访问链路"
	case strings.Contains(t, "存储") || strings.Contains(t, "磁盘") || strings.Contains(t, "disk"):
		return "存储容量与落盘能力"
	case strings.Contains(t, "容器") || strings.Contains(t, "k8s") || strings.Contains(t, "kubernetes") || strings.Contains(t, "pod"):
		return "容器编排与工作负载调度"
	case strings.Contains(t, "中间件") || strings.Contains(t, "kafka") || strings.Contains(t, "mq"):
		return "消息与中间件处理能力"
	case strings.Contains(t, "应用") || strings.Contains(t, "业务"):
		return "业务应用服务质量"
	default:
		return "基础资源承载能力"
	}
}

func ledgerSuggestion(severity string, allNoData bool) string {
	if allNoData {
		return "核对 Prometheus 指标名与 job 标签是否与 Telegraf / Blackbox 采集配置一致，恢复采集后复核。"
	}
	if severity == "critical" {
		return "本周内组织排查，按需扩容、限流或修复故障，并在下次巡检前回填处理结论。"
	}
	return "纳入观察清单，核查阈值配置合理性与近期变更，两周内反馈复核结果。"
}

func ledgerStateLabel(state string) string {
	switch state {
	case "persisting":
		return "持续"
	case "recovered":
		return "已恢复"
	default:
		return "新增"
	}
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// joinCapped 以「、」拼接，超过 max 时收敛为「A、B、C 等 N 个」，避免表格被实例列表撑爆。
func joinCapped(in []string, max int) string {
	if len(in) == 0 {
		return ""
	}
	if len(in) <= max {
		return strings.Join(in, "、")
	}
	return fmt.Sprintf("%s 等 %d 个", strings.Join(in[:max], "、"), len(in))
}
