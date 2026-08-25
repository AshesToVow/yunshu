package inspect

import (
	"fmt"
	"html/template"
	"strings"
)

func gradeLabelCN(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "A":
		return "优秀"
	case "B":
		return "良好"
	case "C":
		return "需关注"
	default:
		return "高风险"
	}
}

func riskLevelCN(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "A":
		return "低风险"
	case "B":
		return "中低风险"
	case "C":
		return "中高风险"
	default:
		return "高风险"
	}
}

// gradeVerdict 面向客户汇报的健康结论（一句话）。
func gradeVerdict(grade string, critical, warning int) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "A":
		if critical == 0 && warning == 0 {
			return "整体运行稳定，核心指标处于健康区间，建议保持当前巡检频率与监控策略。"
		}
		return "整体健康良好，存在少量需关注项，建议在下次例会前完成复核并纳入观察清单。"
	case "B":
		return "整体运行可控，但存在需优化项，建议安排专项排查并在两周内反馈整改进度。"
	case "C":
		return "已出现明显风险信号，建议本周内组织联合排查，制定整改计划并向客户同步进展。"
	default:
		return "存在较高运行风险，建议立即启动应急响应流程，优先处理严重项并同步客户相关负责人。"
	}
}

func buildExecutiveSummary(c CollectResult, score float64, grade string, categoryCount, checkItemCount int) string {
	return fmt.Sprintf(
		"本次巡检覆盖 %d 个检查分类、%d 项检查指标，共采集 %d 条样本。结果分布为：严重 %d 项、警告 %d 项、正常 %d 项；综合健康分 %.0f，评定等级 %s（%s）。",
		categoryCount, checkItemCount, c.Total,
		c.Critical, c.Warning, c.Normal,
		score, grade, gradeLabelCN(grade),
	)
}

func countCheckCategories(samples []MetricSample) int {
	set := map[string]struct{}{}
	for _, s := range samples {
		t := strings.TrimSpace(s.Type)
		if t == "" {
			t = "未分类"
		}
		set[t] = struct{}{}
	}
	return len(set)
}

func countCheckItems(samples []MetricSample) int {
	set := map[string]struct{}{}
	for _, s := range samples {
		key := strings.TrimSpace(s.Type) + "\x00" + strings.TrimSpace(s.Name)
		if strings.TrimSpace(s.Name) == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return len(set)
}

func severityRank(sev string) int {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}

func findingPriorityLabel(sev string) string {
	if strings.EqualFold(strings.TrimSpace(sev), "critical") {
		return "P0"
	}
	return "P1"
}

func percentPart(part, total int) int {
	if total <= 0 || part <= 0 {
		return 0
	}
	return int(float64(part)/float64(total)*100 + 0.5)
}

// buildDistBarSVG 生成结果分布条 SVG（在 Go 侧计算宽度，模板文件保持合法 HTML）。
func buildDistBarSVG(critical, warning, normal int) template.HTML {
	total := critical + warning + normal
	if total <= 0 {
		return ""
	}
	cw := percentPart(critical, total)
	ww := percentPart(warning, total)
	nw := percentPart(normal, total)
	x := 0
	var rects strings.Builder
	appendRect := func(w int, color string) {
		if w <= 0 {
			return
		}
		fmt.Fprintf(&rects, `<rect x="%d" width="%d" height="8" fill="%s" rx="1"/>`, x, w, color)
		x += w
	}
	appendRect(cw, "#dc2626")
	appendRect(ww, "#d97706")
	appendRect(nw, "#16a34a")
	_ = x // x tracks layout
	svg := fmt.Sprintf(
		`<svg class="dist-bar-svg" viewBox="0 0 100 8" preserveAspectRatio="none" aria-hidden="true" role="img">%s</svg>`,
		rects.String(),
	)
	return template.HTML(svg)
}
