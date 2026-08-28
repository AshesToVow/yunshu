package alert

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"unicode/utf8"
)

func runeLen(s string) int {
	return len([]rune(s))
}

func clampByRunes(s string, limit int) string {
	if limit <= 0 {
		return s
	}
	rs := []rune(s)
	if len(rs) <= limit {
		return s
	}
	if limit <= 3 {
		return string(rs[:limit])
	}
	return string(rs[:limit-3]) + "..."
}

// clampUTF8ByBytes 截断到不超过 limitBytes，保证不切断 UTF-8 字符边界。
func clampUTF8ByBytes(s string, limitBytes int) string {
	if limitBytes <= 0 || len(s) <= limitBytes {
		return s
	}
	b := []byte(s)
	b = b[:limitBytes]
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b)
}

func splitTextByBytes(s string, limitBytes int) []string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if s == "" {
		return nil
	}
	if limitBytes <= 0 || len(s) <= limitBytes {
		return []string{s}
	}

	parts := strings.Split(s, "\n\n")
	out := make([]string, 0, len(parts))
	var cur strings.Builder
	flush := func() {
		t := strings.TrimSpace(cur.String())
		if t != "" {
			out = append(out, t)
		}
		cur.Reset()
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		try := p
		if cur.Len() > 0 {
			try = cur.String() + "\n\n" + p
		}
		if len(try) <= limitBytes {
			cur.Reset()
			cur.WriteString(try)
			continue
		}
		if cur.Len() > 0 {
			flush()
		}
		if len(p) <= limitBytes {
			cur.WriteString(p)
			continue
		}

		lines := strings.SplitSeq(p, "\n")
		for ln := range lines {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			tryLine := ln
			if cur.Len() > 0 {
				tryLine = cur.String() + "\n" + ln
			}
			if len(tryLine) <= limitBytes {
				cur.Reset()
				cur.WriteString(tryLine)
				continue
			}
			if cur.Len() > 0 {
				flush()
			}
			if len(ln) <= limitBytes {
				cur.WriteString(ln)
				continue
			}

			remain := ln
			for len(remain) > 0 {
				chunk := clampUTF8ByBytes(remain, limitBytes)
				chunk = strings.TrimSpace(chunk)
				if chunk != "" {
					out = append(out, chunk)
				}
				if len(chunk) == 0 {
					break
				}
				remain = strings.TrimSpace(remain[len(chunk):])
			}
		}
	}
	if cur.Len() > 0 {
		flush()
	}
	return out
}

func jsonSizeBytes(v map[string]any) int {
	bs, _ := json.Marshal(v)
	return len(bs)
}

func reduceMarkdownForBytes(text string, limitBytes int) string {
	text = strings.TrimSpace(text)
	if limitBytes <= 0 || len(text) <= limitBytes {
		return text
	}
	if idx := strings.Index(text, "#### 元信息"); idx >= 0 {
		text = strings.TrimSpace(text[:idx])
		if len(text) <= limitBytes {
			return text
		}
	}
	if idx := strings.Index(text, "#### 摘要"); idx >= 0 {
		head := strings.TrimSpace(text[:idx])
		tail := strings.TrimSpace(text[idx:])
		lines := strings.SplitN(tail, "\n", 3)
		if len(lines) >= 2 {
			summary := strings.TrimSpace(strings.Join(lines[1:], "\n"))
			prefix := head
			if prefix != "" {
				prefix += "\n\n"
			}
			prefix += "#### 摘要\n"
			allow := limitBytes - len(prefix)
			if allow > 16 {
				summary = clampUTF8ByBytes(summary, allow)
				text = strings.TrimSpace(prefix + summary)
				if len(text) <= limitBytes {
					return text
				}
			}
		}
	}
	return strings.TrimSpace(clampUTF8ByBytes(text, limitBytes))
}

func splitBodyByJSONLimit(base map[string]any, setText func(dst map[string]any, text string), getText func(src map[string]any) string, limitBytes int) []map[string]any {
	if base == nil {
		return nil
	}
	if limitBytes <= 0 {
		return []map[string]any{base}
	}
	if jsonSizeBytes(base) <= limitBytes {
		return []map[string]any{base}
	}

	overhead := 0
	{
		tmp := map[string]any{}
		maps.Copy(tmp, base)
		setText(tmp, "")
		overhead = jsonSizeBytes(tmp)
	}
	allowTextBytes := limitBytes - overhead
	if allowTextBytes <= 32 {
		return []map[string]any{base}
	}

	origText := strings.TrimSpace(getText(base))
	origText = reduceMarkdownForBytes(origText, allowTextBytes)
	parts := splitTextByBytes(origText, allowTextBytes)
	if len(parts) == 0 {
		return []map[string]any{base}
	}

	out := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		cp := map[string]any{}
		maps.Copy(cp, base)
		setText(cp, p)
		if jsonSizeBytes(cp) > limitBytes {
			p2 := clampUTF8ByBytes(p, maxInt(16, allowTextBytes-64))
			setText(cp, p2)
		}
		out = append(out, cp)
	}
	return out
}

func splitDingTalkBody(body map[string]any, limit int) []map[string]any {
	return splitBodyByJSONLimit(
		body,
		func(dst map[string]any, text string) {
			if mm, ok := dst["markdown"].(map[string]string); ok {
				nm := map[string]string{}
				maps.Copy(nm, mm)
				nm["text"] = text
				dst["markdown"] = nm
				return
			}
			if mm, ok := dst["markdown"].(map[string]any); ok {
				nm := map[string]any{}
				maps.Copy(nm, mm)
				nm["text"] = text
				dst["markdown"] = nm
				return
			}
		},
		func(src map[string]any) string {
			if mm, ok := src["markdown"].(map[string]string); ok {
				return fmt.Sprintf("%v", mm["text"])
			}
			if mm, ok := src["markdown"].(map[string]any); ok {
				return fmt.Sprintf("%v", mm["text"])
			}
			return ""
		},
		limit,
	)
}

func splitWeComBody(body map[string]any, limit int) []map[string]any {
	if body == nil {
		return nil
	}
	msgType := strings.TrimSpace(fmt.Sprintf("%v", body["msgtype"]))
	if msgType == "text" {
		return splitBodyByJSONLimit(
			body,
			func(dst map[string]any, text string) {
				tm, ok := dst["text"].(map[string]any)
				if !ok {
					return
				}
				nt := map[string]any{}
				maps.Copy(nt, tm)
				nt["content"] = text
				dst["text"] = nt
			},
			func(src map[string]any) string {
				if tm, ok := src["text"].(map[string]any); ok {
					return fmt.Sprintf("%v", tm["content"])
				}
				return ""
			},
			limit,
		)
	}

	return splitBodyByJSONLimit(
		body,
		func(dst map[string]any, text string) {
			if mm, ok := dst["markdown"].(map[string]string); ok {
				nm := map[string]string{}
				maps.Copy(nm, mm)
				nm["content"] = text
				dst["markdown"] = nm
				return
			}
			if mm, ok := dst["markdown"].(map[string]any); ok {
				nm := map[string]any{}
				maps.Copy(nm, mm)
				nm["content"] = text
				dst["markdown"] = nm
				return
			}
		},
		func(src map[string]any) string {
			if mm, ok := src["markdown"].(map[string]string); ok {
				return fmt.Sprintf("%v", mm["content"])
			}
			if mm, ok := src["markdown"].(map[string]any); ok {
				return fmt.Sprintf("%v", mm["content"])
			}
			return ""
		},
		limit,
	)
}
