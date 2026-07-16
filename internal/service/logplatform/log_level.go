package logplatform

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	levelBracketRE = regexp.MustCompile(`(?i)\[(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|PANIC)\s*\]`)
	levelTokenRE   = regexp.MustCompile(`(?i)\s(TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|PANIC)\s`)
	// klog / kube-style：I0716 02:51:52.902837 或行首 I/W/E/F + MMDD
	levelKlogRE = regexp.MustCompile(`(?:^|[\s>])([IWEF])\d{4}\s+\d{2}:\d{2}:\d{2}`)
)

func normalizeLevel(level string) string {
	level = strings.TrimSpace(strings.ToUpper(level))
	switch level {
	case "WARNING":
		return "WARN"
	case "I":
		return "INFO"
	case "W":
		return "WARN"
	case "E":
		return "ERROR"
	case "F":
		return "FATAL"
	default:
		return level
	}
}

func extractLevelFromMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if m := levelBracketRE.FindStringSubmatch(message); len(m) > 1 {
		return normalizeLevel(m[1])
	}
	if m := levelTokenRE.FindStringSubmatch(message); len(m) > 1 {
		return normalizeLevel(m[1])
	}
	if m := levelKlogRE.FindStringSubmatch(message); len(m) > 1 {
		return normalizeLevel(m[1])
	}
	return ""
}

func levelFilter(level string) map[string]any {
	level = normalizeLevel(level)
	if level == "" {
		return nil
	}
	should := make([]map[string]any, 0, 12)
	for _, field := range []string{"level", "log.level", "fields.level"} {
		should = append(should,
			map[string]any{"term": map[string]any{field + ".keyword": level}},
			map[string]any{"term": map[string]any{field: level}},
		)
	}
	for _, clause := range levelMessageClauses(level) {
		should = append(should, clause)
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func levelMessageClauses(level string) []map[string]any {
	bracket := fmt.Sprintf("[%s ]", level)
	if level == "WARN" {
		bracket = "[WARN ]"
	}
	return []map[string]any{
		{"wildcard": map[string]any{"message": fmt.Sprintf("*%s*", bracket)}},
		{"wildcard": map[string]any{"message": fmt.Sprintf("*[%s]*", level)}},
		{"match_phrase": map[string]any{"message": level}},
	}
}

func filePathFilter(path string) map[string]any {
	path = strings.TrimSpace(path)
	pattern := "*" + path + "*"
	should := []map[string]any{
		{"wildcard": map[string]any{"file_path.keyword": pattern}},
		{"wildcard": map[string]any{"file_path": pattern}},
		{"wildcard": map[string]any{"log_file.keyword": pattern}},
		{"wildcard": map[string]any{"log_file": pattern}},
		{"wildcard": map[string]any{"state.filename.keyword": pattern}},
		{"wildcard": map[string]any{"state.filename": pattern}},
		{"match_phrase": map[string]any{"file_path": path}},
	}
	return map[string]any{
		"bool": map[string]any{
			"should":               should,
			"minimum_should_match": 1,
		},
	}
}

func pickLevel(src map[string]any) string {
	meta := nestedFields(src)
	if lv := normalizeLevel(pickString(src, "level", "log.level", "severity", "klevel")); lv != "" {
		return lv
	}
	if meta != nil {
		if lv := normalizeLevel(pickString(meta, "level", "klevel")); lv != "" {
			return lv
		}
	}
	return extractLevelFromMessage(pickString(src, "message", "body", "log", "msg"))
}

func pickFilePath(src map[string]any) string {
	if fp := pickString(src, "file_path", "filepath", "filename", "path", "systemSource", "log.file.path", "log_file"); fp != "" {
		return fp
	}
	if state, ok := src["state"].(map[string]any); ok {
		if fp := pickString(state, "filename"); fp != "" {
			return fp
		}
	}
	if logObj, ok := src["log"].(map[string]any); ok {
		if fileObj, ok := logObj["file"].(map[string]any); ok {
			if fp := pickString(fileObj, "path"); fp != "" {
				return fp
			}
		}
	}
	if beatObj, ok := src["beat"].(map[string]any); ok {
		if fp := pickString(beatObj, "source"); fp != "" {
			return fp
		}
	}
	return ""
}
