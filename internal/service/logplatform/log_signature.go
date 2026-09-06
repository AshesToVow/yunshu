package logplatform

import (
	"regexp"
	"sort"
	"strings"

	"yunshu/internal/pkg/pagination"
)

var (
	reDigits   = regexp.MustCompile(`\d+`)
	reUUIDLike = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	reHexLong  = regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`)
	reIPv4     = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`)
)

type countEntry struct {
	Key   string
	Count int
}

// NormalizeLogSignature 将日志消息归一化为模板签名（UUID/IP/数字替换）。
func NormalizeLogSignature(msg string) string {
	s := strings.TrimSpace(msg)
	if s == "" {
		return ""
	}
	s = reUUIDLike.ReplaceAllString(s, "<uuid>")
	s = reHexLong.ReplaceAllString(s, "<hex>")
	s = reIPv4.ReplaceAllString(s, "<ip>")
	s = reDigits.ReplaceAllString(s, "N")
	s = strings.Join(strings.Fields(s), " ")
	return truncateRunes(s, 160)
}

func inferLevelFromMessage(msg string) string {
	u := strings.ToUpper(msg)
	for _, lv := range []string{"FATAL", "ERROR", "WARN", "WARNING", "INFO", "DEBUG", "TRACE"} {
		if strings.Contains(u, lv) {
			if lv == "WARNING" {
				return "WARN"
			}
			return lv
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func topCountEntries(m map[string]int, n int) []countEntry {
	out := make([]countEntry, 0, len(m))
	for k, v := range m {
		out = append(out, countEntry{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

func topCountMap(m map[string]int, n int) map[string]int {
	entries := topCountEntries(m, n)
	out := make(map[string]int, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Count
	}
	return out
}

// LogSignatureItem 高频错误签名条目。
type LogSignatureItem struct {
	Signature string `json:"signature"`
	Count     int    `json:"count"`
	Sample    string `json:"sample"`
	Level     string `json:"level,omitempty"`
}

// LogSummaryResult 日志采样整理结果（供 UI / AI 复用）。
type LogSummaryResult struct {
	Total              int64              `json:"total"`
	Sampled            int                `json:"sampled"`
	LevelCounts        map[string]int     `json:"level_counts"`
	ServiceCounts      map[string]int     `json:"service_counts"`
	PodCounts          map[string]int     `json:"pod_counts"`
	TopErrorSignatures []LogSignatureItem `json:"top_error_signatures"`
	Samples            []LogSearchItem    `json:"samples"`
}

// SummarizeLogHits 从检索结果提取统计与签名（与 AI analyze_logs 共用）。
func SummarizeLogHits(res *pagination.Result[LogSearchItem], maxSamples int) *LogSummaryResult {
	if res == nil {
		return &LogSummaryResult{LevelCounts: map[string]int{}}
	}
	if maxSamples <= 0 {
		maxSamples = 8
	}
	list := res.List
	levelCnt := map[string]int{}
	svcCnt := map[string]int{}
	podCnt := map[string]int{}
	sigCnt := map[string]int{}
	sigSample := map[string]string{}
	sigLevel := map[string]string{}
	samples := make([]LogSearchItem, 0, maxSamples)

	for i, it := range list {
		lv := strings.TrimSpace(it.Level)
		if lv == "" {
			lv = inferLevelFromMessage(it.Message)
		}
		if lv == "" {
			lv = "(unknown)"
		}
		levelCnt[lv]++
		svc := strings.TrimSpace(it.ServiceName)
		if svc == "" {
			svc = "(unknown)"
		}
		svcCnt[svc]++
		pod := firstNonEmpty(it.Pod, it.PodName)
		if pod == "" {
			pod = "(unknown)"
		}
		podCnt[pod]++

		sig := NormalizeLogSignature(it.Message)
		if sig != "" {
			sigCnt[sig]++
			if _, ok := sigSample[sig]; !ok {
				sigSample[sig] = truncateRunes(it.Message, 240)
				sigLevel[sig] = lv
			}
		}
		if i < maxSamples {
			samples = append(samples, it)
		}
	}

	topSigs := topCountEntries(sigCnt, 10)
	topSigOut := make([]LogSignatureItem, 0, len(topSigs))
	for _, e := range topSigs {
		topSigOut = append(topSigOut, LogSignatureItem{
			Signature: e.Key,
			Count:     e.Count,
			Sample:    sigSample[e.Key],
			Level:     sigLevel[e.Key],
		})
	}

	return &LogSummaryResult{
		Total:              res.Total,
		Sampled:            len(list),
		LevelCounts:        levelCnt,
		ServiceCounts:      topCountMap(svcCnt, 8),
		PodCounts:          topCountMap(podCnt, 8),
		TopErrorSignatures: topSigOut,
		Samples:            samples,
	}
}
