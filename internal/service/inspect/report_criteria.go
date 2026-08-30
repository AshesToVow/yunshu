package inspect

import (
	"fmt"
	"math"
)

// 评分规则常量。与 scorecard() 保持单一事实来源：
// scorecard 负责算分，本文件负责把同一套规则讲清楚，避免报告里的「判定依据」与实际算法脱节。
const (
	scoreBase           = 100.0
	scoreCriticalDeduct = 8.0
	scoreWarningDeduct  = 3.0
	gradeThresholdB     = 90.0
	gradeThresholdC     = 75.0
	gradeThresholdD     = 60.0
)

// GradeBand 等级区间说明，用于报告「判定依据」章节的等级对照表。
type GradeBand struct {
	Grade     string
	Range     string
	Label     string
	RiskLevel string
	Action    string
	Current   bool // 是否为本期落入的区间
}

// ScoreStep 扣分明细的一行，让客户看到 100 分是怎么减到最终分的。
type ScoreStep struct {
	Item   string
	Count  int
	Unit   float64
	Deduct float64
}

// ScoreCriteria 评分与等级的判定依据。
type ScoreCriteria struct {
	Base         float64
	CriticalUnit float64
	WarningUnit  float64
	Steps        []ScoreStep
	RawScore     float64 // 未做下限截断的原始分，用于说明是否触发 0 分保底
	FinalScore   float64
	Floored      bool // 扣分超过 100，最终分被截断为 0
	Formula      string
	Bands        []GradeBand
	SeverityNote string
	ScopeNote    string
}

// buildScoreCriteria 依据本期采集结果还原评分过程。
func buildScoreCriteria(c CollectResult, score float64, grade string, categoryCount, checkItemCount int) ScoreCriteria {
	cr := ScoreCriteria{
		Base:         scoreBase,
		CriticalUnit: scoreCriticalDeduct,
		WarningUnit:  scoreWarningDeduct,
		FinalScore:   score,
	}
	if c.Critical > 0 {
		cr.Steps = append(cr.Steps, ScoreStep{
			Item:   "严重项",
			Count:  c.Critical,
			Unit:   scoreCriticalDeduct,
			Deduct: float64(c.Critical) * scoreCriticalDeduct,
		})
	}
	if c.Warning > 0 {
		cr.Steps = append(cr.Steps, ScoreStep{
			Item:   "警告项",
			Count:  c.Warning,
			Unit:   scoreWarningDeduct,
			Deduct: float64(c.Warning) * scoreWarningDeduct,
		})
	}

	total := float64(c.Critical)*scoreCriticalDeduct + float64(c.Warning)*scoreWarningDeduct
	cr.RawScore = math.Round((scoreBase-total)*10) / 10
	cr.Floored = cr.RawScore < 0

	switch {
	case c.Total == 0:
		cr.Formula = "本次未采集到任何样本，按无异常处理，计 100 分。"
	case len(cr.Steps) == 0:
		cr.Formula = fmt.Sprintf("基础分 100，无严重项与警告项，最终得分 %.1f。", score)
	case cr.Floored:
		cr.Formula = fmt.Sprintf(
			"基础分 100 − 严重 %d 项 × 8 − 警告 %d 项 × 3 = %.1f，低于下限按 0 分计。",
			c.Critical, c.Warning, cr.RawScore,
		)
	default:
		cr.Formula = fmt.Sprintf(
			"基础分 100 − 严重 %d 项 × 8 − 警告 %d 项 × 3 = %.1f。",
			c.Critical, c.Warning, score,
		)
	}

	cr.Bands = gradeBands(grade)
	cr.SeverityNote = "严重/警告的判定来自每个检查项自身配置的阈值与比较符：指标值越线即按该项设定的严重度归档；" +
		"监控未返回样本的检查项按「无数据」计入异常，因为监控盲区同样构成运行风险。"
	cr.ScopeNote = fmt.Sprintf(
		"本期评分基于 %d 个检查分类下的 %d 项检查指标、共 %d 条采集样本，扣分以样本条数为单位计算。",
		categoryCount, checkItemCount, c.Total,
	)
	return cr
}

func gradeBands(current string) []GradeBand {
	bands := []GradeBand{
		{Grade: "A", Range: "90 ~ 100", Label: "优秀", RiskLevel: "低风险", Action: "保持现有巡检频率与监控策略"},
		{Grade: "B", Range: "75 ~ 89.9", Label: "良好", RiskLevel: "中低风险", Action: "安排专项优化，两周内反馈整改进度"},
		{Grade: "C", Range: "60 ~ 74.9", Label: "需关注", RiskLevel: "中高风险", Action: "本周内联合排查并制定整改计划"},
		{Grade: "D", Range: "0 ~ 59.9", Label: "高风险", RiskLevel: "高风险", Action: "立即启动应急响应，优先处理严重项"},
	}
	for i := range bands {
		if bands[i].Grade == current {
			bands[i].Current = true
		}
	}
	return bands
}
