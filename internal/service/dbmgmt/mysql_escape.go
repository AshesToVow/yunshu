package dbmgmt

import "strings"

func escapeMySQLIdent(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

func escapeMySQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
