package logplatform

import (
	"strings"
	"unicode"
)

// ApplySimplifiedDQL 解析简化 DQL，将 field:value 写入结构化筛选，残留写入 Keyword。
//
// 支持：
//   - level:ERROR service:api host:node-1 pod:x namespace:ns container:c trace_id:abc
//   - AND 连接（大小写不敏感）；OR 暂按空格拼接为残留文本
//   - 引号值："my service"
//
// 已有表单字段优先：仅当目标字段为空时才用 DQL 填充。
func ApplySimplifiedDQL(q *LogSearchQuery) {
	if q == nil {
		return
	}
	raw := strings.TrimSpace(q.Keyword)
	if raw == "" || !looksLikeDQL(raw) {
		return
	}
	tokens := splitDQLTokens(raw)
	var residual []string
	for _, tok := range tokens {
		if strings.EqualFold(tok, "AND") {
			continue
		}
		if strings.EqualFold(tok, "OR") {
			residual = append(residual, tok)
			continue
		}
		field, value, ok := splitFieldValue(tok)
		if !ok {
			residual = append(residual, tok)
			continue
		}
		if !applyDQLField(q, field, value) {
			// 未知字段：走 ExtraField（仅首次）
			if strings.TrimSpace(q.ExtraField) == "" {
				q.ExtraField = field
				q.ExtraValue = value
			} else {
				residual = append(residual, tok)
			}
		}
	}
	q.Keyword = strings.TrimSpace(strings.Join(residual, " "))
}

func looksLikeDQL(s string) bool {
	for _, r := range s {
		if r == ':' {
			return true
		}
	}
	return false
}

func splitDQLTokens(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	quote := rune(0)
	flush := func() {
		if b.Len() == 0 {
			return
		}
		out = append(out, b.String())
		b.Reset()
	}
	for _, r := range s {
		if inQuote {
			if r == quote {
				inQuote = false
				continue
			}
			b.WriteRune(r)
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = true
			quote = r
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return out
}

func splitFieldValue(tok string) (field, value string, ok bool) {
	i := strings.IndexByte(tok, ':')
	if i <= 0 || i == len(tok)-1 {
		return "", "", false
	}
	field = strings.TrimSpace(tok[:i])
	value = strings.TrimSpace(tok[i+1:])
	if field == "" || value == "" {
		return "", "", false
	}
	// 拒绝纯数字时间戳误判等：字段名需字母/下划线开头
	r := rune(field[0])
	if !unicode.IsLetter(r) && r != '_' {
		return "", "", false
	}
	return field, value, true
}

func applyDQLField(q *LogSearchQuery, field, value string) bool {
	switch strings.ToLower(field) {
	case "level", "status", "severity":
		if strings.TrimSpace(q.Level) == "" {
			q.Level = value
		}
		return true
	case "service", "service_name", "svc":
		if strings.TrimSpace(q.ServiceName) == "" {
			q.ServiceName = value
		}
		return true
	case "host", "hostname", "server_host":
		if strings.TrimSpace(q.Host) == "" {
			q.Host = value
		}
		return true
	case "namespace", "ns":
		if strings.TrimSpace(q.Namespace) == "" {
			q.Namespace = value
		}
		return true
	case "pod", "podname", "pod_name":
		if strings.TrimSpace(q.Pod) == "" {
			q.Pod = value
		}
		return true
	case "container", "containername", "container_name":
		if strings.TrimSpace(q.Container) == "" {
			q.Container = value
		}
		return true
	case "trace", "trace_id", "traceid":
		if strings.TrimSpace(q.TraceID) == "" {
			q.TraceID = value
		}
		return true
	case "file", "file_path", "path":
		if strings.TrimSpace(q.FilePath) == "" {
			q.FilePath = value
		}
		return true
	case "collector", "collector_mode", "mode":
		if strings.TrimSpace(q.CollectorMode) == "" {
			q.CollectorMode = value
		}
		return true
	default:
		return false
	}
}
