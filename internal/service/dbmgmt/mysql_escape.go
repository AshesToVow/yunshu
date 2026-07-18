package dbmgmt

import "strings"

func escapeMySQLIdent(s string) string {
	// 反引号加倍；剔除 NUL 与反引号内不应出现的字符。
	s = strings.ReplaceAll(s, "\x00", "")
	return strings.ReplaceAll(s, "`", "``")
}

// escapeMySQLString 转义单引号字符串字面量。
// 注意：MySQL 默认未开启 NO_BACKSLASH_ESCAPES，反斜杠是转义符，
// 必须先转义反斜杠再加倍单引号，否则 `\'` 可突破引号闭合造成注入。
func escapeMySQLString(s string) string {
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, "'", "''")
}
