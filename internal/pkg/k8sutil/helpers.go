package k8sutil

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var logTSRegexp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}Z?`)
var applyLooksOkRegexp = regexp.MustCompile(`(?i)\b(created|configured|unchanged|updated)\b`)

func SplitYAMLDocs(manifest string) []string {
	scanner := bufio.NewScanner(strings.NewReader(manifest))
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var docs []string
	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			docs = append(docs, buf.String())
			buf.Reset()
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	docs = append(docs, buf.String())
	return docs
}

func HumanAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func Deref32(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

func IsLikelyText(b []byte) bool {
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}

func ParseAnyTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

func ExtractLineTimestamp(line string) (time.Time, bool) {
	match := logTSRegexp.FindString(line)
	if match == "" {
		return time.Time{}, false
	}
	if t, err := ParseAnyTime(match); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func FilterLogLines(text, keyword, startStr, endStr string) string {
	lines := strings.Split(text, "\n")
	kw := strings.ToLower(strings.TrimSpace(keyword))
	start, _ := ParseAnyTime(startStr)
	end, _ := ParseAnyTime(endStr)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if kw != "" && !strings.Contains(strings.ToLower(line), kw) {
			continue
		}
		if !start.IsZero() || !end.IsZero() {
			ts, ok := ExtractLineTimestamp(line)
			if ok {
				if !start.IsZero() && ts.Before(start) {
					continue
				}
				if !end.IsZero() && ts.After(end) {
					continue
				}
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// IsLikelySuccessfulApplyError 兼容 kom/kubectl apply 成功仍返回非 nil 的窄场景。
// 仅当整段输出每行都像「Kind/name created|configured|unchanged」时才视为成功，避免吞掉真实错误。
func IsLikelySuccessfulApplyError(v any) bool {
	var s string
	switch x := v.(type) {
	case nil:
		return false
	case error:
		s = strings.TrimSpace(x.Error())
	case []string:
		s = strings.TrimSpace(strings.Join(x, "\n"))
	case string:
		s = strings.TrimSpace(x)
	default:
		return false
	}
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	for _, bad := range []string{"error", "forbidden", "unauthorized", "denied", "failed", "invalid", "timeout"} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	lines := strings.Split(s, "\n")
	okLine := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "/") || !applyLooksOkRegexp.MatchString(line) {
			return false
		}
		okLine++
	}
	return okLine > 0
}
