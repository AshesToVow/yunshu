package inspect

import "strings"

// sampleNoteText 将原始采集错误压缩为报告可读提示（compact 模板等复用）。
func sampleNoteText(err string) string {
	err = strings.TrimSpace(err)
	if err == "" {
		return ""
	}
	switch {
	case strings.Contains(err, "无返回样本"):
		return "采集无数据：请核对指标名 / job 是否与 Telegraf、Blackbox 一致"
	case strings.Contains(err, "NaN") || strings.Contains(err, "Inf"):
		return "样本无效（NaN/Inf）"
	case strings.Contains(err, "empty query"):
		return "巡检项查询为空"
	default:
		return truncateRunes(err, 80)
	}
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
