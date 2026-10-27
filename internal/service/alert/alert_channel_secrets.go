package alert

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"yunshu/internal/model"
)

const maskedChannelSecret = "***"

var sensitiveChannelKeyPattern = regexp.MustCompile(`(?i)(authorization|cookie|token|secret|passw(or)?d|api[-_]?key|access[-_]?key)`)

func isSensitiveChannelKey(key string) bool {
	key = strings.TrimSpace(key)
	return key != "" && sensitiveChannelKeyPattern.MatchString(key)
}

// maskChannelSecrets 只用于响应脱敏；投递链路必须读取原始渠道记录。
func validateChannelOutboundURL(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return assertSafeOutboundURL(raw)
}

func maskChannelSecrets(ch *model.AlertChannel) {
	if ch == nil {
		return
	}
	if strings.TrimSpace(ch.Secret) != "" {
		ch.Secret = maskedChannelSecret
	}
	ch.URL = maskChannelURL(ch.URL)
	ch.HeadersJSON = maskChannelHeadersJSON(ch.HeadersJSON)
}

func maskChannelURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	changed := false
	for key := range q {
		if isSensitiveChannelKey(key) {
			q[key] = []string{maskedChannelSecret}
			changed = true
		}
	}
	if !changed {
		return raw
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func maskChannelHeadersJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return raw
	}
	maskChannelSettings(m)
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return string(out)
}

func maskChannelSettings(settings map[string]any) {
	for key, value := range settings {
		switch v := value.(type) {
		case string:
			if isSensitiveChannelKey(key) && strings.TrimSpace(v) != "" {
				settings[key] = maskedChannelSecret
			}
		case map[string]any:
			for subKey, subValue := range v {
				if s, ok := subValue.(string); ok && isSensitiveChannelKey(subKey) && strings.TrimSpace(s) != "" {
					v[subKey] = maskedChannelSecret
				}
			}
		}
	}
}

func restoreMaskedChannelURL(next, previous string) string {
	next = strings.TrimSpace(next)
	if !strings.Contains(next, maskedChannelSecret) {
		return next
	}
	// 脱敏 URL 可能被前端原样回传（改其他字段时），整体沿用旧地址避免把 *** 写库。
	return previous
}

func restoreMaskedChannelSecret(next, previous string) string {
	next = strings.TrimSpace(next)
	if next == maskedChannelSecret {
		return previous
	}
	return next
}

func restoreMaskedChannelHeaders(next, previous string) string {
	next = strings.TrimSpace(next)
	if !strings.Contains(next, maskedChannelSecret) {
		return next
	}
	var nextMap map[string]any
	if err := json.Unmarshal([]byte(next), &nextMap); err != nil {
		return next
	}
	var prevMap map[string]any
	if strings.TrimSpace(previous) != "" {
		_ = json.Unmarshal([]byte(previous), &prevMap)
	}
	restoreMaskedChannelSettings(nextMap, prevMap)
	out, err := json.Marshal(nextMap)
	if err != nil {
		return next
	}
	return string(out)
}

func restoreMaskedChannelSettings(nextMap, prevMap map[string]any) {
	for key, value := range nextMap {
		switch v := value.(type) {
		case string:
			if v != maskedChannelSecret {
				continue
			}
			if prev, ok := prevMap[key]; ok {
				nextMap[key] = prev
			} else {
				delete(nextMap, key)
			}
		case map[string]any:
			prevSub, _ := prevMap[key].(map[string]any)
			for subKey, subValue := range v {
				s, ok := subValue.(string)
				if !ok || s != maskedChannelSecret {
					continue
				}
				if prev, ok := prevSub[subKey]; ok {
					v[subKey] = prev
				} else {
					delete(v, subKey)
				}
			}
		}
	}
}
