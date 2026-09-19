package ai

import (
	"strings"
	"unicode/utf8"
)

// scrubNonBMPForMySQL 将平面外字符（emoji 等 4 字节 UTF-8）替换为 U+FFFD。
// 在表尚未转为 utf8mb4 时，避免 Error 1366；utf8mb4 下仍可安全落库。
func scrubNonBMPForMySQL(s string) string {
	if s == "" || !needsNonBMPScrub(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r > 0xFFFF {
			b.WriteRune('\uFFFD')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func needsNonBMPScrub(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			i++
			continue
		}
		if r > 0xFFFF {
			return true
		}
		i += size
	}
	return false
}
