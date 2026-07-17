package dbmgmt

import (
	"encoding/json"
	"strings"
)

// 标准权限码（对齐常见开源 SQL 审核平台：库/表级 + 细粒度 DML/DDL）。
var standardPrivileges = []string{
	"select", "insert", "update", "delete",
	"create", "alter", "drop", "truncate", "index",
	"create_database", "drop_database",
	"export", "import",
}

func normalizePrivileges(list []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range list {
		p := strings.ToLower(strings.TrimSpace(raw))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func parsePrivilegesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	return normalizePrivileges(arr)
}

func marshalPrivilegesJSON(list []string) string {
	list = normalizePrivileges(list)
	if len(list) == 0 {
		return ""
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func privilegesFromFlags(canQuery, canDML, canDDL, canExport, canImport bool) []string {
	var out []string
	if canQuery {
		out = append(out, "select")
	}
	if canDML {
		out = append(out, "insert", "update", "delete")
	}
	if canDDL {
		out = append(out, "create", "alter", "drop", "truncate", "index")
	}
	if canExport {
		out = append(out, "export")
	}
	if canImport {
		out = append(out, "import")
	}
	return normalizePrivileges(out)
}

func flagsFromPrivileges(list []string) (canQuery, canDML, canDDL, canExport, canImport bool) {
	for _, p := range normalizePrivileges(list) {
		switch p {
		case "select":
			canQuery = true
		case "insert", "update", "delete":
			canDML = true
		case "create", "alter", "drop", "truncate", "index", "create_database", "drop_database":
			canDDL = true
		case "export":
			canExport = true
		case "import":
			canImport = true
		}
	}
	return
}

func isValidDbIdentifier(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
			continue
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func hasPrivilege(list []string, code string) bool {
	for _, p := range normalizePrivileges(list) {
		if p == code {
			return true
		}
	}
	return false
}

func privilegeLabels(list []string) []string {
	labels := map[string]string{
		"select": "SELECT", "insert": "INSERT", "update": "UPDATE", "delete": "DELETE",
		"create": "CREATE", "alter": "ALTER", "drop": "DROP", "truncate": "TRUNCATE", "index": "INDEX",
		"create_database": "CREATE DATABASE", "drop_database": "DROP DATABASE",
		"export": "导出", "import": "导入",
	}
	var out []string
	for _, p := range normalizePrivileges(list) {
		if lbl, ok := labels[p]; ok {
			out = append(out, lbl)
		} else {
			out = append(out, strings.ToUpper(p))
		}
	}
	return out
}
