package dbmgmt

import (
	"regexp"
	"strings"

	"yunshu/internal/model"
)

var allMySQLPrivNames = []string{
	"SELECT", "INSERT", "UPDATE", "DELETE",
	"CREATE", "ALTER", "INDEX", "DROP",
	"CREATE TEMPORARY TABLES", "SHOW VIEW", "CREATE ROUTINE", "ALTER ROUTINE",
	"EXECUTE", "CREATE VIEW", "EVENT", "TRIGGER",
	"GRANT", "SUPER", "PROCESS", "RELOAD", "SHUTDOWN",
	"SHOW DATABASES", "LOCK TABLES", "REFERENCES",
	"REPLICATION CLIENT", "REPLICATION SLAVE", "CREATE USER",
}

var grantLineRe = regexp.MustCompile(`(?i)^GRANT\s+(.+?)\s+ON\s+(.+?)\s+TO[\s` + "`" + `]`)

func parseMySQLGrantPrivileges(grantLines []string, level, database string) []string {
	targetDB := strings.ToLower(strings.TrimSpace(database))
	isDBLevel := strings.EqualFold(strings.TrimSpace(level), model.DbAppUserPrivDatabase)
	seen := map[string]struct{}{}
	for _, raw := range grantLines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(strings.ToUpper(line), "GRANT USAGE ON") || strings.HasPrefix(strings.ToUpper(line), "GRANT PROXY ON") {
			continue
		}
		m := grantLineRe.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		scope, dbName := parseGrantObject(m[2])
		if !grantAppliesToScope(scope, dbName, isDBLevel, targetDB) {
			continue
		}
		for _, p := range parseGrantPrivPart(m[1]) {
			seen[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for _, p := range allMySQLPrivNames {
		if _, ok := seen[p]; ok {
			out = append(out, p)
		}
	}
	return out
}

func parseGrantObject(onPart string) (scope string, database string) {
	normalized := strings.ReplaceAll(strings.TrimSpace(onPart), "`", "")
	if normalized == "*.*" || normalized == "*" {
		return "global", ""
	}
	if i := strings.Index(normalized, ".*"); i > 0 {
		return "database", strings.ToLower(normalized[:i])
	}
	if i := strings.Index(normalized, "."); i > 0 {
		return "table", strings.ToLower(normalized[:i])
	}
	return "global", ""
}

func grantAppliesToScope(scope, dbName string, isDBLevel bool, targetDB string) bool {
	if !isDBLevel {
		return scope == "global"
	}
	if targetDB == "" {
		return false
	}
	if scope == "global" {
		return true
	}
	return (scope == "database" || scope == "table") && dbName == targetDB
}

func parseGrantPrivPart(part string) []string {
	text := strings.TrimSpace(part)
	if text == "" {
		return nil
	}
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "ALL PRIVILEGES") || strings.Contains(upper, " ALL ") || upper == "ALL" {
		return append([]string(nil), allMySQLPrivNames...)
	}
	remaining := upper
	var found []string
	for _, priv := range sortedMySQLPrivNamesByLen() {
		if strings.Contains(remaining, priv) {
			found = append(found, priv)
			remaining = strings.ReplaceAll(remaining, priv, " ")
		}
	}
	return found
}

func sortedMySQLPrivNamesByLen() []string {
	out := append([]string(nil), allMySQLPrivNames...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if len(out[j]) > len(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
