package repository

import "strings"

// escapeLikePattern 转义 LIKE 通配符，需配合 `ESCAPE '!'` 使用（MySQL/SQLite 语义一致）。
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "!", "!!")
	s = strings.ReplaceAll(s, "%", "!%")
	s = strings.ReplaceAll(s, "_", "!_")
	return s
}

func likeContains(s string) string {
	return "%" + escapeLikePattern(strings.TrimSpace(s)) + "%"
}
