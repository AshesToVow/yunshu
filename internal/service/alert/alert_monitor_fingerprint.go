package alert

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// monitorSeriesFingerprint 平台规则按「规则 + 样本标签」生成序列指纹，使同一规则多序列各自独立 pending/firing。
func monitorSeriesFingerprint(ruleID uint, labels map[string]string) string {
	merged := make(map[string]string, len(labels)+1)
	for k, v := range labels {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" || k == "fingerprint" || k == "__name__" {
			continue
		}
		merged[k] = v
	}
	merged["monitor_rule_id"] = fmt.Sprintf("%d", ruleID)

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("rule=%d;", ruleID))
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(merged[k])
		b.WriteByte(';')
	}
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("mr%d_%s", ruleID, hex.EncodeToString(sum[:16]))
}
